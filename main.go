// Command scaffold generates a Go Domain Driven Design project layout.
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/Allan-Nava/Go-DDD-Scaffold/scaffold"
	"github.com/urfave/cli/v2"
)

// version is set at build time with -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

// The assets are embedded so the binary is self-contained: `go install` can put
// it anywhere, with no repository and no $GOPATH beside it. The "all:" prefix
// keeps dot-files such as template/env/.env.local in the tree.
//
//go:embed all:template
var templateAssets embed.FS

//go:embed all:static
var staticAssets embed.FS

func main() {
	app := &cli.App{
		Name:     "scaffold",
		Version:  version,
		Compiled: time.Now(),
		Usage:    "Generate a Go DDD project layout",
		Commands: []*cli.Command{initCommand()},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "scaffold:", err)
		os.Exit(1)
	}
}

func initCommand() *cli.Command {
	return &cli.Command{
		Name:      "init",
		Aliases:   []string{"i"},
		Usage:     "Generate the project layout in the current directory",
		ArgsUsage: "[dir]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "overwrite files that already exist",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "trace what the generator is doing",
			},
		},
		Action: func(c *cli.Context) error {
			// The destination is where the user is, not where the binary lives.
			dest := c.Args().First()
			if dest == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				dest = cwd
			}

			templates, err := fs.Sub(templateAssets, "template")
			if err != nil {
				return err
			}
			static, err := fs.Sub(staticAssets, "static")
			if err != nil {
				return err
			}

			res, err := scaffold.New(scaffold.Options{
				Templates: templates,
				Static:    static,
				Force:     c.Bool("force"),
				Debug:     c.Bool("debug"),
			}).Generate(dest)
			if err != nil {
				return err
			}

			fmt.Printf("\n%d file(s) created in %s", len(res.Created), dest)
			if n := len(res.Skipped); n > 0 {
				fmt.Printf(", %d skipped (re-run with --force to overwrite)", n)
			}
			fmt.Print("\nNext: go mod tidy && make run\n")
			return nil
		},
	}
}
