package main

import (
	"github.com/urfave/cli"
)

func main(){
	app := cli.NewApp()
	app.Version = "1.0.0-rc"
	app.Usage = "Generate Scaffold Domain Driven Design project layout for Go."

}