package main

import (
	"github.com/db-mipt/game-client/internal"
	"github.com/db-mipt/game-client/internal/config"
	"github.com/db-mipt/game-client/internal/httphandler"
	"github.com/db-mipt/game-client/internal/logger"
	"github.com/db-mipt/game-client/internal/repository"
	"go.uber.org/fx"
)

func main() {

	fx.New(
		fx.Provide(logger.NewLogger),
		fx.Provide(httphandler.NewHttpHandler),
		fx.Provide(config.NewAppConfig),
		fx.Provide(repository.NewPlayerRepository),
		fx.Provide(internal.NewAppService),

		fx.Invoke(func(*internal.AppService) {}),
	).Run()

}
