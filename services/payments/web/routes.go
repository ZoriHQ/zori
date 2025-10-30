package web

import (
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/services/payments/services"
)

func RegisterPaymentRoutes(
	s *server.Server,
	providerManager *services.ProviderManager,
	connectionService *services.ConnectionService,
	authMiddleware middlewares.AuthMiddleware,
) {
	paymentGroup := s.Group("/api/v1/payment-providers")
	paymentGroup.Use(authMiddleware.Middleware())

	// Provider connection instructions
	server.GroupGET(paymentGroup, "/instructions", connectionService.GetProviderInstructions)

	// CRUD operations
	server.GroupPOST(paymentGroup, "", providerManager.CreateProvider)
	server.GroupGET(paymentGroup, "", providerManager.ListProviders)
	server.GroupGET(paymentGroup, "/:id", providerManager.GetProvider)
	server.GroupPUT(paymentGroup, "/:id", providerManager.UpdateProvider)
	server.GroupDELETE(paymentGroup, "/:id", providerManager.DeleteProvider)

	// Stripe Connect OAuth callback (no auth required - uses state parameter)
	s.Echo.GET("/api/v1/payment-providers/stripe/connect/callback", connectionService.HandleStripeConnectCallback)
}
