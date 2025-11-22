package web

import (
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/services/funnels/services"
)

func RegisterRoutes(
	s *server.Server,
	funnelService *services.FunnelService,
	authMiddleware middlewares.AuthMiddleware,
) {
	funnelGroup := s.Group("/api/v1/funnels")
	funnelGroup.Use(authMiddleware.Middleware())

	server.GroupPOST(funnelGroup, "", funnelService.CreateFunnel)
	server.GroupGetWithFilter(funnelGroup, "", funnelService.ListFunnels)
	server.GroupGET(funnelGroup, "/:id", funnelService.GetFunnel)
	server.GroupPUT(funnelGroup, "/:id", funnelService.UpdateFunnel)
	server.GroupDELETE(funnelGroup, "/:id", funnelService.DeleteFunnel)
	server.GroupGetWithFilter(funnelGroup, "/:id/analyze", funnelService.AnalyzeFunnel)
}
