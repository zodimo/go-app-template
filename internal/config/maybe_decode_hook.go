package config

import (
	jsonv2 "encoding/json/v2"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-viper/mapstructure/v2"
)

// maybePkgPath is the import path of the go-maybe library whose Maybe[T] type
// this hook knows how to construct.
//
// Gotcha: the type MUST be matched by package path, not by type name. For a
// generic instantiation, reflect.Type.Name() returns "Maybe[string]" (type
// argument embedded), not "Maybe" — a name equality check silently misses
// every field and the original "expected a map or struct" error comes back.
const maybePkgPath = "github.com/zodimo/go-maybe"

// maybeDecodeHook is a mapstructure.DecodeHookFuncType that decodes plain
// config values into maybe.Maybe[T]-typed struct fields during viper.Unmarshal.
//
// go-maybe's Maybe[T] keeps both of its fields (value, hasValue) unexported
// and implements only the encoding/json/v2 marshal/unmarshal interfaces
// (jsontext.MarshalerTo / UnmarshalerFrom). mapstructure therefore treats it
// as an opaque struct and cannot fill it from a scalar. This hook bridges the
// gap in two steps:
//
//  1. Weak-convert the source value to the element type T with a plain nested
//     mapstructure decoder (WeaklyTypedInput: true, no hook). This reuses
//     mapstructure's own weak typing, so environment strings like "50" become
//     int and "true" become bool, and maps/slices decode element-wise.
//
//  2. Round-trip the now-exactly-typed value through encoding/json/v2
//     (jsonv2.Marshal -> jsonv2.Unmarshal into a fresh Maybe[T]). This
//     invokes go-maybe's UnmarshalJSONFrom and constructs Some(T) without
//     needing reflection to set unexported fields.
//
// Building a Maybe through encoding/json/v2 requires GOEXPERIMENT=jsonv2. This
// is not an additional constraint: go-maybe itself imports encoding/json/v2,
// so the module stops compiling without the experiment flag anyway.
//
// Sources that are already Maybe[T] (viper defaults) and nil sources are
// passed through untouched, so they decode to their zero value (None).
func maybeDecodeHook(from reflect.Type, to reflect.Type, data any) (any, error) {
	// Only handle Maybe[T] targets, matched by package path (see maybePkgPath).
	if to.PkgPath() != maybePkgPath || !strings.HasPrefix(to.Name(), "Maybe[") {
		return data, nil
	}
	// An already-constructed Maybe[T] (e.g. a viper default) needs no work.
	if from == to {
		return data, nil
	}
	// nil (JSON null, or DecodeNil) means None: returning nil leaves the field
	// untouched, and the zero value of Maybe[T] is None.
	if data == nil {
		return nil, nil
	}
	rv := reflect.ValueOf(data)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface, reflect.Chan, reflect.Func:
		if rv.IsNil() {
			return nil, nil
		}
	}

	// Recover the element type T from the Maybe[T] layout. Fail loudly if
	// go-maybe ever renames or reorders its fields.
	if to.NumField() == 0 || to.Field(0).Name != "value" {
		return nil, fmt.Errorf("maybe decode hook: unexpected Maybe layout for %s", to)
	}
	elemType := to.Field(0).Type

	// 1. Weak-convert the source to the element type with a plain decoder.
	elemOut := reflect.New(elemType)
	inner, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           elemOut.Interface(),
		WeaklyTypedInput: true,
	})
	if err != nil {
		return nil, fmt.Errorf("maybe decode hook: create decoder for %s: %w", elemType, err)
	}
	if err := inner.Decode(data); err != nil {
		return nil, fmt.Errorf("maybe decode hook: cannot decode %v (%T) into %s: %w", data, data, elemType, err)
	}

	// 2. Round-trip through encoding/json/v2 to construct Some(T).
	b, err := jsonv2.Marshal(elemOut.Elem().Interface())
	if err != nil {
		return nil, fmt.Errorf("maybe decode hook: marshal %s: %w", elemType, err)
	}
	out := reflect.New(to)
	if err := jsonv2.Unmarshal(b, out.Interface()); err != nil {
		return nil, fmt.Errorf("maybe decode hook: unmarshal into %s: %w", to, err)
	}
	return out.Elem().Interface(), nil
}
