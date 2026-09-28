package database

import (
	"fmt"

	"github.com/zodimo/go-app-template/internal/core"
	"github.com/zodimo/go-maybe"
)

var _ core.CompliantConfig = (*Config)(nil)

type Config struct {
	configContext   core.ConfigContext             `mapstructure:"-" json:"-"`
	Database        maybe.Maybe[string]            `mapstructure:"database" json:"database"`
	Params          maybe.Maybe[map[string]string] `mapstructure:"params" json:"params"`
	MigrationParams maybe.Maybe[map[string]string] `mapstructure:"migration_params" json:"migration_params"`
}

func (c *Config) DbUrl() string {
	return fmt.Sprintf("sqlite://%s", c.Database.UnwrapUnsafe())
}

func (s *Config) DbUrlForMigration() string {

	merged := make(map[string]string)
	for k, v := range s.Params.UnwrapUnsafe() {
		merged[k] = v
	}
	for k, v := range s.MigrationParams.UnwrapUnsafe() {
		merged[k] = v
	}

	return fmt.Sprintf("%s?%s", s.DbUrl(), mapToQuery(merged))
}

func DefaultParams() map[string]string {
	return map[string]string{
		"journal_mode": "WAL",
		"busy_timeout": "5000",
	}
}

func DefaultMigrationQueries() map[string]string {

	// sqlite://my_database.db?x-migrations-table=custom_table&x-no-tx-wrap=true

	return map[string]string{}
}
