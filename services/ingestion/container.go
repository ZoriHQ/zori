package ingestion

import (
	"zori/services/ingestion/data"
	"zori/services/ingestion/services"
	"zori/services/ingestion/web"

	"go.uber.org/fx"
)

func BuildIngestionDiContainer() fx.Option {
	return fx.Module("ingestion",
		fx.Provide(data.NewVisitorRepository),
		fx.Provide(services.NewIngestor),
		fx.Provide(services.NewIdentifier),
		fx.Provide(web.NewIngestionServer),
	)
}
