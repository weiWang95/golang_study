package bson

import (
	"reflect"
	"strings"

	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gogf/gf/v2/util/gutil"
)

func OmitEmpty(data any) map[string]any {
	value := reflect.ValueOf(data)
	if value.Type().Kind() == reflect.Ptr {
		value = value.Elem()
	}
	result := make(map[string]any)

	switch value.Type().Kind() {
	case reflect.Map:
		for _, k := range value.MapKeys() {
			v := value.MapIndex(k)
			if v.IsZero() || gutil.IsEmpty(v.Interface()) {
				continue
			}

			result[gconv.String(k.Interface())] = v.Interface()
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			v := value.Field(i)
			if v.IsZero() {
				continue
			}

			if gutil.IsEmpty(v.Interface()) {
				continue
			}

			field := value.Type().Field(i)
			name := field.Name

			if tag := field.Tag.Get("bson"); tag != "" {
				tagName := strings.Split(tag, ",")[0]
				if tagName != "-" {
					name = tagName
				}
			}

			result[name] = v.Interface()
		}
	default:
		panic("type not support")
	}

	return result
}
