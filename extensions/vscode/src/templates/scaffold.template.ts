//
//
export function getScaffoldTemplate(scaffoldName: string): string {
    return `package main
    import (
        "log"
        "os"
        "github.com/gofiber/fiber/v2"
        "github.com/gofiber/fiber/v2/middleware/cors"
        "github.com/gofiber/fiber/v2/middleware/logger"
        "github.com/gofiber/fiber/v2/middleware/recover"
    )
    
    
    func main() {
        // setup fiber stuff
        f := fiber.New()
        f.Use(logger.New())
        f.Use(cors.New(cors.Config{
            AllowOrigins: "*",
            AllowHeaders: "Content-Type, Authorization",
            AllowMethods: "GET, HEAD, OPTIONS, PUT, PATCH, POST, DELETE",
        }))
        f.Use(recover.New())
        // Config
        //config.SetEnvConfig()
        log.Println("fiber ", f)
        //
    
    
        //health check endpoint
        f.Get("/health", func(c *fiber.Ctx) error {
            return c.SendStatus(http.StatusOK)
        })
    
        f.Listen("0.0.0.0:8080")
        //
    }

`;
}