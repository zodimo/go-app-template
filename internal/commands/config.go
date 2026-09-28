package commands

import (
	"github.com/zodimo/go-app-template/internal/config/commands"
)

var ConfigCmd = commands.RootConfigCmd

func init() {
	ConfigCmd.AddCommand(newVersionCmd())
}
