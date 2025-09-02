package configparser

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"
)

// Parse fills in the struct from environment variables and default values
func Parse(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("expected pointer to struct")
	}
	return parseStruct(rv.Elem())
}

func parseStruct(rv reflect.Value) error {
	rt := rv.Type()

	for i := range rv.NumField() {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		envTag := fieldType.Tag.Get("env")
		defaultTag := fieldType.Tag.Get("default")

		// If it's a nested struct and doesn't have `env`, parse recursively
		if field.Kind() == reflect.Struct && envTag == "" {
			if err := parseStruct(field); err != nil {
				return err
			}
			continue
		}

		if envTag == "" {
			continue
		}

		val, ok := os.LookupEnv(envTag)

		if !ok || val == "" {
			val = defaultTag // fallback to default if no env var
		}
		if val == "" {
			continue // still empty? Skip.
		}

		if !field.CanSet() {
			return fmt.Errorf("cannot set field %s", fieldType.Name)
		}

		if fieldType.Type == reflect.TypeOf(time.Duration(0)) {
			dur, err := time.ParseDuration(val)
			if err != nil {
				return fmt.Errorf("failed to parse duration for %s: %v", envTag, err)
			}
			field.Set(reflect.ValueOf(dur))
			continue
		}

		switch field.Kind() {
		case reflect.String:
			field.SetString(val)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse int for %s: %v", envTag, err)
			}
			field.SetInt(n)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse uint for %s: %v", envTag, err)
			}
			field.SetUint(n)
		case reflect.Bool:
			b, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("failed to parse bool for %s: %v", envTag, err)
			}
			field.SetBool(b)
		case reflect.Float32, reflect.Float64:
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return fmt.Errorf("failed to parse float for %s: %v", envTag, err)
			}
			field.SetFloat(f)
		default:
			return fmt.Errorf("unsupported kind %s for field %s", field.Kind(), fieldType.Name)
		}
	}

	return nil
}
