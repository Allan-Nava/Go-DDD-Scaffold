package config

import (
	"fmt"
	"log"

	env "github.com/caarlos0/env/v11"
)

var CONFIGURATION *Configuration

type Configuration struct {
	//
	AppEnv      string `env:"APP_ENV" envDefault:"local"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"debug"`
	RunningMode string `env:"RUNNING_MODE" envDefault:"main"` //fallback or main
	HTTPAddr    string `env:"HTTP_ADDR" envDefault:"0.0.0.0:8080"`
	DBName      string `env:"DB_NAME"`
	DBUsername  string `env:"DB_USERNAME"`
	DBPassword  string `env:"DB_PASSWORD"`
	DBHost      string `env:"DB_HOST"`
	DBPort      string `env:"DB_PORT" envDefault:"3306"`
	DBIdleConn  int    `env:"DB_IDLE_CONN" envDefault:"10"`
	DBMaxConn   int    `env:"DB_MAX_CONN" envDefault:"100"`
	//
}

func SetEnvConfig() error {
	cfg := Configuration{}
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("parse environment: %w", err)
	}
	log.Println("load configuration OK")
	CONFIGURATION = &cfg
	return nil
}
