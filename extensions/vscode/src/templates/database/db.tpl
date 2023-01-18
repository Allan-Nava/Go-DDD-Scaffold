package config

import (
	"fmt"

	"github.com/caarlos0/env/v6"
    "log"
)

var CONFIGURATION *Configuration

type Configuration struct {
	//
	AppEnv      string `env:"APP_ENV"`
	LogLevel    string `env:"LOG_LEVEL"`
	RunningMode string `env:"RUNNING_MODE"` //fallback or main
	DBName		string `env:"DB_NAME"`
	DBUsername	string `env:"DB_USERNAME"`
	DBPassword	string `env:"DB_PASSWORD"`
	DBHost		string `env:"DB_HOST"`
	DBPort		string `env:"DB_PORT"`
	//
}


func SetEnvConfig() {
	cfg := Configuration{}
	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("%+v\n", err)
	}
	logrus.Info("\nload configuration OK")
	CONFIGURATION = &cfg
}