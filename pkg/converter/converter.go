package converter

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/onfirebyte/glutys/pkg/util"
)

type nestType string

const (
	pointer nestType = "pointer"
	slice   nestType = "slice"
)

type TSConverter struct {
	CustomTypes map[reflect.Type]string

	createdTypes map[string]struct{}
}

func (c *TSConverter) ParseType(parent string, t reflect.Type) (string, string) {
	// if slice, parse slice element
	nestedDepth := []nestType{}
	for {
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
			nestedDepth = append(nestedDepth, pointer)
		} else if t.Kind() == reflect.Slice {
			t = t.Elem()
			nestedDepth = append(nestedDepth, slice)
		} else {
			break
		}
	}
	slices.Reverse(nestedDepth)

	// if not struct, return empty string
	if t.Kind() != reflect.Struct {
		tsType, _ := c.TypeMap("", t)

		for _, nest := range nestedDepth {
			switch nest {
			case pointer:
				tsType = fmt.Sprintf("(%s | null)", tsType)
			case slice:
				tsType = fmt.Sprintf("%s[]", tsType)
			}
		}
		return "", tsType
	}

	childrenType := make([]reflect.Type, 0)

	typeName := ""
	if parent != "" {
		typeName = parent + t.Name()
	} else {
		typeName = packageNameFromPkgPath(t.PkgPath()) + t.Name()
	}

	if c.createdTypes == nil {
		c.createdTypes = make(map[string]struct{})
	}

	typeRes := typeName

	for _, nest := range nestedDepth {
		switch nest {
		case pointer:
			typeRes = fmt.Sprintf("(%s | null)", typeRes)
		case slice:
			typeRes = fmt.Sprintf("%s[]", typeRes)
		}
	}

	if _, ok := c.createdTypes[typeName]; ok {
		return "", typeRes
	}

	c.createdTypes[typeName] = struct{}{}

	res := "export type " + typeName + " = {\n"

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name := field.Name
		tag := field.Tag.Get("json")
		tagName := strings.Split(tag, ",")[0]
		typ := field.Type
		typeName, t := c.TypeMap(t.Name(), typ)

		if t != nil {
			childrenType = append(childrenType, *t)
		}

		if tag != "" && tagName != "" {
			res += "\t" + tagName + ": " + typeName + "\n"
		} else {
			res += "\t" + name + ": " + typeName + "\n"
		}
	}

	res += "}\n\n"

	for _, child := range childrenType {
		fmt.Println("child type", child.Kind(), t)
		if child.Kind() == reflect.Struct {
			continue
		}
		r, _ := c.ParseType(typeName, child)
		res += r
	}

	return res, typeRes
}

func (c *TSConverter) TypeMap(parent string, typ reflect.Type) (string, *reflect.Type) {
	if name, ok := c.CustomTypes[typ]; ok {
		return name, nil
	}

	switch typ.Kind() {
	case reflect.String:
		return "string", nil
	case reflect.Int:
		return "number", nil
	case reflect.Int8:
		return "number", nil
	case reflect.Int16:
		return "number", nil
	case reflect.Int32:
		return "number", nil
	case reflect.Int64:
		return "number", nil
	case reflect.Float32:
		return "number", nil
	case reflect.Float64:
		return "number", nil
	case reflect.Bool:
		return "boolean", nil
	case reflect.Struct:
		typeName := ""
		if parent != "" {
			typeName = parent + typ.Name()
		} else {
			typeName = packageNameFromPkgPath(typ.PkgPath()) + typ.Name()
		}
		return typeName, &typ
	case reflect.Ptr:
		res, t := c.TypeMap(parent, typ.Elem())
		return res + " | null", t
	case reflect.Slice:
		res, t := c.TypeMap(parent, typ.Elem())
		return res + "[]", t
	default:
		return "any", nil
	}
}

func ConvertStruct(structName string) string {
	return ""
}

func packageNameFromPkgPath(pkgPath string) string {
	splitted := strings.Split(pkgPath, "/")
	name := splitted[len(splitted)-1]

	return util.CamelCaseToPublic(name)
}
