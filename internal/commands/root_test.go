package commands

import (
	"testing"
)

func TestRootCommand_FlagBindings(t *testing.T) {
	flag := RootCmd.PersistentFlags().Lookup("config")
	if flag == nil {
		t.Fatal("--config persistent flag not registered")
	}
	flag = RootCmd.PersistentFlags().Lookup("require-config")
	if flag == nil {
		t.Fatal("--require-config persistent flag not registered")
	}
	flag = RootCmd.PersistentFlags().Lookup("debug")
	if flag == nil {
		t.Fatal("--debug persistent flag not registered")
	}
	flag = RootCmd.PersistentFlags().Lookup("json")
	if flag == nil {
		t.Fatal("--json persistent flag not registered")
	}
	flag = RootCmd.PersistentFlags().Lookup("cwd")
	if flag == nil {
		t.Fatal("--cwd persistent flag not registered")
	}
}
