package glutys

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/onfirebyte/glutys/pkg/converter"
)

type ContextParser = func(*http.Request) (any, error)

type Builder struct {
	ContextParsers map[reflect.Type]any
	ContextTypes   map[reflect.Type]struct{}
	GeneratePath   string
	Converter      converter.TSConverter

	tsFile    string
	tsMethods string
	routes    map[string][]any
}

func (g *Builder) AddContextParser(f any) {
	// f must be a function
	if reflect.TypeOf(f).Kind() != reflect.Func {
		panic("Context parser must be a function")
	}

	// f must have one argument which is *http.Request
	fnType := reflect.TypeOf(f)
	argNum := fnType.NumIn()
	if argNum != 1 || fnType.In(0) != reflect.TypeOf(&http.Request{}) {
		panic("Context parser must have one argument which is *http.Request")
	}

	// f must return at least 1-2 values, 2nd of which is optional error
	retNum := fnType.NumOut()
	if retNum < 1 || retNum > 2 {
		panic("Context parser must return at least 1-2 values, 2nd of which is optional error")
	}

	if retNum == 2 {
		// 2nd return value must be error
		if fnType.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
			panic("2nd return value must be error")
		}
	}

	if g.ContextParsers == nil {
		g.ContextParsers = make(map[reflect.Type]any)
	}
	if g.ContextTypes == nil {
		g.ContextTypes = make(map[reflect.Type]struct{})
	}

	retType := fnType.Out(0)

	g.ContextParsers[retType] = f
	g.ContextTypes[retType] = struct{}{}

}

func (g *Builder) CreateRouter(routes map[string][]any) {
	g.routes = routes
}

func (g *Builder) Build() (string, string) {
	goFile := g.buildRouter()

	g.tsFile += fmt.Sprintf("export type GlutysContract = {\n%s}\n", g.tsMethods)

	return goFile, g.tsFile
}
