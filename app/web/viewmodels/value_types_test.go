package viewmodels

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestViewmodelsUseValueTypesOnly(t *testing.T) {
	t.Helper()

	dir := "."
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(infoFileInfo fs.FileInfo) bool {
		name := infoFileInfo.Name()
		return filepath.Ext(name) == ".go" && filepath.Base(name) != "value_types_test.go"
	}, 0)
	require.NoError(t, err)

	pkg, ok := pkgs["viewmodels"]
	require.True(t, ok, "viewmodels package not found")

	var pointerFields []string
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range structType.Fields.List {
					if !exprContainsPointer(field.Type) {
						continue
					}
					if len(field.Names) == 0 {
						pointerFields = append(pointerFields, typeSpec.Name.Name)
						continue
					}
					for _, name := range field.Names {
						pointerFields = append(pointerFields, typeSpec.Name.Name+"."+name.Name)
					}
				}
			}
		}
	}

	require.Empty(t, pointerFields, "viewmodels must not contain pointer-backed fields")
}

func exprContainsPointer(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return true
	case *ast.ArrayType:
		return exprContainsPointer(e.Elt)
	case *ast.MapType:
		return exprContainsPointer(e.Key) || exprContainsPointer(e.Value)
	case *ast.ChanType:
		return exprContainsPointer(e.Value)
	case *ast.Ellipsis:
		return exprContainsPointer(e.Elt)
	case *ast.StructType:
		for _, field := range e.Fields.List {
			if exprContainsPointer(field.Type) {
				return true
			}
		}
	case *ast.FuncType:
		if e.Params != nil {
			for _, field := range e.Params.List {
				if exprContainsPointer(field.Type) {
					return true
				}
			}
		}
		if e.Results != nil {
			for _, field := range e.Results.List {
				if exprContainsPointer(field.Type) {
					return true
				}
			}
		}
	}
	return false
}
