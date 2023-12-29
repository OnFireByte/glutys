package util

import (
	"reflect"
	"strings"

	"github.com/dave/jennifer/jen"
)

func CamelCaseToPublic(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
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
