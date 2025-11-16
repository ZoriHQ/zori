package web

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"zori/internal/cache"
	"zori/internal/logger"
	"zori/internal/storage/postgres/models"
	"zori/internal/telemetry"
	"zori/services/ingestion/services"
	"zori/services/ingestion/types"
	projectsServices "zori/services/projects/services"

	"github.com/valyala/fasthttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type IngestionServer struct {
	ingestor       *services.Ingestor
	identifier     *services.Identifier
	projectService *projectsServices.ProjectService
	cacheService   *cache.CacheService
	tracerProvider *telemetry.Provider
	tracer         trace.Tracer
	logger         *logger.Logger
}

func NewIngestionServer(
	ingestor *services.Ingestor,
	identifier *services.Identifier,
	projectService *projectsServices.ProjectService,
	cacheService *cache.CacheService,
	tracerProvider *telemetry.Provider,
	log *logger.Logger,
) *IngestionServer {
	var tracer trace.Tracer
	if tracerProvider != nil {
		tracer = tracerProvider.Tracer("zori.ingestion")
	}

	return &IngestionServer{
		ingestor:       ingestor,
		identifier:     identifier,
		projectService: projectService,
		cacheService:   cacheService,
		tracerProvider: tracerProvider,
		tracer:         tracer,
		logger:         log,
	}
}

func (h *IngestionServer) HandleRequest(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
	ctx.Response.Header.SetBytesV("Access-Control-Allow-Origin", []byte("*"))
	ctx.Response.Header.SetBytesV("Access-Control-Allow-Methods", []byte("POST"))
	ctx.Response.Header.SetBytesV("Access-Control-Allow-Headers", []byte("Content-Type, X-Zori-PT, x-zori-version"))
	ctx.Response.Header.SetBytesV("Access-Control-Max-Age", []byte("86400"))

	if ctx.IsOptions() {
		ctx.Response.SetStatusCode(fasthttp.StatusNoContent)
		return
	}

	path := string(ctx.Path())
	switch path {
	case "/ingest":
		h.Injest(ctx)
	case "/identify":
		h.Identify(ctx)
	case "/health":
		ctx.Response.SetStatusCode(fasthttp.StatusOK)
		ctx.Response.SetBodyString("Zori - Ingestion Server")
		break
	default:
		ctx.Error("Not Found", fasthttp.StatusNotFound)
	}
}

func (h *IngestionServer) Injest(ctx *fasthttp.RequestCtx) {
	if !ctx.IsPost() {
		ctx.Error("Bad Request", fasthttp.StatusBadRequest)
		return
	}

	spanCtx, span := h.startSpan(ctx, "ingestion.ingest")
	defer span.End()

	var clientEvent types.ClientEventV1
	if err := json.Unmarshal(ctx.PostBody(), &clientEvent); err != nil {
		telemetry.RecordError(span, err)
		h.logger.WithContext(spanCtx).Error("Failed to decode event payload", "error", err)
		ctx.Error("Failed to decode event payload", fasthttp.StatusBadRequest)
		return
	}

	visitorIDCookieBytes := ctx.Request.Header.Cookie("visitor_id")
	if visitorIDCookieBytes == nil {
		firstTimeVisitorCookie := fasthttp.Cookie{}
		firstTimeVisitorCookie.SetKey("visitor_id")
		firstTimeVisitorCookie.SetValue(clientEvent.VisitorID)
		firstTimeVisitorCookie.SetMaxAge(3600000)
		firstTimeVisitorCookie.SetDomain(".zorihq.com")
		firstTimeVisitorCookie.SetPath(("/"))
		firstTimeVisitorCookie.SetSecure(false)
		ctx.Response.Header.SetCookie(&firstTimeVisitorCookie)
		visitorIDCookieBytes = firstTimeVisitorCookie.Value()
	}

	projectTokenBytes := ctx.Request.Header.Peek("x-zori-pt")
	if projectTokenBytes == nil {
		telemetry.SetStatus(span, fmt.Errorf("missing project token"))
		ctx.Error("X-Zori-PT Missing in the request header", fasthttp.StatusUnauthorized)
		return
	}

	projectToken := string(projectTokenBytes)

	projectFromCache, err := h.cacheService.Get(ctx, cache.ProjectCacheKey.FromValue(projectToken))
	if err != nil {
		telemetry.RecordError(span, err)
		h.logger.WithContext(spanCtx).Warn("Invalid project token", "token", projectToken)
		ctx.Error("Invalid Project Token", fasthttp.StatusUnauthorized)
		return
	}

	var project models.Project
	if projectFromCache == nil {
		projectPointer, err := h.projectService.GetProjectByPublishableToken(projectToken)
		if err != nil {
			telemetry.RecordError(span, err)
			h.logger.WithContext(spanCtx).Warn("Project not found", "token", projectToken)
			ctx.Error("Invalid Project Token", fasthttp.StatusUnauthorized)
			return
		}

		err = h.cacheService.Set(ctx, cache.ProjectCacheKey.FromValue(projectToken), *projectPointer, time.Minute)
		if err != nil {
			telemetry.RecordError(span, err)
			h.logger.WithContext(spanCtx).Error("Failed to cache project", "error", err)
			ctx.Error("Failed to cache project", fasthttp.StatusInternalServerError)
			return
		}

		project = *projectPointer
	} else {
		if err = json.Unmarshal([]byte(*projectFromCache), &project); err != nil {
			telemetry.RecordError(span, err)
			h.logger.WithContext(spanCtx).Error("Failed to unmarshal project", "error", err)
			ctx.Error("Failed to unmarshal project", fasthttp.StatusInternalServerError)
			return
		}
	}

	telemetry.AddIngestionAttributes(span, project.OrganizationID, project.ID, clientEvent.VisitorID, "ingest")

	if project.FirstEventReceivedAt == nil {
		go func() {
			err = h.projectService.SetFirstEventReceivedNow(project.ID)
			if err != nil {
				h.logger.Error("Failed to update project", "error", err, "project_id", project.ID)
			}
		}()
	}

	requestHost := string(ctx.Request.Host())
	if strings.Contains(requestHost, "localhost") && project.AllowLocalHost {
		localhostParts := strings.Split(requestHost, ":")
		if len(localhostParts) == 2 {
			host := localhostParts[0]
			if host != "localhost" {
				ctx.Error("Invalid Host", fasthttp.StatusBadRequest)
				return
			}
		}
	} else if strings.Contains(requestHost, "localhost") && !project.AllowLocalHost {
		ctx.Error("Localhost events are now allowed for the project", fasthttp.StatusBadRequest)
		return
	}

	if clientEvent.VisitorID != string(visitorIDCookieBytes) {
		ctx.Error("Missing or Invalid Visitor ID", fasthttp.StatusBadRequest)
		return
	}

	clientEvent.UserAgent = string(ctx.UserAgent())

	cloudFlareHeaderIP := ctx.Request.Header.Peek("cf-connecting-ip")
	if cloudFlareHeaderIP != nil {
		clientEvent.IP = string(cloudFlareHeaderIP)
	} else if xForwardedForHeader := ctx.Request.Header.Peek(fasthttp.HeaderXForwardedFor); xForwardedForHeader != nil {
		clientEvent.IP = string(xForwardedForHeader)
	} else {
		clientEvent.IP = ctx.RemoteIP().String()
	}

	go h.ingestor.Ingest(&project, &clientEvent)

	telemetry.SetStatus(span, nil)
	h.logger.WithContext(spanCtx).Debug("Event ingested successfully",
		"project_id", project.ID,
		"org_id", project.OrganizationID,
		"visitor_id", clientEvent.VisitorID)

	fmt.Fprintf(ctx, "ACCEPTED %d", len(ctx.PostBody()))
}

func (h *IngestionServer) Identify(ctx *fasthttp.RequestCtx) {
	if !ctx.IsPost() {
		ctx.Error("Bad Request", fasthttp.StatusBadRequest)
		return
	}

	spanCtx, span := h.startSpan(ctx, "ingestion.identify")
	defer span.End()

	var identifyEvent types.IdentifyEventV1
	if err := json.Unmarshal(ctx.PostBody(), &identifyEvent); err != nil {
		telemetry.RecordError(span, err)
		h.logger.WithContext(spanCtx).Error("Failed to decode identify payload", "error", err)
		ctx.Error("Failed to decode identify payload", fasthttp.StatusBadRequest)
		return
	}

	if identifyEvent.VisitorID == "" {
		telemetry.SetStatus(span, fmt.Errorf("missing visitor_id"))
		h.logger.WithContext(spanCtx).Warn("Missing visitor_id in identify request")
		ctx.Error("visitor_id is required", fasthttp.StatusBadRequest)
		return
	}

	visitorIDCookieBytes := ctx.Request.Header.Cookie("visitor_id")
	if visitorIDCookieBytes == nil {
		firstTimeVisitorCookie := fasthttp.Cookie{}
		firstTimeVisitorCookie.SetKey("visitor_id")
		firstTimeVisitorCookie.SetValue(identifyEvent.VisitorID)
		firstTimeVisitorCookie.SetMaxAge(3600000)
		firstTimeVisitorCookie.SetDomain(".zorihq.com")
		firstTimeVisitorCookie.SetPath("/")
		firstTimeVisitorCookie.SetSecure(false)
		ctx.Response.Header.SetCookie(&firstTimeVisitorCookie)
		visitorIDCookieBytes = firstTimeVisitorCookie.Value()
	}

	projectTokenBytes := ctx.Request.Header.Peek("x-zori-pt")
	if projectTokenBytes == nil {
		ctx.Error("X-Zori-PT Missing in the request header", fasthttp.StatusUnauthorized)
		return
	}

	projectToken := string(projectTokenBytes)

	projectFromCache, err := h.cacheService.Get(ctx, cache.ProjectCacheKey.FromValue(projectToken))
	if err != nil {
		ctx.Error("Invalid Project Token", fasthttp.StatusUnauthorized)
		return
	}

	var project models.Project
	if projectFromCache == nil {
		projectPointer, err := h.projectService.GetProjectByPublishableToken(projectToken)
		if err != nil {
			ctx.Error("Invalid Project Token", fasthttp.StatusUnauthorized)
			return
		}

		err = h.cacheService.Set(ctx, cache.ProjectCacheKey.FromValue(projectToken), *projectPointer, time.Minute)
		if err != nil {
			ctx.Error("Failed to cache project", fasthttp.StatusInternalServerError)
			return
		}

		project = *projectPointer
	} else {
		if err = json.Unmarshal([]byte(*projectFromCache), &project); err != nil {
			ctx.Error("Failed to unmarshal project", fasthttp.StatusInternalServerError)
			return
		}
	}

	if identifyEvent.VisitorID != string(visitorIDCookieBytes) {
		ctx.Error("Missing or Invalid Visitor ID", fasthttp.StatusBadRequest)
		return
	}

	identifyEvent.UserAgent = string(ctx.UserAgent())

	cloudFlareHeaderIP := ctx.Request.Header.Peek("cf-connecting-ip")
	if cloudFlareHeaderIP != nil {
		identifyEvent.IP = string(cloudFlareHeaderIP)
	} else if xForwardedForHeader := ctx.Request.Header.Peek(fasthttp.HeaderXForwardedFor); xForwardedForHeader != nil {
		identifyEvent.IP = string(xForwardedForHeader)
	} else {
		identifyEvent.IP = ctx.RemoteIP().String()
	}

	go func() {
		err := h.identifier.Identify(ctx, &project, &identifyEvent)
		if err != nil {
			h.logger.Error("Identify error", "error", err)
			return
		}
	}()

	fmt.Fprintf(ctx, "IDENTIFIED %d", len(ctx.PostBody()))
}

// startSpan creates a new span for FastHTTP requests
func (h *IngestionServer) startSpan(ctx *fasthttp.RequestCtx, name string) (context.Context, trace.Span) {
	if h.tracer == nil {
		return context.Background(), trace.SpanFromContext(context.Background())
	}

	spanCtx, span := h.tracer.Start(context.Background(), name,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			semconv.HTTPRequestMethodKey.String(string(ctx.Method())),
			semconv.URLPath(string(ctx.Path())),
			semconv.URLScheme(string(ctx.URI().Scheme())),
			semconv.ServerAddressKey.String(string(ctx.Host())),
			semconv.UserAgentOriginal(string(ctx.UserAgent())),
			semconv.ClientAddressKey.String(ctx.RemoteIP().String()),
		),
	)

	return spanCtx, span
}
