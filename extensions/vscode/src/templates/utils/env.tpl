package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func SetupEnv() {
	if os.Getenv("APP_ENV") == "local" {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}
}