package main

import (
	"fmt"
	"reflect"
)

func i2s(data interface{}, out interface{}) error {
	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Pointer {
		return fmt.Errorf("must be pointer type")
	} else {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		x, ok := data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("cant unpack map")
		}

		for i := 0; i < v.NumField(); i++ {
			fn := v.Type().Field(i).Name
			fv, ok := x[fn]
			if !ok {
				return fmt.Errorf("field %s missed", fn)
			}

			err := i2s(fv, v.Field(i).Addr().Interface())
			if err != nil {
				return fmt.Errorf("error at field %s: %s", fn, err.Error())
			}
		}

	case reflect.Slice:
		x, ok := data.([]interface{})
		if !ok {
			return fmt.Errorf("cant unpack slice")
		}

		for n, i := range x {
			sv := reflect.New(v.Type().Elem())
			err := i2s(i, sv.Interface())
			if err != nil {
				return fmt.Errorf("error at emel %d: %s", n, err.Error())
			}
			v.Set(reflect.Append(v, sv.Elem()))
		}

	case reflect.String:
		x, ok := data.(string)
		if !ok {
			return fmt.Errorf("cant unpack string")
		}
		v.SetString(x)

	case reflect.Int:
		x, ok := data.(float64)
		if !ok {
			return fmt.Errorf("cant unpack int")
		}
		v.SetInt(int64(x))

	case reflect.Bool:
		x, ok := data.(bool)
		if !ok {
			return fmt.Errorf("cant unpack bool")
		}
		v.SetBool(x)

	default:
		return fmt.Errorf("unsupported type %s", v.Kind().String())
	}

	return nil
}
