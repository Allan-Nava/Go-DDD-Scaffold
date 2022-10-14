package scaffold

import (
    "os"
)

const (
	GoScaffoldPath = "src/github.com/Allan-Nava/Go-DDD-Scaffold"
)

func init() {
	Gopath = os.Getenv("GOPATH")
	if Gopath == "" {
		panic("cannot find $GOPATH environment variable")
	}
}var Gopath string

type scaffold struct {
	debug bool
}

func New(debug bool) *scaffold {
	return &scaffold{debug: debug}
}