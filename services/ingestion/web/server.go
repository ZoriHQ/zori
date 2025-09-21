package web

import (
	"fmt"

	"github.com/valyala/fasthttp"
)

type IngestionServer struct {
}

func NewIngestionServer() *IngestionServer {
	return &IngestionServer{}
}

func (h *IngestionServer) Inject(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, "ACCEPTED %d", len(ctx.PostBody()))
}
