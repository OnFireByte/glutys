package util

import (
	"reflect"
	"runtime"
	"strings"

	"github.com/dave/jennifer/jen"
)

func CamelCaseToPublic(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func PublicToCamelCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(string(s[0])) + s[1:]
}

func GetJenType(t reflect.Type) *jen.Statement {
	res := jen.Op("")
	for {
		if t.Kind() == reflect.Ptr {
			res = res.Op("*")
			t = t.Elem()
		} else if t.Kind() == reflect.Slice {
			res = res.Index()
			t = t.Elem()
		} else {
			break
		}
	}

	if t.PkgPath() == "" {
		res = res.Id(t.Name())
	} else {
		res = res.Qual(t.PkgPath(), t.Name())
	}
	return res
}

func GetFunctionName(i any) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	nameSplit := strings.Split(fullName, "/")
	return strings.TrimSpace(nameSplit[len(nameSplit)-1])
}
