package logging

import (
	"github.com/zodimo/go-app-template/internal/core"
	"github.com/zodimo/go-maybe"
)

var _ core.CompliantConfig = (*LogConfig)(nil)

type LogConfig struct {
	configContext core.ConfigContext  `mapstructure:"-" json:"-"`
	Directory     maybe.Maybe[string] `mapstructure:"directory" json:"directory"`
	Filename      maybe.Maybe[string] `mapstructure:"filename" json:"filename"`
	MaxSize       maybe.Maybe[int]    `mapstructure:"max_size" json:"max_size"`
	MaxBackups    maybe.Maybe[int]    `mapstructure:"max_backups" json:"max_backups"`
	MaxAge        maybe.Maybe[int]    `mapstructure:"max_age" json:"max_age"`
	Compress      maybe.Maybe[bool]   `mapstructure:"compress" json:"compress"`
	Debug         maybe.Maybe[bool]   `mapstructure:"debug" json:"debug"`
}

func NewLogConfig(
	configContext core.ConfigContext,
	directory string,
	filename string,
	maxSize int,
	maxBackups int,
	maxAge int,
	compress bool,
	debug bool,
) LogConfig {
	return LogConfig{
		Directory:  maybe.Some(directory),
		Filename:   maybe.Some(filename),
		MaxSize:    maybe.Some(maxSize),
		MaxBackups: maybe.Some(maxBackups),
		MaxAge:     maybe.Some(maxAge),
		Compress:   maybe.Some(compress),
		Debug:      maybe.Some(debug),
	}
}


