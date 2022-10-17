package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Allan-Nava/Go-DDD-Scaffold/scaffold"

	"github.com/urfave/cli"
)

//
func main() {
	app := cli.NewApp()
	app.Version = "1.0.1"
	app.Usage = "Generate Scaffold Domain Driven Design project layout for Go."
	app.Commands = []&cli.Command{
		{
			Name:    "init",
			Aliases: []string{"i"},
			Usage:   " Generate scaffold project layout",
			Action: func(c *cli.Context) error {
				currDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
				if err != nil {
					return err
				}
				err = scaffold.New(true).Generate(currDir)
				fmt.Printf("error:%+v currDir %s\n ", err, currDir)
				if err == nil {
					fmt.Println("Success Created. Please excute `make up` to start service.")
				}
				return err
			},
			{
				Name:    "help",
				Aliases: []string{"help"},
				Usage:   " Generate scaffold project layout",
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
