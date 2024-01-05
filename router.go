package glutys

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"

	. "github.com/dave/jennifer/jen"
	"github.com/onfirebyte/glutys/pkg/util"
)

type Middleware func(next http.Handler) http.Handler

func (g *Builder) buildHandler() string {
	routes := g.routes
	f := NewFilePath(g.packagePath)

	f.HeaderComment("This file is generated by glutys. DO NOT EDIT.")

	paths := []string{}

	generateInitFunc(f, g)
	generateHandlerStruct(f, g)

	for path, procedures := range routes {
		paths = append(paths, path)
		for i, procedure := range procedures {
			// print function arg types
			fnValue := reflect.ValueOf(procedure)
			fnType := fnValue.Type()
			argNum := fnType.NumIn()
			argTypes := make([]reflect.Type, argNum)

			argTSTypes := []struct {
				tsType string
				index  int
			}{}

			for j := 0; j < argNum; j++ {
				argTypes[j] = fnType.In(j)
				_, isContext := g.contextTypes[argTypes[j]]
				_, isDependency := g.diTypeMap[argTypes[j]]
				if !isContext && !isDependency {
					tsFile, tsType := g.converter.ParseType(NoDot(path), argTypes[j])

					g.tsFile += tsFile
					argTSTypes = append(argTSTypes, struct {
						tsType string
						index  int
					}{
						tsType: tsType,
						index:  j,
					})
				}
			}

			// print function return types
			retNum := fnType.NumOut()
			returnTypes := make([]reflect.Type, retNum)
			retTSType := ""
			for i := 0; i < retNum; i++ {
				returnTypes[i] = fnType.Out(i)
			}

			tsFile, tsType := g.converter.ParseType("", returnTypes[0])
			g.tsFile += tsFile
			retTSType = tsType

			argTS := []string{}
			fName := util.GetFunctionName(procedure)
			fArgsName, ok := g.funcArgNames[fName]
			if !ok {
				panic(fmt.Sprintf("Function \"%s\" not found", fName))
			}
			for _, argTSType := range argTSTypes {
				argTS = append(argTS, fmt.Sprintf("%s: %s", fArgsName[argTSType.index], argTSType.tsType))
			}

			g.tsMethods += fmt.Sprintf("\t\"%s\": (%s) => Promise<%s>;\n", path, strings.Join(argTS, ", "), retTSType)

			// check if function is middleware
			isMiddleware := false
			if argNum == 1 &&
				argTypes[0].String() == "http.Handler" &&
				retNum == 1 &&
				returnTypes[0].String() == "http.Handler" {
				isMiddleware = true
			}

			// isProperFunc := true

			if isMiddleware {
				fmt.Println("Middleware")
				continue
			}

			if len(procedures)-1 != i {
				panic("Handler must be last in the list")
			}

			if retNum > 2 {
				panic("Handler must be either void or return only one value with optional error")
			}

			if retNum == 2 && returnTypes[1].String() != "error" {
				panic("Handler must be either void or return only one value with optional error")
			}

			// generate handler function
			generateHandlerFunction(f, fnValue, argTypes, returnTypes, path, g)

		}

	}

	sort.StringSlice(paths).Sort()

	handlerCases := []Code{}
	for _, path := range paths {
		handlerCases = append(handlerCases, Case(Lit(path)).Block(
			Id("h").Dot(HandlerName(path)).Call(Id("w"), Id("r"), Op("&").Id("body")),
		))
	}

	f.Func().Params(Id("h").Add(Id("*Handler"))).Id("Handle").Params(Id("w").Qual("net/http", "ResponseWriter"), Id("r").Op("*").Qual("net/http", "Request")).Block(
		Id("w").Dot("Header").Call().Dot("Set").Call(Lit("Content-Type"), Lit("application/json")),
		// parse body
		Id("body").Op(":=").Qual("github.com/onfirebyte/glutys", "RequestBody").Values(),
		Id("err").Op(":=").Id("json").Dot("NewDecoder").Call(Id("r").Dot("Body")).Dot("Decode").Call(Op("&").Id("body")),
		If(Id("err").Op("!=").Nil()).Block(
			Id("response").Op(":=").Map(String()).Interface().Values(Dict{
				Lit("error"): Lit("Bad Request"),
				Lit("msg"):   Lit("Invalid JSON"),
			}),
			Id("w").Dot("WriteHeader").Call(Qual("net/http", "StatusBadRequest")),
			Id("json").Dot("NewEncoder").Call(Id("w")).Dot("Encode").Call(Id("response")),
		),

		Switch(Id("body").Dot("Method")).Block(handlerCases...),
	)

	return f.GoString()
}

func generateInitFunc(
	f *File,
	g *Builder,
) {
	f.Var().Id("json").Qual("github.com/onfirebyte/jsoniter", "API")
	f.Func().Id("init").Params().Block(
		Id("json").Op("=").Qual("github.com/onfirebyte/jsoniter", "Config").Block(
			Id("EscapeHTML").Op(":").True().Op(","),
			Id("SortMapKeys").Op(":").True().Op(","),
			Id("ValidateJsonRawMessage").Op(":").True().Op(","),
			Id("EmptyCollections").Op(":").True().Op(","),
		).Dot("Froze").Call(),
	)
}

func generateHandlerStruct(
	f *File,
	g *Builder,
) {
	structFields := []Code{}
	newFuncParams := []Code{}
	declareValues := []Code{}

	for _, diType := range g.diTypes {
		pointerCount := 0

		for diType.Kind() == reflect.Ptr {
			pointerCount++
			diType = diType.Elem()
		}

		qual := Qual(diType.PkgPath(), diType.Name())
		for i := 0; i < pointerCount; i++ {
			qual = Op("*").Add(qual)
		}

		structFields = append(structFields, Id(diType.Name()).Add(qual))

		newFuncParams = append(newFuncParams, Id(util.PublicToCamelCase(diType.Name())).Add(qual))

		declareValues = append(declareValues, Id(diType.Name()).Op(":").Id(util.PublicToCamelCase(diType.Name())))
	}

	f.Type().Id("Handler").Struct(structFields...)

	f.Func().Id("NewHandler").Params(newFuncParams...).Id("*Handler").Block(
		Return(Op("&").Id("Handler").Values(declareValues...)),
	)
}

func generateHandlerFunction(
	f *File,
	fnValue reflect.Value,
	argTypes []reflect.Type,
	returnTypes []reflect.Type,
	path string,
	g *Builder,
) {
	blocks := []Code{}

	argPos := 0
	argVars := []string{}
	for i, argType := range argTypes {

		// Handle context
		if _, isContext := g.contextTypes[argType]; isContext {

			contextParser := g.contextParsers[argType]
			contextParserValue := reflect.ValueOf(contextParser)
			contextParserType := contextParserValue.Type()
			retTypeName := contextParserType.Out(0).Name()

			hasErr := reflect.TypeOf(contextParser).NumOut() == 2

			argName := retTypeName + strconv.Itoa(i)
			argVars = append(argVars, argName)
			identifiers := []Code{Id(argName)}
			if hasErr {
				identifiers = append(identifiers, Id("err"+argName))
			}

			fnFullName := runtime.FuncForPC(contextParserValue.Pointer()).Name()
			fnNameSplit := strings.Split(fnFullName, ".")
			fnPkgPath := strings.Join(fnNameSplit[:len(fnNameSplit)-1], ".")
			fnName := fnNameSplit[len(fnNameSplit)-1]

			marshaled := List(identifiers...).Op(":=").Qual(fnPkgPath, fnName).Call(
				Id("r"),
			)
			blocks = append(blocks, marshaled)

			if hasErr {
				blocks = append(blocks, If(Id("err"+argName).Op("!=").Nil()).Block(
					Id("response").Op(":=").Map(String()).Interface().Values(Dict{
						Lit("error"): Lit("Invalid Context"),
						Lit("msg"):   Id("err" + argName).Dot("Error").Call(),
					}),
					Id("w").Dot("WriteHeader").Call(Qual("net/http", "StatusBadRequest")),
					Id("json").Dot("NewEncoder").Call(Id("w")).Dot("Encode").Call(Id("response")),
					Return(),
				))
			}

			continue
		}

		realArgType := argType
		pCount := 0
		for {
			if _, isDependency := g.diTypeMap[realArgType]; isDependency {
				break
			} else if realArgType.Kind() == reflect.Ptr {
				realArgType = realArgType.Elem()
				pCount++
			} else if realArgType.Kind() == reflect.Slice {
				realArgType = realArgType.Elem()
			} else {
				break
			}
		}

		// Handle dependency injection
		if _, isDependency := g.diTypeMap[realArgType]; isDependency {
			for realArgType.Kind() == reflect.Ptr {
				realArgType = realArgType.Elem()
			}
			argVars = append(argVars, fmt.Sprintf("%sh.%s", strings.Repeat("&", pCount), realArgType.Name()))
			continue
		}

		// Handle normal argument

		if realArgType.Kind() == reflect.Interface {
			panic(fmt.Sprintf("Try to use interface %s as argument for RPC. Check the build script again if it's suppose to be dependency.", argType.Name()))
		}

		argName := util.PublicToCamelCase(realArgType.Name()) + strconv.Itoa(i)
		argVars = append(argVars, argName)

		varDeclare := Var().Id(argName).Add(util.GetJenType(argType))

		marshaled := Id("err"+argName).Op(":=").Id("json").Dot("Unmarshal").Call(
			Id("body").Dot("Args").Index(Lit(argPos)),
			Op("&").Id(argName),
		)

		ifErr := If(Id("err"+argName).Op("!=").Id("nil")).Block(
			Id("response").Op(":=").Map(String()).Interface().Values(Dict{
				Lit("error"): Lit("Invalid JSON"),
				Lit("msg"):   Id("err" + argName).Dot("Error").Call(),
			}),
			Id("w").Dot("WriteHeader").Call(Qual("net/http", "StatusBadRequest")),
			Id("json").Dot("NewEncoder").Call(Id("w")).Dot("Encode").Call(Id("response")),
			Return(),
		)

		blocks = append(blocks, varDeclare, marshaled, ifErr)
		argPos++

	}

	callIdentifiers := []Code{
		Id("res"),
	}

	hasErr := len(returnTypes) == 2
	if hasErr {
		callIdentifiers = append(callIdentifiers, Id("err"))
	}

	fnFullName := runtime.FuncForPC(fnValue.Pointer()).Name()
	fnNameSplit := strings.Split(fnFullName, ".")
	fnPkgPath := strings.Join(fnNameSplit[:len(fnNameSplit)-1], ".")
	fnName := fnNameSplit[len(fnNameSplit)-1]

	fnArgNameIds := []Code{}
	for _, argVar := range argVars {
		fnArgNameIds = append(fnArgNameIds, Id(argVar))
	}

	call := List(callIdentifiers...).Op(":=").Qual(fnPkgPath, fnName).Call(fnArgNameIds...)

	blocks = append(blocks, call)

	if hasErr {
		blocks = append(blocks, If(Id("err").Op("!=").Nil()).Block(
			Id("w").Dot("WriteHeader").Call(Qual("net/http", "StatusBadRequest")),
			Id("json").Dot("NewEncoder").Call(Id("w")).Dot("Encode").Call(
				Map(String()).Interface().Values(Dict{
					Lit("error"): Lit("Bad Request"),
					Lit("msg"):   Id("err").Dot("Error").Call(),
				}),
			),
			Return(),
		))
	}

	blocks = append(blocks,
		Id("w").Dot("WriteHeader").Call(Qual("net/http", "StatusOK")),
		Id("json").Dot("NewEncoder").Call(Id("w")).Dot("Encode").Call(
			Id("res")),
		Return())

	f.Func().Params(Id("h").Add(Id("*Handler"))).Id(HandlerName(path)).
		Params(
			Id("w").Qual("net/http", "ResponseWriter"),
			Id("r").Op("*").Qual("net/http", "Request"),
			Id("body").Op("*").Qual("github.com/onfirebyte/glutys", "RequestBody")).
		Block(
			blocks...,
		)
}

func HandlerName(path string) string {
	pathSlice := strings.Split(path, ".")
	for i, p := range pathSlice {
		pathSlice[i] = util.CamelCaseToPublic(p)
	}
	return strings.Join(pathSlice, "") + "Handler"
}

func NoDot(path string) string {
	pathSlice := strings.Split(path, ".")
	for i, p := range pathSlice {
		pathSlice[i] = util.CamelCaseToPublic(p)
	}
	return strings.Join(pathSlice, "")
}
