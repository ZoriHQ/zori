package web

import (
	"encoding/json"
	"fmt"
	"strings"
	"zori/services/ingestion/services"
	"zori/services/ingestion/types"
	projectsServices "zori/services/projects/services"

	"github.com/valyala/fasthttp"
)

type IngestionServer struct {
	ingestor       *services.Ingestor
	projectService *projectsServices.ProjectService
}

func NewIngestionServer(ingestor *services.Ingestor, projectService *projectsServices.ProjectService) *IngestionServer {
	return &IngestionServer{
		ingestor:       ingestor,
		projectService: projectService,
	}
}

func (h *IngestionServer) Injest(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
	ctx.Response.Header.SetBytesV("Access-Control-Allow-Origin", []byte("*"))
	ctx.Response.Header.SetBytesV("Access-Control-Allow-Methods", []byte("POST"))
	ctx.Response.Header.SetBytesV("Access-Control-Allow-Headers", []byte("Content-Type, X-Zori-PT, x-zori-version"))
	ctx.Response.Header.SetBytesV("Access-Control-Max-Age", []byte("86400"))

	if string(ctx.Path()) != "/ingest" {
		ctx.Error("Not Found", fasthttp.StatusNotFound)
		return
	}

	if ctx.IsOptions() {
		ctx.Response.SetStatusCode(fasthttp.StatusNoContent)
		return
	}

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

	project, err := h.projectService.GetProjectByPublishableToken(projectToken)
	if err != nil {
		ctx.Error("Invalid Project Token", fasthttp.StatusUnauthorized)
		return
	}

	fmt.Println("Host of the request origin", string(ctx.Request.Host()))

	// checking for localhost events
	requestHost := string(ctx.Request.Host())
	if strings.Contains(requestHost, "localhost") && project.AllowLocalHost {
		localhostParts := strings.Split(requestHost, ":")
		if len(localhostParts) == 2 {
			host := localhostParts[1]
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

	fmt.Println("Ingested....")

	go h.ingestor.Ingest(project, &clientEvent)

	fmt.Fprintf(ctx, "ACCEPTED %d", len(ctx.PostBody()))
}
