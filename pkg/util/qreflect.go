package util

import (
	"reflect"
)

// AttrToUnderscore 获取struct的所有属性并转为下划线模式
func AttrToUnderscore(st interface{}) []string {
	t := reflect.ValueOf(st)
	vType := t.Elem().Type()
	names := make([]string, 0)
	for i := 0; i < vType.NumField(); i++ {
		if vType.Field(i).Type.Kind() == reflect.Struct {
			continue
		}
		name := vType.Field(i).Name
		if name != "" {
			names = append(names, UnderscoreName(name))
		}
	}
	return names
}
