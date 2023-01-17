package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Allan-Nava/Go-DDD-Scaffold/scaffold"
	"github.com/urfave/cli/v2"
)

func main() {
	// 
	app := &cli.App{
		Name:     "scaffold",
		Version:  "v1.0.01",
		Compiled: time.Now(),
		Usage:    "Generate scaffold go-ddd project layout",
		Commands: []*cli.Command{
			{
				Name:    "init",
				Aliases: []string{"i"},
				Usage:   " Generate scaffold project layout",
				Action: func(c *cli.Context) error {
					currDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
					if err != nil {
						log.Println("err filepath.Abs ", err)
						return err
					}
					err = scaffold.New(false).Generate(currDir)
					fmt.Printf("error:%+v currDir %s\n ", err, currDir)
					if err == nil {
						fmt.Println("Success Created. Please excute `make up` to start service.")
					}
					return err
				},
			}, {
				Name:    "help",
				Aliases: []string{"help"},
				Usage:   "Help scaffold project layout",
				Action: func(c *cli.Context) error {
					log.Println("help cli.Context", c)
					return nil
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
