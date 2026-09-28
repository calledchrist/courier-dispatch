package ddd

import "reflect"

func HasTooManyProps(value any, max int) bool {
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Struct {
		return val.NumField() > max
	}

	return false
}

func HasTooFewProps(value any, min int) bool {
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Struct {
		return val.NumField() < min
	}

	return false
}

func IsObject(v any) bool {
	kind := reflect.TypeOf(v).Kind()

	return kind == reflect.Struct || kind == reflect.Map
}
