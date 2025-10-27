package web

import (
	"fmt"
	"net/http"
	"zori/internal/server"
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

func RegisterRoutes(s *server.Server, projectService *projectsService.ProjectService, jwtService *services.JWTService, eventsService *services.EventsService) {
	s.Echo.GET("/events/stream", func(c echo.Context) error {
		ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		defer ws.Close()

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
			ws.WriteMessage(websocket.TextMessage, msg.Data)
		})
		if err != nil {
			return err
		}
		defer subscription.Unsubscribe()

		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				break
			}
		}

		return nil
	})
}
