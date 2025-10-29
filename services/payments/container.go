package payments

import (
	"context"
	"zori/internal/config"
	"zori/internal/utils"
	"zori/services/payments/data"
	"zori/services/payments/services"
	"zori/services/payments/web"

	"go.uber.org/fx"
)

func BuildPaymentsDIContainer() fx.Option {
	return fx.Module("payments",
		fx.Provide(data.NewPaymentProviderData),
		fx.Provide(services.NewBackfillService),
		fx.Provide(services.NewProviderManager),
		fx.Provide(services.NewConnectionService),
		fx.Provide(services.NewWebhookHandler),
		fx.Provide(services.NewPaymentProcessor),
		fx.Provide(func(cfg *config.Config) *utils.Encryptor {
			return utils.NewEncryptor(cfg.EncryptionKey)
		}),
	)
}

func BuildPaymentsWebDIContainer() fx.Option {
	return fx.Invoke(web.RegisterPaymentRoutes)
}

func BuildPaymentsProcessorDIContainer() fx.Option {
	return fx.Invoke(func(lc fx.Lifecycle, processor *services.PaymentProcessor) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				go processor.Start()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				return processor.Stop()
			},
		})
	})
}

func BuildPaymentsWebhookDIContainer() fx.Option {
	return fx.Invoke(web.RegisterWebhookRoutes)
}
