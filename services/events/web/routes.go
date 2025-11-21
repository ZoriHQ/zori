package web

import (
	"fmt"
	"net/http"
	"zori/internal/server"
	"zori/internal/telemetry"
	"zori/services/events/services"
	projectsService "zori/services/projects/services"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func RegisterRoutes(s *server.Server, projectService *projectsService.ProjectService, jwtService *services.JWTService, eventsService *services.EventsService, logger *telemetry.Logger) {
	s.Echo.GET("/events/stream", func(c echo.Context) error {
		ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		defer func() {
			if err := ws.Close(); err != nil {
				logger.Error("Failed to close websocket", telemetry.Error(err))
			}
		}()

		id := c.QueryParam("id")
		token := c.QueryParam("token")

		if id == "" || token == "" {
			return fmt.Errorf("missing id or token")
		}

		validatedToken, err := jwtService.ValidateAccessToken(token)
		if err != nil {
			return err
		}

		project, err := projectService.GetProjectNoContext(c.Request().Context(), id, validatedToken.OrganizationID)
		if err != nil {
			return err
		}

		subscription, err := eventsService.SubscribeOnEventsStream(project.ID, func(msg *nats.Msg) {
			if err := ws.WriteMessage(websocket.TextMessage, msg.Data); err != nil {
				logger.Error("Failed to write message to websocket", telemetry.Error(err), telemetry.String("project_id", project.ID))
			}
		})
		if err != nil {
			return err
		}
		defer func() {
			if err := subscription.Unsubscribe(); err != nil {
				logger.Error("Failed to unsubscribe from events stream", telemetry.Error(err), telemetry.String("project_id", project.ID))
			}
		}()

		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				break
			}
		}

		return nil
	})
}
