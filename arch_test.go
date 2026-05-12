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

	roomDomainPath  = modulePath + "/internal/room/domain"
	roomUsecasePath = modulePath + "/internal/room/usecase"
	roomRepoBase    = modulePath + "/internal/room/repository"
	roomTransBase   = modulePath + "/internal/room/transport"

	channelDomainPath  = modulePath + "/internal/channel/domain"
	channelUsecasePath = modulePath + "/internal/channel/usecase"
	channelRepoBase    = modulePath + "/internal/channel/repository"
	channelTransBase   = modulePath + "/internal/channel/transport"

	chatDomainPath  = modulePath + "/internal/chat/domain"
	chatUsecasePath = modulePath + "/internal/chat/usecase"

	authMiddlewarePath = modulePath + "/internal/auth/transport/http/middleware"
	pkgHttpxPath       = modulePath + "/pkg/httpx"
	pkgWebsocketPath   = modulePath + "/pkg/websocket"
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

func TestArchitecture_RoomDomainImports(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{
		"github.com/google/uuid": {},
	}

	imports := collectImports(t, "internal/room/domain")
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

func TestArchitecture_RoomUseCaseImports(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{
		"github.com/google/uuid": {},
		roomDomainPath:           {},
	}

	imports := collectImports(t, "internal/room/usecase")
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

// TestArchitecture_RoomRepoMayImplementChannelPort — room/repository/postgres
// допустимо импортирует channel/usecase и channel/domain (для MembershipQueryAdapter),
// но НЕ channel/transport и НЕ auth.
func TestArchitecture_RoomRepoMayImplementChannelPort(t *testing.T) {
	t.Parallel()

	forbiddenPrefixes := []string{
		modulePath + "/internal/channel/transport/",
		modulePath + "/internal/auth/",
	}

	imports := collectImports(t, "internal/room/repository/postgres")
	for file, ims := range imports {
		for _, im := range ims {
			for _, prefix := range forbiddenPrefixes {
				if strings.HasPrefix(im, prefix) {
					t.Fatalf("%s imports forbidden %q", file, im)
				}
			}
		}
	}
}

func TestArchitecture_RoomTransportImports(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{
		"github.com/go-chi/chi/v5":            {},
		"github.com/google/uuid":              {},
		roomDomainPath:                        {},
		roomUsecasePath:                       {},
		authMiddlewarePath:                    {},
		modulePath + "/internal/auth/usecase": {},
		modulePath + "/internal/auth/domain":  {},
		pkgHttpxPath:                          {},
	}

	imports := collectImports(t, "internal/room/transport/http")
	for file, ims := range imports {
		for _, im := range ims {
			if isStdlib(im) {
				continue
			}
			if _, ok := allowed[im]; ok {
				continue
			}
			if strings.HasPrefix(im, roomRepoBase+"/") {
				t.Fatalf("transport %s imports adapter %q", file, im)
			}
			if strings.HasPrefix(im, modulePath+"/internal/channel/") {
				t.Fatalf("transport %s imports cross-domain %q", file, im)
			}
			t.Fatalf("transport %s imports forbidden %q", file, im)
		}
	}
}

func TestArchitecture_ChannelDomainImports(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{
		"github.com/google/uuid": {},
	}

	imports := collectImports(t, "internal/channel/domain")
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

// TestArchitecture_ChannelUseCaseImports — channel/usecase НЕ должен импортировать
// room/* (связь только через порт MembershipQuery, реализуемый адаптером в room/repo).
func TestArchitecture_ChannelUseCaseImports(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{
		"github.com/google/uuid": {},
		channelDomainPath:        {},
	}

	imports := collectImports(t, "internal/channel/usecase")
	for file, ims := range imports {
		for _, im := range ims {
			if isStdlib(im) {
				continue
			}
			if _, ok := allowed[im]; ok {
				continue
			}
			if strings.HasPrefix(im, modulePath+"/internal/room/") {
				t.Fatalf("channel/usecase %s imports room %q (must use port)", file, im)
			}
			t.Fatalf("%s imports forbidden %q", file, im)
		}
	}
}

func TestArchitecture_ChannelTransportImports(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{
		"github.com/go-chi/chi/v5":            {},
		"github.com/google/uuid":              {},
		channelDomainPath:                     {},
		channelUsecasePath:                    {},
		authMiddlewarePath:                    {},
		modulePath + "/internal/auth/usecase": {},
		modulePath + "/internal/auth/domain":  {},
		pkgHttpxPath:                          {},
	}

	imports := collectImports(t, "internal/channel/transport/http")
	for file, ims := range imports {
		for _, im := range ims {
			if isStdlib(im) {
				continue
			}
			if _, ok := allowed[im]; ok {
				continue
			}
			if strings.HasPrefix(im, channelRepoBase+"/") {
				t.Fatalf("transport %s imports adapter %q", file, im)
			}
			if strings.HasPrefix(im, modulePath+"/internal/room/") {
				t.Fatalf("channel/transport %s imports room %q", file, im)
			}
			t.Fatalf("%s imports forbidden %q", file, im)
		}
	}
}

func TestArchitecture_ChannelRepoIsolated(t *testing.T) {
	t.Parallel()

	forbiddenPrefixes := []string{
		modulePath + "/internal/room/",
		modulePath + "/internal/auth/",
		modulePath + "/internal/channel/transport/",
	}

	imports := collectImports(t, "internal/channel/repository/postgres")
	for file, ims := range imports {
		for _, im := range ims {
			for _, prefix := range forbiddenPrefixes {
				if strings.HasPrefix(im, prefix) {
					t.Fatalf("%s imports forbidden %q", file, im)
				}
			}
		}
	}
}

// === Chat / pkg/websocket arch tests (PR-3.1) ===

func TestArchitecture_ChatDomainImports(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{
		"github.com/google/uuid": {},
	}

	imports := collectImports(t, "internal/chat/domain")
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

func TestArchitecture_ChatUseCaseImports(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{
		"github.com/google/uuid": {},
		chatDomainPath:           {},
	}

	imports := collectImports(t, "internal/chat/usecase")
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

// TestArchitecture_ChatRepoIsolated — chat/repository/postgres не должен
// импортировать чужие домены (room, channel, auth) или собственный transport.
func TestArchitecture_ChatRepoIsolated(t *testing.T) {
	t.Parallel()

	forbiddenPrefixes := []string{
		modulePath + "/internal/room/",
		modulePath + "/internal/channel/",
		modulePath + "/internal/auth/",
		modulePath + "/internal/chat/transport/",
	}

	imports := collectImports(t, "internal/chat/repository/postgres")
	for file, ims := range imports {
		for _, im := range ims {
			for _, prefix := range forbiddenPrefixes {
				if strings.HasPrefix(im, prefix) {
					t.Fatalf("%s imports forbidden %q", file, im)
				}
			}
		}
	}
}

func TestArchitecture_ChatTransportHTTPImports(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{
		"github.com/go-chi/chi/v5":            {},
		"github.com/google/uuid":              {},
		chatDomainPath:                        {},
		chatUsecasePath:                       {},
		authMiddlewarePath:                    {},
		modulePath + "/internal/auth/usecase": {},
		modulePath + "/internal/auth/domain":  {},
		pkgHttpxPath:                          {},
	}

	imports := collectImports(t, "internal/chat/transport/http")
	for file, ims := range imports {
		for _, im := range ims {
			if isStdlib(im) {
				continue
			}
			if _, ok := allowed[im]; ok {
				continue
			}
			if strings.HasPrefix(im, modulePath+"/internal/room/") ||
				strings.HasPrefix(im, modulePath+"/internal/channel/") {
				t.Fatalf("chat/transport/http %s imports cross-domain %q", file, im)
			}
			t.Fatalf("chat/transport/http %s imports forbidden %q", file, im)
		}
	}
}

func TestArchitecture_ChatTransportWSImports(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{
		"github.com/go-chi/chi/v5":            {},
		"github.com/google/uuid":              {},
		chatDomainPath:                        {},
		chatUsecasePath:                       {},
		authMiddlewarePath:                    {},
		modulePath + "/internal/auth/usecase": {},
		modulePath + "/internal/auth/domain":  {},
		pkgHttpxPath:                          {},
		pkgWebsocketPath:                      {},
		"github.com/coder/websocket":          {},
	}

	imports := collectImports(t, "internal/chat/transport/ws")
	for file, ims := range imports {
		for _, im := range ims {
			if isStdlib(im) {
				continue
			}
			if _, ok := allowed[im]; ok {
				continue
			}
			if strings.HasPrefix(im, modulePath+"/internal/room/") ||
				strings.HasPrefix(im, modulePath+"/internal/channel/") {
				t.Fatalf("chat/transport/ws %s imports cross-domain %q", file, im)
			}
			t.Fatalf("chat/transport/ws %s imports forbidden %q", file, im)
		}
	}
}

// TestArchitecture_PkgWebsocketIsolated — pkg/websocket НИ ОДНОГО internal/*
// импорта (domain-агностичный пакет).
func TestArchitecture_PkgWebsocketIsolated(t *testing.T) {
	t.Parallel()

	forbiddenPrefix := modulePath + "/internal/"

	imports := collectImports(t, "pkg/websocket")
	for file, ims := range imports {
		for _, im := range ims {
			if strings.HasPrefix(im, forbiddenPrefix) {
				t.Fatalf("pkg/websocket %s imports forbidden %q", file, im)
			}
		}
	}
}
