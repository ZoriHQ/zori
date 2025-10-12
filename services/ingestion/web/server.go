package web

import (
	"encoding/json"
	"fmt"
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
	ctx.Response.Header.SetBytesV("Access-Control-Allow-Origin", ctx.Request.Header.Peek("Origin"))

	if string(ctx.Path()) != "/ingest" {
		ctx.Error("Not Found", fasthttp.StatusNotFound)
		return
	}

	if !ctx.IsPost() {
		ctx.Error("Bad Request", fasthttp.StatusBadRequest)
		return
	}

	visitorIDCookieBytes := ctx.Request.Header.Cookie("visitor_id")
	if visitorIDCookieBytes == nil {
		ctx.Error("Bad Request - No Visitor ID", fasthttp.StatusBadRequest)
		return
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

	var clientEvent types.ClientEventV1
	if err = json.Unmarshal(ctx.PostBody(), &clientEvent); err != nil {
		ctx.Error("Failed to decode event payload", fasthttp.StatusBadRequest)
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

	go h.ingestor.Ingest(project, &clientEvent)

	fmt.Fprintf(ctx, "ACCEPTED %d", len(ctx.PostBody()))
}
