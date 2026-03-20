package viewmodels

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEveryViewmodelStructHasConstructor(t *testing.T) {
	pkgFiles := parsePackageFiles(t, ".")

	structs := map[string]struct{}{}
	constructors := map[string]struct{}{}

	for _, file := range pkgFiles {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if d.Tok != token.TYPE {
					continue
				}
				for _, spec := range d.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if _, ok := typeSpec.Type.(*ast.StructType); ok {
						structs[typeSpec.Name.Name] = struct{}{}
					}
				}
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name != nil {
					constructors[d.Name.Name] = struct{}{}
				}
			}
		}
	}

	var missing []string
	for name := range structs {
		if _, ok := constructors["New"+name]; !ok {
			missing = append(missing, name)
		}
	}

	require.Empty(t, missing, "every viewmodel struct must have a constructor")
}

func TestNoExternalViewmodelCompositeLiterals(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join(".", "..", "..", ".."))
	fset := token.NewFileSet()

	var offenders []string
	for _, root := range []string{"app", "modules"} {
		base := filepath.Join(repoRoot, root)
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			require.NoError(t, err)
			if d.IsDir() {
				if d.Name() == "gen" {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			file, parseErr := parser.ParseFile(fset, path, nil, 0)
			require.NoError(t, parseErr)
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				sel, ok := lit.Type.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok || pkg.Name != "viewmodels" {
					return true
				}
				pos := fset.Position(lit.Pos())
				offenders = append(offenders, pos.String())
				return true
			})
			return nil
		})
		require.NoError(t, err)
	}

	require.Empty(t, offenders, "use viewmodel constructors outside app/web/viewmodels")
}

func parsePackageFiles(t *testing.T, dir string) map[string]*ast.File {
	t.Helper()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info fs.FileInfo) bool {
		name := info.Name()
		return filepath.Ext(name) == ".go" && !strings.HasSuffix(name, "_test.go")
	}, 0)
	require.NoError(t, err)

	pkg, ok := pkgs["viewmodels"]
	require.True(t, ok, "viewmodels package not found")
	return pkg.Files
}
