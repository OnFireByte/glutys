package glutys

import (
	"fmt"
	"net/http"
	"os"
	"reflect"

	"github.com/onfirebyte/glutys/pkg/converter"
)

type ContextParser = func(*http.Request) (any, error)

type Builder struct {
	// Specifies the path to project root, default is "."
	RootPath string

	// used to store and finding context parser function
	contextParsers map[reflect.Type]any
	// used to store and finding context types
	contextTypes map[reflect.Type]struct{}

	// used to finding dependency types
	diTypeMap map[reflect.Type]struct{}
	// used preserve order of dependency types
	diTypes []reflect.Type

	packagePath string
	converter   converter.TSConverter

	tsFile       string
	tsMethods    string
	routes       map[string][]any
	funcArgNames map[string][]string

	tsFilePath string
	goFilePath string
}

// NewBuilder creates a new instance of the Builder struct.
//
//   - pagePath is the go package path of the generated file.
//
//   - goFilePath is the relative path to the generated go file.
//
//   - tsFilePath is the relative path to the generated typescript file.
//
// The RootPath is set to ".", you can change it by setting the RootPath field.
func NewBuilder(packagePath string, goFilePath string, tsFilePath string) *Builder {
	return &Builder{
		RootPath:       ".",
		packagePath:    packagePath,
		converter:      converter.TSConverter{},
		routes:         make(map[string][]any),
		funcArgNames:   make(map[string][]string),
		contextParsers: make(map[reflect.Type]any),
		contextTypes:   make(map[reflect.Type]struct{}),
		diTypeMap:      make(map[reflect.Type]struct{}),
		goFilePath:     goFilePath,
		tsFilePath:     tsFilePath,
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

func (g *Builder) AddDependencyType(t any) *Builder {
	rt := reflect.TypeOf(t).Elem()
	g.diTypeMap[rt] = struct{}{}
	g.diTypes = append(g.diTypes, rt)
	return g
}

func (g *Builder) CreateRouter(routes map[string][]any) *Builder {
	g.routes = routes
	return g
}

func (g *Builder) Build() {
	fmt.Println("Scanning project...")
	g.scanRPC()

	fmt.Println("Generating router...")
	goFile := g.buildHandler()

	g.tsFile += fmt.Sprintf("export type GlutysContract = {\n%s}\n", g.tsMethods)

	file, err := os.Create(g.goFilePath)
	if err != nil {
		panic(err)
	}

	file.WriteString(goFile)

	tsFile, err := os.Create(g.tsFilePath)
	if err != nil {
		panic(err)
	}

	tsFile.WriteString(g.tsFile)
	fmt.Println("Done!")
}
