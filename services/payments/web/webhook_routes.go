package web

import (
	"zori/internal/server"
	"zori/services/payments/services"
)

func RegisterWebhookRoutes(server *server.Server, webhookHandler *services.WebhookHandler) {
	webhookGroup := server.Echo.Group("/webhooks")
	// Stripe App lifecycle webhook (handles app installation/deauthorization)
	webhookGroup.POST("/stripe/app/lifecycle", webhookHandler.HandleStripeAppLifecycleWebhook)
	// Stripe App payment webhook for cloud version
	webhookGroup.POST("/stripe/cloud/app", webhookHandler.HandleStripeAppWebhook)
	// Stripe webhook for self-hosted version (per project)
	webhookGroup.POST("/stripe/:project_id", webhookHandler.HandleStripeWebhook)
}
