package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"zori/internal/cache"
	"zori/internal/storage/postgres/models"
	"zori/internal/telemetry"
	projectsServices "zori/services/projects/services"
	"zori/services/traces/services"
	"zori/services/traces/types"

	"github.com/valyala/fasthttp"
)

type TracesHandler struct {
	ingestionService *services.IngestionService
	projectService   *projectsServices.ProjectService
	cacheService     *cache.CacheService
	logger           *telemetry.Logger
}

func NewTracesHandler(
	ingestionService *services.IngestionService,
	projectService *projectsServices.ProjectService,
	cacheService *cache.CacheService,
	logger *telemetry.Logger,
) *TracesHandler {
	return &TracesHandler{
		ingestionService: ingestionService,
		projectService:   projectService,
		cacheService:     cacheService,
		logger:           logger,
	}
}

func (h *TracesHandler) setCORSHeaders(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
	ctx.Response.Header.Set("Content-Type", "application/json")
}

func (h *TracesHandler) HandleGetProjects(ctx *fasthttp.RequestCtx) {
	h.setCORSHeaders(ctx)

	if ctx.IsOptions() {
		ctx.Response.SetStatusCode(fasthttp.StatusNoContent)
		return
	}

	if !ctx.IsGet() {
		h.sendError(ctx, fasthttp.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	project, err := h.authenticateRequest(ctx)
	if err != nil {
		h.sendError(ctx, fasthttp.StatusUnauthorized, "Invalid or missing API keys")
		return
	}

	response := ProjectResponse{
		Data: []ProjectData{
			{
				ID:   project.ID,
				Name: project.Name,
			},
		},
	}

	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	responseBytes, _ := json.Marshal(response)
	ctx.Response.SetBody(responseBytes)
}

type ProjectResponse struct {
	Data []ProjectData `json:"data"`
}

type ProjectData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *TracesHandler) HandleIngestion(ctx *fasthttp.RequestCtx) {
	h.setCORSHeaders(ctx)

	if ctx.IsOptions() {
		ctx.Response.SetStatusCode(fasthttp.StatusNoContent)
		return
	}

	if !ctx.IsPost() {
		h.sendError(ctx, fasthttp.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	project, err := h.authenticateRequest(ctx)
	if err != nil {
		h.sendError(ctx, fasthttp.StatusUnauthorized, "Invalid or missing API keys")
		return
	}

	var req types.IngestionRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		h.sendError(ctx, fasthttp.StatusBadRequest, "Invalid JSON body")
		return
	}

	if len(req.Batch) == 0 {
		h.sendError(ctx, fasthttp.StatusBadRequest, "Batch cannot be empty")
		return
	}

	response := h.ingestionService.ProcessBatch(context.Background(), project.ID, req.Batch)

	ctx.Response.SetStatusCode(fasthttp.StatusMultiStatus)
	responseBytes, _ := json.Marshal(response)
	ctx.Response.SetBody(responseBytes)
}

func (h *TracesHandler) HandlePublicIngestion(ctx *fasthttp.RequestCtx) {
	h.HandleIngestion(ctx)
}

func (h *TracesHandler) HandleOtelTraces(ctx *fasthttp.RequestCtx) {
	h.setCORSHeaders(ctx)

	if ctx.IsOptions() {
		ctx.Response.SetStatusCode(fasthttp.StatusNoContent)
		return
	}

	if !ctx.IsPost() {
		h.sendError(ctx, fasthttp.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	project, err := h.authenticateRequest(ctx)
	if err != nil {
		h.sendError(ctx, fasthttp.StatusUnauthorized, "Invalid or missing API keys")
		return
	}

	rawBody := ctx.PostBody()
	var req types.OtelTraceRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		h.logger.Error("Failed to parse OTEL request",
			telemetry.Error(err),
			telemetry.String("project_id", project.ID),
			telemetry.String("raw_body", string(rawBody)))
		h.sendError(ctx, fasthttp.StatusBadRequest, "Invalid JSON body")
		return
	}

	if len(req.ResourceSpans) == 0 {
		h.sendError(ctx, fasthttp.StatusBadRequest, "resourceSpans cannot be empty")
		return
	}

	if err := h.ingestionService.ProcessOtelTraces(context.Background(), project.ID, &req); err != nil {
		h.logger.Error("Failed to process OTEL traces",
			telemetry.Error(err),
			telemetry.String("project_id", project.ID),
			telemetry.String("raw_body", string(rawBody)))
	}

	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	ctx.Response.SetBody([]byte("{}"))
}

func (h *TracesHandler) authenticateRequest(ctx *fasthttp.RequestCtx) (*models.Project, error) {
	authHeader := string(ctx.Request.Header.Peek("Authorization"))
	if authHeader == "" {
		return nil, &AuthError{Message: "Missing Authorization header"}
	}

	if !strings.HasPrefix(authHeader, "Basic ") {
		return nil, &AuthError{Message: "Invalid Authorization header format"}
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
	if err != nil {
		return nil, &AuthError{Message: "Invalid base64 encoding"}
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return nil, &AuthError{Message: "Invalid credentials format"}
	}

	publicKey := parts[0]
	secretKey := parts[1]

	cacheKey := cache.LangfuseProjectCacheKey.FromValue(publicKey)
	cachedProject, err := h.cacheService.Get(ctx, cacheKey)
	if err == nil && cachedProject != nil {
		var project models.Project
		if err := json.Unmarshal([]byte(*cachedProject), &project); err == nil {
			if project.LangfuseSecretKey == nil {
				return nil, &AuthError{Message: "Project does not have Langfuse keys configured"}
			}
			if secretKey != *project.LangfuseSecretKey {
				return nil, &AuthError{Message: "Invalid secret key"}
			}
			return &project, nil
		}
	}

	project, err := h.projectService.GetProjectByLangfusePublicKey(context.Background(), publicKey)
	if err != nil {
		return nil, &AuthError{Message: "Invalid public key"}
	}

	if project.LangfuseSecretKey == nil {
		return nil, &AuthError{Message: "Project does not have Langfuse keys configured"}
	}

	if secretKey != *project.LangfuseSecretKey {
		return nil, &AuthError{Message: "Invalid secret key"}
	}

	if err := h.cacheService.Set(ctx, cacheKey, project, time.Minute); err != nil {
		h.logger.Error("Failed to cache Langfuse project", telemetry.Error(err))
	}

	return project, nil
}

func (h *TracesHandler) sendError(ctx *fasthttp.RequestCtx, status int, message string) {
	ctx.Response.SetStatusCode(status)
	ctx.Response.Header.Set("Content-Type", "application/json")
	response := map[string]string{"error": message}
	responseBytes, _ := json.Marshal(response)
	ctx.Response.SetBody(responseBytes)
}

type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}
