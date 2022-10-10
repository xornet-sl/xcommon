package xcommon

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func ExtractDefaults(s interface{}) map[string]interface{} {
	ret := map[string]interface{}{}
	path := []string{}

	_parseStruct(reflect.TypeOf(s), path, ret)
	return ret
}

func _parseStruct(
	sType reflect.Type,
	path []string,
	ret map[string]interface{},
) {
	t := sType
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		ft := f.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}

		if ft.Kind() != reflect.Struct {
			value, ok := f.Tag.Lookup("default")
			if ok {
				path = append(path, strings.ToLower(f.Name))
				convertedValue, err := _convertValue(value, ft)
				if err != nil {
					println("Conversion error: ", err.Error())
				}
				ret[strings.Join(path, ".")] = convertedValue
				path = path[:len(path)-1]
			}
			continue
		}

		path = append(path, strings.ToLower(f.Name))
		_parseStruct(ft, path, ret)
		path = path[:len(path)-1]
	}
}

func _convertValue(value string, ft reflect.Type) (interface{}, error) {
	switch ft.Kind() {
	case reflect.String:
		return value, nil
	case reflect.Bool:
		return strconv.ParseBool(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.ParseInt(value, 10, 64)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.ParseUint(value, 10, 64)
	case reflect.Float32, reflect.Float64:
		return strconv.ParseFloat(value, ft.Bits())
	default:
		return nil, fmt.Errorf("Not implemented conversion type %s", ft.Kind())
	}
}
