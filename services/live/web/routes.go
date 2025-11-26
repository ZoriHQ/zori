package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"zori/internal/server"
	"zori/internal/telemetry"
	eventsServices "zori/services/events/services"
	liveServices "zori/services/live/services"
	"zori/services/live/types"
	projectsServices "zori/services/projects/services"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func RegisterRoutes(
	s *server.Server,
	projectService *projectsServices.ProjectService,
	jwtService *eventsServices.JWTService,
	liveTracker *liveServices.LiveSessionTracker,
	logger *telemetry.Logger,
) {
	s.Echo.GET("/live/visitors", func(c echo.Context) error {
		ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		defer func() {
			if err := ws.Close(); err != nil {
				logger.Error("Failed to close websocket", telemetry.Error(err))
			}
		}()

		projectID := c.QueryParam("id")
		token := c.QueryParam("token")

		if projectID == "" || token == "" {
			return fmt.Errorf("missing id or token")
		}

		validatedToken, err := jwtService.ValidateAccessToken(token)
		if err != nil {
			return err
		}

		project, err := projectService.GetProjectNoContext(c.Request().Context(), projectID, validatedToken.OrganizationID)
		if err != nil {
			return err
		}

		initialCount, err := liveTracker.GetActiveCount(context.Background(), project.ID)
		if err != nil {
			logger.Error("Failed to get initial live count",
				telemetry.Error(err),
				telemetry.String("project_id", project.ID),
			)
		} else {
			initialMessage := types.LiveVisitorCount{
				ProjectID: project.ID,
				Count:     initialCount,
			}
			if data, err := json.Marshal(initialMessage); err == nil {
				if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
					logger.Error("Failed to write initial count to websocket",
						telemetry.Error(err),
						telemetry.String("project_id", project.ID),
					)
				}
			}
		}

		unsubscribe, err := liveTracker.SubscribeToUpdates(project.ID, func(count *types.LiveVisitorCount) {
			data, err := json.Marshal(count)
			if err != nil {
				logger.Error("Failed to marshal live count",
					telemetry.Error(err),
					telemetry.String("project_id", project.ID),
				)
				return
			}
			if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
				logger.Error("Failed to write message to websocket",
					telemetry.Error(err),
					telemetry.String("project_id", project.ID),
				)
			}
		})
		if err != nil {
			return err
		}
		defer func() {
			if err := unsubscribe(); err != nil {
				logger.Error("Failed to unsubscribe from live updates",
					telemetry.Error(err),
					telemetry.String("project_id", project.ID),
				)
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
