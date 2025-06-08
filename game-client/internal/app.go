package internal

import (
	"context"
	"fmt"
	"github.com/db-mipt/game-client/internal/config"
	"github.com/db-mipt/game-client/internal/httphandler"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
	"time"
)

type AppService struct {
	cfg config.HTTP
	log *logrus.Entry
	e   *echo.Echo

	httpHandler *httphandler.HttpHandler
}

func NewAppService(
	lc fx.Lifecycle,
	cfg *config.Config,
	log *logrus.Entry,
	httphandler *httphandler.HttpHandler,
) *AppService {

	srv := &AppService{
		cfg:         cfg.HTTP,
		e:           echo.New(),
		log:         log,
		httpHandler: httphandler,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := srv.Start(ctx); err != nil {
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}

func (svc *AppService) Start(ctx context.Context) error {
	svc.e.Server.IdleTimeout = 15 * time.Second
	svc.registerRoutes()

	svc.log.Info("Starting HTTP server")
	go func() {
		svc.log.Fatal(svc.e.Start(fmt.Sprintf("%v:%v", svc.cfg.BindAddress, svc.cfg.BindPort)))
	}()

	return nil
}

func (svc *AppService) Shutdown(ctx context.Context) error {
	return svc.e.Shutdown(ctx)
}

func (svc *AppService) registerRoutes() {
	svc.e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccessControlAllowOrigin,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
			echo.HeaderContentDisposition,
			echo.HeaderIfModifiedSince,
			echo.HeaderLastModified,
			"Pragma",
			"Cache-Control"},
		AllowCredentials: true,
	}))

	assets := svc.e.Group("/leaderboard")

	assets.POST("/update", svc.httpHandler.UpdateScore)
	assets.GET("/get-player", svc.httpHandler.GetPlayerRank)
	assets.GET("/get-board", svc.httpHandler.GetLeaderboard)
}
