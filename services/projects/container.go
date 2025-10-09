package projects

import (
	"zori/services/projects/data"
	"zori/services/projects/services"
	"zori/services/projects/web"

	"go.uber.org/fx"
)

func BuildProjectsDIContainer() fx.Option {
	return fx.Module("projects",
		fx.Provide(
			data.NewProjectData,
			services.NewProjectService,
		),
	)
}

func BuildProjectWebDIContainer() fx.Option {
	return fx.Module("project_web",
		fx.Invoke(web.RegisterRoutes),
	)
}
