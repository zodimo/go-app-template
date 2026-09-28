package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type Store struct {
	connection *sql.DB
	config     *Config
}

func NewStore(ctx context.Context, config *Config) (*Store, error) {

	if !config.IsValid() {
		return nil, fmt.Errorf("config has errors")
	}

	s := &Store{
		config: config,
	}

	dir := filepath.Dir(config.Database.UnwrapUnsafe())
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("NewStore: failed to create directory: %w", err)
		}
	}

	err := migrateNow(s)
	if err != nil {
		return nil, fmt.Errorf("NewStore: migrateNow: %w", err)
	}

	db, err := sql.Open("sqlite", s.dataSourceName())
	if err != nil {
		return nil, fmt.Errorf("Could not open the datasource: %s, %w", s.dataSourceName(), err)
	}

	db.SetMaxOpenConns(10)

	s.connection = db

	// cleanup
	go func() {
		<-ctx.Done()
		db.Close()
	}()

	return s, nil
}

func (s *Store) Raw() *sql.DB {
	return s.connection
}

func (s *Store) DbUrl() string {
	return s.config.DbUrl()
}

func (s *Store) DbUrlForMigration() string {
	return s.config.DbUrlForMigration()
}

func (s *Store) dataSourceName() string {
	return fmt.Sprintf("%s?%s", s.config.Database, mapToQuery(s.config.Params.UnwrapUnsafe()))
}

func mapToQuery(args map[string]string) string {
	// pieces := []string{}
	// for k, v := range args {
	// 	pieces = append(pieces, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(v)))
	// }
	// return strings.Join(pieces, "&")

	pragmas := []string{}
	otherParams := []string{}

	for k, v := range args {
		// Map common pragmas to the modernc.org/sqlite _pragma=name(value) format
		if k == "journal_mode" || k == "busy_timeout" || k == "foreign_keys" || k == "synchronous" {
			pragmas = append(pragmas, fmt.Sprintf("_pragma=%s(%s)", k, v))
		} else {
			otherParams = append(otherParams, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(v)))
		}
	}

	pieces := append(pragmas, otherParams...)
	return strings.Join(pieces, "&")
}
