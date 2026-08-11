package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// setup fiber stuff
	f := fiber.New()
	// recover first, so a panic in any later middleware is still caught
	f.Use(recover.New())
	f.Use(logger.New())
	f.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Content-Type, Authorization",
		AllowMethods: "GET, HEAD, OPTIONS, PUT, PATCH, POST, DELETE",
	}))
	// Config
	//config.SetEnvConfig()

	// health check endpoint
	f.Get("/health", func(c *fiber.Ctx) error {
		// fiber.StatusOK, so net/http does not have to be imported
		return c.SendStatus(fiber.StatusOK)
	})

	log.Fatal(f.Listen("0.0.0.0:8080"))
}
