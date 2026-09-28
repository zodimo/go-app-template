package config

import (
	"reflect"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/zodimo/go-maybe"
)

// hookTarget mirrors the shape of the real config structs: plain fields plus
// maybe.Maybe[T] fields of every element kind used in the application.
type hookTarget struct {
	Directory maybe.Maybe[string]
	MaxSize   maybe.Maybe[int]
	Compress  maybe.Maybe[bool]
	Params    maybe.Maybe[map[string]string]
	Tags      maybe.Maybe[[]string]
	Missing   maybe.Maybe[string]
	Debug     bool
	Working   string
}

// decodeWithHook decodes input through a mapstructure decoder configured the
// same way viper configures it (WeaklyTypedInput + composed default hooks),
// plus the maybe decode hook.
func decodeWithHook(t *testing.T, input any) *hookTarget {
	t.Helper()
	out := &hookTarget{}
	cfg := &mapstructure.DecoderConfig{
		Result:           out,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			maybeDecodeHook,
		),
	}
	d, err := mapstructure.NewDecoder(cfg)
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	if err := d.Decode(input); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func TestMaybeDecodeHook_ConfigFileScalars(t *testing.T) {
	// Values as viper presents them after reading a JSON config file:
	// numbers arrive as float64, nested objects as map[string]any, arrays as []any.
	got := decodeWithHook(t, map[string]any{
		"Directory": "/tmp/x",
		"MaxSize":   float64(50),
		"Compress":  true,
		"Params":    map[string]any{"busy_timeout": "5000", "journal_mode": "WAL"},
		"Tags":      []any{"one", "two"},
		"Debug":     true,
		"Working":   "/tmp",
	})

	if v, _ := got.Directory.Unwrap(); v != "/tmp/x" {
		t.Errorf("Directory = %v, want Some(/tmp/x)", got.Directory)
	}
	if v, _ := got.MaxSize.Unwrap(); v != 50 {
		t.Errorf("MaxSize = %v, want Some(50)", got.MaxSize)
	}
	if v, _ := got.Compress.Unwrap(); v != true {
		t.Errorf("Compress = %v, want Some(true)", got.Compress)
	}
	m, _ := got.Params.Unwrap()
	if m["busy_timeout"] != "5000" || m["journal_mode"] != "WAL" {
		t.Errorf("Params = %v, want decoded map entries", m)
	}
	tags, _ := got.Tags.Unwrap()
	if len(tags) != 2 || tags[0] != "one" || tags[1] != "two" {
		t.Errorf("Tags = %v, want [one two]", tags)
	}
	if got.Debug != true || got.Working != "/tmp" {
		t.Errorf("plain fields decoded unexpectedly: Debug=%v Working=%q", got.Debug, got.Working)
	}
}

func TestMaybeDecodeHook_EnvStringsWeakTyping(t *testing.T) {
	// Environment variables always arrive as strings; the hook must weakly
	// convert them to the element type ("50" -> int, "true" -> bool).
	got := decodeWithHook(t, map[string]any{
		"Directory": "test-database",
		"MaxSize":   "50",
		"Compress":  "true",
	})

	if v, _ := got.Directory.Unwrap(); v != "test-database" {
		t.Errorf("Directory = %v, want Some(test-database)", got.Directory)
	}
	if v, _ := got.MaxSize.Unwrap(); v != 50 {
		t.Errorf("MaxSize = %v, want Some(50)", got.MaxSize)
	}
	if v, _ := got.Compress.Unwrap(); v != true {
		t.Errorf("Compress = %v, want Some(true)", got.Compress)
	}
}

func TestMaybeDecodeHook_MissingKeyIsNone(t *testing.T) {
	got := decodeWithHook(t, map[string]any{})
	if !got.Missing.IsNone() {
		t.Errorf("Missing = %v, want None", got.Missing)
	}
	if !got.MaxSize.IsNone() {
		t.Errorf("MaxSize = %v, want None", got.MaxSize)
	}
}

func TestMaybeDecodeHook_AlreadyMaybePassthrough(t *testing.T) {
	// Viper defaults are registered as already-constructed Maybe[T] values.
	got := decodeWithHook(t, map[string]any{
		"Directory": maybe.Some("default-dir"),
		"MaxSize":   maybe.Some(3),
	})
	if v, _ := got.Directory.Unwrap(); v != "default-dir" {
		t.Errorf("Directory = %v, want Some(default-dir) passthrough", got.Directory)
	}
	if v, _ := got.MaxSize.Unwrap(); v != 3 {
		t.Errorf("MaxSize = %v, want Some(3) passthrough", got.MaxSize)
	}
}

func TestMaybeDecodeHook_NilSourceIsNone(t *testing.T) {
	got := decodeWithHook(t, map[string]any{
		"Directory": nil,
		"MaxSize":   nil,
	})
	if !got.Directory.IsNone() {
		t.Errorf("Directory = %v, want None", got.Directory)
	}
	if !got.MaxSize.IsNone() {
		t.Errorf("MaxSize = %v, want None", got.MaxSize)
	}
}

func TestMaybeDecodeHook_NonMaybeTargetsPassThrough(t *testing.T) {
	got := decodeWithHook(t, map[string]any{
		"Debug":   "true", // weak-typed plain field, must NOT be intercepted
		"Working": "/work",
	})
	if got.Debug != true {
		t.Errorf("Debug = %v, want true (weak-typed by viper defaults)", got.Debug)
	}
	if got.Working != "/work" {
		t.Errorf("Working = %q, want /work", got.Working)
	}
}

func TestMaybeDecodeHook_ElementTypeFromValueField(t *testing.T) {
	// The element type is recovered from the Maybe[T] layout; guard that the
	// first field is still named "value" so a go-maybe layout change fails loudly.
	to := reflect.TypeOf(maybe.Maybe[string]{})
	if to.Field(0).Name != "value" {
		t.Fatalf("go-maybe layout changed: first field is %q, want \"value\"", to.Field(0).Name)
	}
}
