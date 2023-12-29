package glutys

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/onfirebyte/glutys/pkg/converter"
)

type ContextParser = func(*http.Request) (any, error)

type Builder struct {
	// Specifies the path to project root, default is "."
	RootPath string

	contextParsers map[reflect.Type]any
	contextTypes   map[reflect.Type]struct{}
	generatePath   string
	converter      converter.TSConverter

	tsFile       string
	tsMethods    string
	routes       map[string][]any
	funcArgNames map[string][]string
}

func NewBuilder(generatePath string) *Builder {
	return &Builder{
		RootPath:       ".",
		generatePath:   generatePath,
		converter:      converter.TSConverter{},
		routes:         make(map[string][]any),
		funcArgNames:   make(map[string][]string),
		contextParsers: make(map[reflect.Type]any),
		contextTypes:   make(map[reflect.Type]struct{}),
	}
}

func (g *Builder) AddCustomType(t any, name string) *Builder {
	g.converter.CustomTypes[reflect.TypeOf(t)] = name
	return g
}

func (g *Builder) AddContextParser(f any) *Builder {
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

	retType := fnType.Out(0)

	g.contextParsers[retType] = f
	g.contextTypes[retType] = struct{}{}
	return g
}

func (g *Builder) CreateRouter(routes map[string][]any) *Builder {
	g.routes = routes
	return g
}

func (g *Builder) Build() (string, string) {
	fmt.Println("Scanning project...")
	g.scanRPC()

	fmt.Println("Generating router...")
	goFile := g.buildRouter()

	g.tsFile += fmt.Sprintf("export type GlutysContract = {\n%s}\n", g.tsMethods)

	return goFile, g.tsFile
}
