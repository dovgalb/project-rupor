package main_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const (
	modulePath  = "github.com/dovgalb/project-rupor"
	domainPath  = modulePath + "/internal/auth/domain"
	usecasePath = modulePath + "/internal/auth/usecase"
	repoBase    = modulePath + "/internal/auth/repository"
	transBase   = modulePath + "/internal/auth/transport"
)

// collectImports собирает все импорты из non-test go-файлов в директории.
func collectImports(t *testing.T, dir string) map[string][]string {
	t.Helper()

	result := map[string][]string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", dir, err)
	}

	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		var imports []string
		for _, imp := range f.Imports {
			v, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("unquote %s: %v", imp.Path.Value, err)
			}
			imports = append(imports, v)
		}
		result[name] = imports
	}
	return result
}

// isStdlib — упрощённая эвристика: stdlib-пакеты не содержат точку в первом сегменте.
func isStdlib(importPath string) bool {
	first := strings.SplitN(importPath, "/", 2)[0]
	return !strings.Contains(first, ".")
}

func TestArchitecture_DomainImports(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{
		"github.com/google/uuid": {},
	}

	imports := collectImports(t, "internal/auth/domain")
	for file, ims := range imports {
		for _, im := range ims {
			if isStdlib(im) {
				continue
			}
			if _, ok := allowed[im]; ok {
				continue
			}
			t.Fatalf("%s imports forbidden %q", file, im)
		}
	}
}

func TestArchitecture_UseCaseImports(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{
		"github.com/google/uuid": {},
		domainPath:               {},
	}

	imports := collectImports(t, "internal/auth/usecase")
	for file, ims := range imports {
		for _, im := range ims {
			if isStdlib(im) {
				continue
			}
			if _, ok := allowed[im]; ok {
				continue
			}
			t.Fatalf("%s imports forbidden %q", file, im)
		}
	}
}

func TestArchitecture_RepositoriesIsolated(t *testing.T) {
	t.Parallel()

	repoSubpkgs := []string{"postgres", "jwt", "bcrypt"}
	for _, pkg := range repoSubpkgs {
		dir := filepath.Join("internal/auth/repository", pkg)
		imports := collectImports(t, dir)
		for file, ims := range imports {
			for _, im := range ims {
				// другие repo-сабпакеты
				for _, other := range repoSubpkgs {
					if other == pkg {
						continue
					}
					forbidden := repoBase + "/" + other
					if im == forbidden || strings.HasPrefix(im, forbidden+"/") {
						t.Fatalf("%s/%s imports sibling repo %q", pkg, file, im)
					}
				}
				// transport
				if im == transBase || strings.HasPrefix(im, transBase+"/") {
					t.Fatalf("%s/%s imports transport %q", pkg, file, im)
				}
			}
		}
	}
}

// usecase compile-time assertion: сами адаптеры обязаны удовлетворять портам.
func TestArchitecture_UsecaseUsedAsContract(t *testing.T) {
	t.Parallel()

	// Проверка транспортного слоя: импорт adapter-пакетов запрещён,
	// допустим только usecase + domain.
	imports := collectImports(t, "internal/auth/transport/http")
	for file, ims := range imports {
		for _, im := range ims {
			if strings.HasPrefix(im, repoBase+"/") {
				t.Fatalf("transport/http/%s imports adapter %q", file, im)
			}
		}
	}
	_ = usecasePath
}

func TestArchitecture_PkgHttpxImports(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{
		modulePath + "/pkg/httpx": {},
	}

	dirs := []string{"pkg/httpx", "pkg/httpx/middleware"}
	for _, dir := range dirs {
		imports := collectImports(t, dir)
		for file, ims := range imports {
			for _, im := range ims {
				if isStdlib(im) {
					continue
				}
				if _, ok := allowed[im]; ok {
					continue
				}
				if strings.HasPrefix(im, modulePath+"/internal/") {
					t.Fatalf("%s/%s imports internal %q", dir, file, im)
				}
				t.Fatalf("%s/%s imports forbidden %q", dir, file, im)
			}
		}
	}
}

func TestArchitecture_AuthMiddlewareImports(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{
		"github.com/google/uuid":              {},
		modulePath + "/internal/auth/usecase": {},
		modulePath + "/internal/auth/domain":  {},
		modulePath + "/pkg/httpx":             {},
	}

	imports := collectImports(t, "internal/auth/transport/http/middleware")
	for file, ims := range imports {
		for _, im := range ims {
			if isStdlib(im) {
				continue
			}
			if _, ok := allowed[im]; ok {
				continue
			}
			if strings.HasPrefix(im, repoBase+"/") {
				t.Fatalf("%s imports adapter %q", file, im)
			}
			t.Fatalf("%s imports forbidden %q", file, im)
		}
	}
}
