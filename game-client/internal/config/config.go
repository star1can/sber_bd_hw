package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
)

type (
	Config struct {
		App
		HTTP
		REDIS
	}

	App struct {
		AppName    string `env:"APP_NAME" env-default:"leader-board-app"`
		AppVersion string `env:"APP_VERSION" env-default:"1.0"`
		LogLevel   string `env:"LOG_LEVEL" env-default:"debug"`
	}

	HTTP struct {
		BindAddress string `env:"BIND_ADDRESS" env-default:"0.0.0.0"`
		BindPort    uint   `env:"BIND_PORT" env-default:"8085"`
	}

	REDIS struct {
		ClusterAddress     string `env:"REDIS_CLUSTER_ADDRESS" env-default:"redis-node-1:6379,redis-node-2:6379,redis-node-3:6379"`
		Password           string `env:"REDIS_CLUSTER_PASSWORD" env-default:""`
		LeaderBoardKeyBase string `env:"REDIS_CLUSTER_DB" env-default:"arena_legends_leaderboard"`
	}
)

func NewAppConfig() *Config {
	cfg := &Config{}

	err := cleanenv.ReadEnv(cfg)
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	return cfg
}
