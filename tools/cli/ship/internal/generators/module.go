package generators

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ModuleMakeOptions struct {
	Name       string
	Path       string
	ModuleBase string
	DryRun     bool
	Force      bool
}

type ModuleDeps struct {
	Out        io.Writer
	Err        io.Writer
	PathExists func(path string) bool
}

type moduleScaffoldFile struct {
	Path          string
	Content       string
	ContractOwner string
}

func RunMakeModule(args []string, d ModuleDeps) int {
	opts, err := ParseMakeModuleArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid make:module arguments: %v\n", err)
		return 1
	}

	tokens := splitWords(opts.Name)
	if len(tokens) == 0 {
		fmt.Fprintln(d.Err, "usage: ship make:module <Name> [--path modules] [--module-base github.com/leomorpho/goship-modules] [--dry-run] [--force]")
		return 1
	}
	moduleName := strings.Join(tokens, "")
	moduleDir := filepath.Join(opts.Path, moduleName)
	modulePath := strings.TrimRight(opts.ModuleBase, "/") + "/" + moduleName

	if d.PathExists(moduleDir) && !opts.Force {
		fmt.Fprintf(d.Err, "refusing to overwrite existing module directory: %s (use --force)\n", moduleDir)
		return 1
	}

	files := moduleScaffoldFiles(moduleDir, moduleName, modulePath)

	if opts.DryRun {
		fmt.Fprintf(d.Out, "Module scaffold plan (dry-run):\n- module: %s\n- dir: %s\n", modulePath, moduleDir)
		for _, file := range files {
			fmt.Fprintf(d.Out, "- file: %s -> owner: %s\n", file.Path, file.ContractOwner)
		}
		return 0
	}

	for _, file := range files {
		if err := os.MkdirAll(filepath.Dir(file.Path), 0o755); err != nil {
			fmt.Fprintf(d.Err, "failed to create directory for %s: %v\n", file.Path, err)
			return 1
		}
		if err := os.WriteFile(file.Path, []byte(file.Content), 0o644); err != nil {
			fmt.Fprintf(d.Err, "failed to write %s: %v\n", file.Path, err)
			return 1
		}
	}

	fmt.Fprintf(d.Out, "Generated module scaffold at %s (%s)\n", moduleDir, modulePath)
	return 0
}

func moduleScaffoldFiles(moduleDir, moduleName, modulePath string) []moduleScaffoldFile {
	files := []moduleScaffoldFile{
		{Path: filepath.Join(moduleDir, "go.mod"), Content: renderModuleGoMod(modulePath), ContractOwner: "module-runtime"},
		{Path: filepath.Join(moduleDir, "module.go"), Content: renderModuleEntrypoint(moduleName), ContractOwner: "install-contract"},
		{Path: filepath.Join(moduleDir, "contracts.go"), Content: renderModuleContracts(moduleName), ContractOwner: "service-contract"},
		{Path: filepath.Join(moduleDir, "types.go"), Content: renderModuleTypes(moduleName), ContractOwner: "domain-types"},
		{Path: filepath.Join(moduleDir, "errors.go"), Content: renderModuleErrors(moduleName), ContractOwner: "error-contract"},
		{Path: filepath.Join(moduleDir, "service.go"), Content: renderModuleService(moduleName), ContractOwner: "service-runtime"},
		{Path: filepath.Join(moduleDir, "service_test.go"), Content: renderModuleServiceTest(moduleName), ContractOwner: "service-tests"},
		{Path: filepath.Join(moduleDir, "db", "bobgen.yaml"), Content: renderModuleBobgenConfig(moduleDir), ContractOwner: "db-codegen"},
		{Path: filepath.Join(moduleDir, "db", "migrate", "migrations", ".gitkeep"), Content: "", ContractOwner: "migrations"},
		{Path: filepath.Join(moduleDir, "db", "queries", ".gitkeep"), Content: "", ContractOwner: "queries"},
		{Path: filepath.Join(moduleDir, "db", "gen", ".gitkeep"), Content: "", ContractOwner: "generated-db"},
		{Path: filepath.Join(moduleDir, "AGENTS.md"), Content: renderModuleAgentsMD(moduleName), ContractOwner: "agent-context"},
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files
}

func ParseMakeModuleArgs(args []string) (ModuleMakeOptions, error) {
	opts := ModuleMakeOptions{Path: "modules", ModuleBase: "github.com/leomorpho/goship-modules"}
	if len(args) == 0 {
		return opts, errors.New("usage: ship make:module <Name> [--path modules] [--module-base github.com/leomorpho/goship-modules] [--dry-run] [--force]")
	}
	opts.Name = strings.TrimSpace(args[0])
	for i := 1; i < len(args); i++ {
		switch {
		case args[i] == "--dry-run":
			opts.DryRun = true
		case args[i] == "--force":
			opts.Force = true
		case strings.HasPrefix(args[i], "--path="):
			opts.Path = strings.TrimSpace(strings.TrimPrefix(args[i], "--path="))
		case strings.HasPrefix(args[i], "--module-base="):
			opts.ModuleBase = strings.TrimSpace(strings.TrimPrefix(args[i], "--module-base="))
		case args[i] == "--path":
			if i+1 >= len(args) {
				return opts, errors.New("missing value for --path")
			}
			i++
			opts.Path = strings.TrimSpace(args[i])
		case args[i] == "--module-base":
			if i+1 >= len(args) {
				return opts, errors.New("missing value for --module-base")
			}
			i++
			opts.ModuleBase = strings.TrimSpace(args[i])
		default:
			return opts, fmt.Errorf("unknown option: %s", args[i])
		}
	}
	if opts.Name == "" {
		return opts, errors.New("module name cannot be empty")
	}
	if strings.TrimSpace(opts.Path) == "" {
		return opts, errors.New("path cannot be empty")
	}
	if strings.TrimSpace(opts.ModuleBase) == "" {
		return opts, errors.New("module-base cannot be empty")
	}
	normalizedPath, err := normalizeOwnedGeneratorPath(opts.Path, "modules")
	if err != nil {
		return opts, err
	}
	opts.Path = normalizedPath
	return opts, nil
}

func renderModuleGoMod(modulePath string) string {
	return fmt.Sprintf(`module %s

go 1.23.0

require github.com/AfterShip/email-verifier v1.3.3
`, modulePath)
}

func renderModuleEntrypoint(packageName string) string {
	return fmt.Sprintf(`package %s

const ModuleID = %q

// InstallContract defines the files/surfaces a module install is expected to touch.
type InstallContract struct {
	Routes     []string
	Config     []string
	Assets     []string
	Jobs       []string
	Templates  []string
	Migrations []string
}

// Contract returns the default module install contract shape for this scaffold.
func Contract() InstallContract {
	return InstallContract{
		Config: []string{
			"config/modules.yaml",
		},
	}
}

// New is the module entrypoint used by app wiring.
func New(store Store) *Service {
	return NewService(store)
}
`, packageName, packageName)
}

func renderModuleContracts(packageName string) string {
	return fmt.Sprintf(`package %s

import "context"

// Store defines the DB boundary for this module.
type Store interface {
	CreateList(ctx context.Context, list List) error
	Subscribe(ctx context.Context, email string, list List, latitude, longitude *float64) (*Subscription, error)
	Unsubscribe(ctx context.Context, email string, token string, list List) error
	Confirm(ctx context.Context, code string) error
}
`, packageName)
}

func renderModuleTypes(packageName string) string {
	return fmt.Sprintf(`package %s

// List identifies a subscription list (e.g. newsletter).
type List string

// Subscription is the module-owned subscription model.
type Subscription struct {
	ID               int
	Email            string
	Verified         bool
	ConfirmationCode string
	Lat              float64
	Lon              float64
}
`, packageName)
}

func renderModuleErrors(packageName string) string {
	return fmt.Sprintf(`package %s

import (
	"errors"
	"fmt"
)

type ErrAlreadySubscribed struct {
	List string
	Err  error
}

func (e *ErrAlreadySubscribed) Error() string {
	return fmt.Sprintf("email address is already subscribed to list %%s, error is: %%v", e.List, e.Err)
}

var ErrInvalidConfirmationCode = errors.New("confirmation code is invalid")
var ErrEmailSyntaxInvalid = errors.New("email address syntax is invalid")
var ErrEmailAddressInvalid = errors.New("invalid email address")

type ErrEmailVerificationFailed struct {
	Err error
}

func (e *ErrEmailVerificationFailed) Error() string {
	return fmt.Sprintf("verify email address failed, error is: %%v", e.Err)
}
`, packageName)
}

func renderModuleService(packageName string) string {
	return fmt.Sprintf(`package %s

import (
	"context"

	emailverifier "github.com/AfterShip/email-verifier"
)

// Service is the public API for this module.
type Service struct {
	store      Store
	verifyFunc func(email string) error
}

func NewService(store Store) *Service {
	return NewServiceWithVerifier(store, verifyWithAfterShip())
}

func NewServiceWithVerifier(store Store, verifyFunc func(email string) error) *Service {
	return &Service{store: store, verifyFunc: verifyFunc}
}

func verifyWithAfterShip() func(email string) error {
	verifier := emailverifier.NewVerifier()
	return func(email string) error {
		ret, err := verifier.Verify(email)
		if err != nil {
			return &ErrEmailVerificationFailed{Err: err}
		}
		if !ret.Syntax.Valid {
			return ErrEmailSyntaxInvalid
		}
		return nil
	}
}

func (s *Service) VerifyEmailValidity(email string) error {
	if s.verifyFunc == nil {
		return nil
	}
	return s.verifyFunc(email)
}

func (s *Service) CreateList(ctx context.Context, list List) error {
	return s.store.CreateList(ctx, list)
}

func (s *Service) Subscribe(
	ctx context.Context, email string, list List, latitude, longitude *float64,
) (*Subscription, error) {
	if err := s.VerifyEmailValidity(email); err != nil {
		return nil, ErrEmailAddressInvalid
	}
	return s.store.Subscribe(ctx, email, list, latitude, longitude)
}

func (s *Service) Unsubscribe(ctx context.Context, email string, token string, list List) error {
	return s.store.Unsubscribe(ctx, email, token, list)
}

func (s *Service) Confirm(ctx context.Context, code string) error {
	return s.store.Confirm(ctx, code)
}
`, packageName)
}

func renderModuleServiceTest(packageName string) string {
	return fmt.Sprintf(`package %s

import (
	"context"
	"errors"
	"testing"
)

type mockStore struct {
	subscribeResult *Subscription
}

func (m *mockStore) CreateList(context.Context, List) error { return nil }

func (m *mockStore) Subscribe(context.Context, string, List, *float64, *float64) (*Subscription, error) {
	if m.subscribeResult == nil {
		m.subscribeResult = &Subscription{ID: 1, ConfirmationCode: "abc"}
	}
	return m.subscribeResult, nil
}

func (m *mockStore) Unsubscribe(context.Context, string, string, List) error { return nil }
func (m *mockStore) Confirm(context.Context, string) error                    { return nil }

func TestServiceDelegates(t *testing.T) {
	svc := NewServiceWithVerifier(&mockStore{}, func(string) error { return nil })
	if err := svc.CreateList(context.Background(), List("newsletter")); err != nil {
		t.Fatalf("CreateList err: %%v", err)
	}
	if _, err := svc.Subscribe(context.Background(), "a@b.com", List("newsletter"), nil, nil); err != nil {
		t.Fatalf("Subscribe err: %%v", err)
	}
	if err := svc.Unsubscribe(context.Background(), "a@b.com", "tok", List("newsletter")); err != nil {
		t.Fatalf("Unsubscribe err: %%v", err)
	}
	if err := svc.Confirm(context.Background(), "tok"); err != nil {
		t.Fatalf("Confirm err: %%v", err)
	}
}

func TestServiceValidationError(t *testing.T) {
	svc := NewServiceWithVerifier(&mockStore{}, func(string) error { return errors.New("invalid") })
	if _, err := svc.Subscribe(context.Background(), "bad", List("newsletter"), nil, nil); !errors.Is(err, ErrEmailAddressInvalid) {
		t.Fatalf("err=%%v want=%%v", err, ErrEmailAddressInvalid)
	}
}
`, packageName)
}

func renderModuleBobgenConfig(moduleDir string) string {
	modulePath := filepath.ToSlash(moduleDir)
	return fmt.Sprintf(`# Module-local Bob codegen config.
sql:
  dialect: psql
  pattern: "%s"
  queries:
    - "%s"

output: "%s"
`,
		filepath.ToSlash(filepath.Join(modulePath, "db", "migrate", "migrations", "*.sql")),
		filepath.ToSlash(filepath.Join(modulePath, "db", "queries")),
		filepath.ToSlash(filepath.Join(modulePath, "db", "gen")),
	)
}

func renderModuleAgentsMD(moduleName string) string {
	return fmt.Sprintf(`# Module: %s

## What This Module Does

<one paragraph>

## Files

- `+"`module.go`"+` - module ID, config schema, interface implementation
- `+"`service.go`"+` - business logic (exported API)
- `+"`store.go`"+` - storage interface
- `+"`store_sql.go`"+` - SQL implementation using Bob
- `+"`routes.go`"+` - route registration (implement `+"`core.RoutableModule`"+`)
- `+"`views/`"+` - templ templates
- `+"`db/migrate/migrations/`"+` - SQL migration files

## Interfaces Implemented

- `+"`core.Module`"+` (`+"`module.go`"+`)
- `+"`core.RoutableModule`"+` (`+"`routes.go`"+`) if this module has HTTP routes

## Dependencies

- Other modules this module imports: <list or "none">
- Framework packages used: <list>

## Conventions

- All HTTP handlers are in `+"`routes.go`"+`, nowhere else.
- All business logic is in `+"`service.go`"+`; controllers never call the store directly.
- All DB access goes through the store interface; never put raw SQL in `+"`service.go`"+`.
- Viewmodels are value types with no pointer fields.
- Run `+"`ship verify`"+` after every change.
`, moduleName)
}
