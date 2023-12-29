package glutys

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func (g *Builder) scanRPC() {
	// scan go file in current project
	fset := token.NewFileSet()

	err := filepath.Walk(g.RootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			// Parse the Go file
			file, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if parseErr != nil {
				log.Fatalf("Error parsing file %s: %s\n", path, parseErr)
			}

			// Process the AST nodes
			ast.Inspect(file, func(node ast.Node) bool {
				// check if node is a function declaration
				if _, ok := node.(*ast.FuncDecl); node != nil && ok {
					// get function name with package name
					funcName := fmt.Sprintf("%s.%s", file.Name.Name, node.(*ast.FuncDecl).Name.Name)

					// get arguments
					args := node.(*ast.FuncDecl).Type.Params.List
					argsName := []string{}
					for _, arg := range args {
						argsName = append(argsName, arg.Names[0].Name)
					}

					g.funcArgNames[funcName] = argsName
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Error walking through directory: %s\n", err)
	}
}
