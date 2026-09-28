package app

import (
	"fmt"

	"github.com/zodimo/go-app-template/internal/exitcode"
)

// Replace with your app

type App struct{}

func (a *App) Run() exitcode.ExitStatus {
	fmt.Println("Hello world")
	return exitcode.ExitStatusOK
}
