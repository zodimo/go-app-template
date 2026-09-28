package commands

import (
	"github.com/zodimo/go-app-template/internal/database/commands"
)

var DatabaseCmd = commands.RootConfigCmd

func init() {
	DatabaseCmd.AddCommand(newVersionCmd())
}
