package web

import (
	"fmt"
	"zori/services/ingestion/services"

	"github.com/valyala/fasthttp"
)

type IngestionServer struct {
	ingestor *services.Ingestor
}

func NewIngestionServer(ingestor *services.Ingestor) *IngestionServer {
	return &IngestionServer{
		ingestor: ingestor,
	}
}

func (h *IngestionServer) Injest(ctx *fasthttp.RequestCtx) {
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
		ctx.Error("Bad Request", fasthttp.StatusBadRequest)
		return
	}

	projectTokenBytes := ctx.Request.Header.Peek("x-zori-pt")
	if projectTokenBytes == nil {
		ctx.Error("X-Zori-PT Missing in the request header", fasthttp.StatusUnauthorized)
		return
	}

	projectToken := string(projectTokenBytes)

	fmt.Fprintf(ctx, "ACCEPTED %d", len(ctx.PostBody()))
}
