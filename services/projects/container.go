package projects

import (
	"marker/services/projects/data"
	"marker/services/projects/services"
	"marker/services/projects/web"

	"go.uber.org/fx"
)

func BuildProjectsDIContainer() fx.Option {
	return fx.Module("projects",
		fx.Provide(
			data.NewProjectData,
			services.NewProjectService,
		),
		fx.Invoke(web.RegisterRoutes),
	)
}
