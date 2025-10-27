package web

import (
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/services/payments/services"
)

func RegisterPaymentRoutes(
	s *server.Server,
	providerManager *services.ProviderManager,
	stackAuthMiddleware *middlewares.StackAuthMiddleware,
) {
	paymentGroup := s.Group("/api/v1/payment-providers")
	paymentGroup.Use(stackAuthMiddleware.Middleware())

	server.GroupPOST(paymentGroup, "", providerManager.CreateProvider)
	server.GroupGET(paymentGroup, "", providerManager.ListProviders)
	server.GroupGET(paymentGroup, "/:id", providerManager.GetProvider)
	server.GroupPUT(paymentGroup, "/:id", providerManager.UpdateProvider)

	server.GroupDELETE(paymentGroup, "/:id", providerManager.DeleteProvider)
}
