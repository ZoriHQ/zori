package web

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"zori/internal/cache"
	"zori/internal/storage/postgres/models"
	"zori/services/ingestion/services"
	"zori/services/ingestion/types"
	projectsServices "zori/services/projects/services"

	"github.com/valyala/fasthttp"
)

type IngestionServer struct {
	ingestor       *services.Ingestor
	identifier     *services.Identifier
	projectService *projectsServices.ProjectService
	cacheService   *cache.CacheService
}

func NewIngestionServer(ingestor *services.Ingestor, identifier *services.Identifier, projectService *projectsServices.ProjectService, cacheService *cache.CacheService) *IngestionServer {
	return &IngestionServer{
		ingestor:       ingestor,
		identifier:     identifier,
		projectService: projectService,
		cacheService:   cacheService,
	}
}

// HandleRequest is the main router for ingestion endpoints
func (h *IngestionServer) HandleRequest(ctx *fasthttp.RequestCtx) {
	// Set CORS headers for all requests
	ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
	ctx.Response.Header.SetBytesV("Access-Control-Allow-Origin", []byte("*"))
	ctx.Response.Header.SetBytesV("Access-Control-Allow-Methods", []byte("POST"))
	ctx.Response.Header.SetBytesV("Access-Control-Allow-Headers", []byte("Content-Type, X-Zori-PT, x-zori-version"))
	ctx.Response.Header.SetBytesV("Access-Control-Max-Age", []byte("86400"))

	// Handle OPTIONS preflight
	if ctx.IsOptions() {
		ctx.Response.SetStatusCode(fasthttp.StatusNoContent)
		return
	}

	// Route to appropriate handler
	path := string(ctx.Path())
	switch path {
	case "/ingest":
		h.Injest(ctx)
	case "/identify":
		h.Identify(ctx)
	default:
		ctx.Error("Not Found", fasthttp.StatusNotFound)
	}
}

func (h *IngestionServer) Injest(ctx *fasthttp.RequestCtx) {
	if !ctx.IsPost() {
		ctx.Error("Bad Request", fasthttp.StatusBadRequest)
		return
	}

	var clientEvent types.ClientEventV1
	if err := json.Unmarshal(ctx.PostBody(), &clientEvent); err != nil {
		ctx.Error("Failed to decode event payload", fasthttp.StatusBadRequest)
		return
	}

	visitorIDCookieBytes := ctx.Request.Header.Cookie("visitor_id")
	if visitorIDCookieBytes == nil {
		// if visitor id is not present in cookies, we assume this is the first time the user is visiting the site
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

	if project.FirstEventReceivedAt == nil {
		go func() {
			err = h.projectService.SetFirstEventReceivedNow(project.ID)
			if err != nil {
				ctx.Error("Failed to update project", fasthttp.StatusInternalServerError)
			}
		}()
	}

	// checking for localhost events
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

	// trying to extract user IP
	cloudFlareHeaderIP := ctx.Request.Header.Peek("cf-connecting-ip")
	if cloudFlareHeaderIP != nil {
		clientEvent.IP = string(cloudFlareHeaderIP)
	} else if xForwardedForHeader := ctx.Request.Header.Peek(fasthttp.HeaderXForwardedFor); xForwardedForHeader != nil {
		clientEvent.IP = string(xForwardedForHeader)
	} else {
		clientEvent.IP = ctx.RemoteIP().String()
	}

	go h.ingestor.Ingest(&project, &clientEvent)

	fmt.Fprintf(ctx, "ACCEPTED %d", len(ctx.PostBody()))
}

func (h *IngestionServer) Identify(ctx *fasthttp.RequestCtx) {
	if !ctx.IsPost() {
		ctx.Error("Bad Request", fasthttp.StatusBadRequest)
		return
	}

	var identifyEvent types.IdentifyEventV1
	if err := json.Unmarshal(ctx.PostBody(), &identifyEvent); err != nil {
		ctx.Error("Failed to decode identify payload", fasthttp.StatusBadRequest)
		return
	}

	// Validate required fields
	if identifyEvent.VisitorID == "" {
		ctx.Error("visitor_id is required", fasthttp.StatusBadRequest)
		return
	}

	visitorIDCookieBytes := ctx.Request.Header.Cookie("visitor_id")
	if visitorIDCookieBytes == nil {
		// Set visitor_id cookie if not present
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

	// Extract user IP
	cloudFlareHeaderIP := ctx.Request.Header.Peek("cf-connecting-ip")
	if cloudFlareHeaderIP != nil {
		identifyEvent.IP = string(cloudFlareHeaderIP)
	} else if xForwardedForHeader := ctx.Request.Header.Peek(fasthttp.HeaderXForwardedFor); xForwardedForHeader != nil {
		identifyEvent.IP = string(xForwardedForHeader)
	} else {
		identifyEvent.IP = ctx.RemoteIP().String()
	}

	// Process identify event
	go h.identifier.Identify(ctx, &project, &identifyEvent)

	fmt.Fprintf(ctx, "IDENTIFIED %d", len(ctx.PostBody()))
}
