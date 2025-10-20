package web

import (
	"zori/services/payments/services"

	"github.com/labstack/echo/v4"
)

func RegisterWebhookRoutes(e *echo.Echo, webhookHandler *services.WebhookHandler) {
	webhookGroup := e.Group("/webhooks")
	webhookGroup.POST("/stripe/:project_id", webhookHandler.HandleStripeWebhook)
}
