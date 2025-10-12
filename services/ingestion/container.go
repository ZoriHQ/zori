package ingestion

import (
	"zori/services/ingestion/services"
	"zori/services/ingestion/web"

	"go.uber.org/fx"
)

func BuildIngestionDiContainer() fx.Option {
	return fx.Module("ingestion",
		fx.Provide(services.NewIngestor),
		fx.Provide(web.NewIngestionServer),
	)
}
