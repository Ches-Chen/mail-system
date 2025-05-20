//go:build wireinject
// +build wireinject

package injector

import (
	"context"
	"github.com/google/wire"
	"msps/internal/app/api"
	"msps/internal/app/controller"
	"msps/internal/app/router"
)

func BuildInjector(ctx context.Context) (*Injector, func(), error) {
	wire.Build(
		wire.Struct(new(Injector), "*"),
		initHttpServer,
		router.ProviderSet,
		api.ProviderSet,
		controller.ProviderSet,
	)
	return &Injector{}, nil, nil
}
