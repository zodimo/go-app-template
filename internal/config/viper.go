package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/zodimo/go-app-template/internal/identity"
)

var keyReplacer = strings.NewReplacer(".", "_", "-", "_")

type ConfigSearchPaths struct {
	cwd   string
	paths []string
}

func NewConfigSearchPaths(cwd string, paths ...string) *ConfigSearchPaths {
	return &ConfigSearchPaths{
		cwd:   cwd,
		paths: paths,
	}
}

func (c *ConfigSearchPaths) GetSearchPaths() []string {
	return append([]string{c.cwd}, c.paths...)
}

func (c *ConfigSearchPaths) PrintSearchPaths() {
	fmt.Println("Search Paths:")
	fmt.Printf("\t%s [dynamic: current working directory]\n", c.cwd)
	for _, path := range c.paths {
		fmt.Printf("\t%s\n", path)
	}
}

func GetConfigSearchPaths(cwd string, paths []string) *ConfigSearchPaths {
	return NewConfigSearchPaths(cwd, paths...)
}

// configureViper sets up viper's configuration paths and environment variables.
func configureViper(workingDir string) {
	viper.SetConfigName(ConfigName())

	configSearchPaths := GetConfigSearchPaths(workingDir, defaultSearchPaths)
	for _, path := range configSearchPaths.GetSearchPaths() {
		viper.AddConfigPath(os.ExpandEnv(path))
	}
	viper.SetEnvPrefix(strings.ToUpper(identity.EnvPrefix))
	viper.SetEnvKeyReplacer(keyReplacer)
	viper.AutomaticEnv()
}

func handleConfigPath(configPath string) error {
	if configPath == "" {
		return nil
	}
	abs, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("failed to determine absolute path for %q: %w", configPath, err)
	}
	viper.SetConfigFile(abs)
	return nil
}

func UpdateCfgFile(update func(*Config)) error {
	if cfg == nil {
		return fmt.Errorf("cannot update config file: config not loaded")
	}
	path := viper.ConfigFileUsed()
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory for default config path: %w", err)
		}
		path = filepath.Join(home, fmt.Sprintf(".%s.json", identity.AppName))
	}

	var raw map[string]json.RawMessage
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			raw = make(map[string]json.RawMessage)
		} else {
			return fmt.Errorf("failed to read config file at %q: %w", path, err)
		}
	} else {
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("failed to parse config file at %q: %w", path, err)
		}
	}

	var userCfg Config
	if known, ok := raw["working_dir"]; ok {
		_ = json.Unmarshal(known, &userCfg.WorkingDir)
	}
	if known, ok := raw["data"]; ok {
		_ = json.Unmarshal(known, &userCfg.Data)
	}
	if known, ok := raw["log"]; ok {
		_ = json.Unmarshal(known, &userCfg.Log)
	}
	if known, ok := raw["debug"]; ok {
		_ = json.Unmarshal(known, &userCfg.Debug)
	}

	update(&userCfg)

	raw["working_dir"], _ = json.Marshal(userCfg.WorkingDir)
	raw["data"], _ = json.Marshal(userCfg.Data)
	raw["log"], _ = json.Marshal(userCfg.Log)
	raw["debug"], _ = json.Marshal(userCfg.Debug)

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated config to JSON: %w", err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("failed to write updated config to %q: %w", path, err)
	}
	return nil
}
