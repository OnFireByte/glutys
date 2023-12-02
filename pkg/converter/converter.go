package converter

import (
	"reflect"
	"strings"
)

type TSConverter struct {
	CustomTypes map[string]string
}

func (c *TSConverter) ParseType(parent string, obj any) (string, string) {
	// read tags
	// read fields

	v := reflect.ValueOf(obj)
	t := v.Type()

	// if slice, parse slice element
	sliceDepth := 0
	for t.Kind() == reflect.Slice {
		sliceDepth++
		t = t.Elem()
	}

	// if not struct, return empty string
	if t.Kind() != reflect.Struct {
		tsType, _ := c.TypeMap("", t)
		return "", tsType
	}

	childrenType := make([]reflect.Type, 0)

	typeName := parent + t.Name()

	res := "export type " + parent + t.Name() + " = {\n"

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
		r, _ := c.ParseType(typeName, reflect.New(child).Elem().Interface())
		res += r
	}

	return res, typeName + strings.Repeat("[]", sliceDepth)
}

func (c *TSConverter) TypeMap(parent string, typ reflect.Type) (string, *reflect.Type) {
	if name, ok := c.CustomTypes[typ.Name()]; ok {
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
		return parent + typ.Name(), &typ
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
