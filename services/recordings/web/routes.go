package web

import (
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/services/recordings/services"
)

func RegisterRoutes(
	s *server.Server,
	recordingsService *services.RecordingsService,
	authMiddleware middlewares.AuthMiddleware,
) {
	recordingsGroup := s.Group("/api/v1/recordings")
	recordingsGroup.Use(authMiddleware.Middleware())

	server.GroupGetWithFilter(recordingsGroup, "", recordingsService.GetRecordings)
	server.GroupGetWithFilter(recordingsGroup, "/:session_id", recordingsService.GetRecordingEvents)
}
