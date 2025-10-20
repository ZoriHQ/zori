package web

import (
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/services/payments/services"
)

func RegisterPaymentRoutes(
	s *server.Server,
	providerManager *services.ProviderManager,
	jwtMiddleware *middlewares.JwtMiddleware,
) {
	// Payment provider management routes (authenticated)
	paymentGroup := s.Group("/api/v1/payment-providers")
	paymentGroup.Use(jwtMiddleware.Middleware())

	// Create a new payment provider
	server.GroupPOST(paymentGroup, "", providerManager.CreateProvider)

	// List payment providers
	server.GroupGET(paymentGroup, "", providerManager.ListProviders)

	// Get a payment provider
	server.GroupGET(paymentGroup, "/:id", providerManager.GetProvider)

	// Update a payment provider
	server.GroupPUT(paymentGroup, "/:id", providerManager.UpdateProvider)

	// Delete a payment provider
	server.GroupDELETE(paymentGroup, "/:id", providerManager.DeleteProvider)
}
