package test

import (
	"fmt"
	"testing"
)

func TestMain(m *testing.M) {
	m.Run()
}

func PrintPanic(expected any, actual any) {
	panic(fmt.Sprintf("\nexpected: \t%s\nactual: \t%s", expected, actual))
}