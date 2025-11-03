package web

import (
	"zori/internal/server"
	"zori/services/payments/services"
)

func RegisterWebhookRoutes(server *server.Server, webhookHandler *services.WebhookHandler) {
	webhookGroup := server.Echo.Group("/webhooks")
	webhookGroup.POST("/stripe/app/lifecycle", webhookHandler.HandleStripeAppLifecycleWebhook)
	webhookGroup.POST("/stripe/cloud/app", webhookHandler.HandleStripeAppWebhook)
	webhookGroup.POST("/stripe/:project_id", webhookHandler.HandleStripeWebhook)
}
