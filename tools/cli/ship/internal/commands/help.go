package commands

import (
	"fmt"
	"io"
)

func PrintRootHelp(w io.Writer) {
	fmt.Fprintln(w, "ship - GoShip CLI")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  ship <command> [options]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Direct Commands:")
	fmt.Fprintln(w, "  ship new <app> [flags]                   Create a new app scaffold")
	fmt.Fprintln(w, "  ship dev [--web|--worker|--all]          Run local runtime processes")
	fmt.Fprintln(w, "  ship check                               Run fast project checks")
	fmt.Fprintln(w, "  ship test [--integration]                Run tests (unit by default)")
	fmt.Fprintln(w, "  ship verify [--skip-tests] [--json]      Run full verification workflow")
	fmt.Fprintln(w, "  ship doctor [--json]                     Run repository policy checks")
	fmt.Fprintln(w, "  ship config:validate [--json]            Validate config contract")
	fmt.Fprintln(w, "  ship routes [--json]                     Show route inventory")
	fmt.Fprintln(w, "  ship describe [--pretty]                 Show runtime/module inventory")
	fmt.Fprintln(w, "  ship runtime:report --json               Show machine-readable runtime capability report")
	fmt.Fprintln(w, "  ship run:command <name> [-- <args...>]   Run app-defined CLI command")
	fmt.Fprintln(w, "  ship module:add <name> [--dry-run]       Enable a module")
	fmt.Fprintln(w, "  ship module:remove <name> [--dry-run]    Disable a module")
	fmt.Fprintln(w, "  ship upgrade --to <version> [--dry-run]  Upgrade pinned CLI tooling")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Command Groups:")
	fmt.Fprintln(w, "  ship config --help                       Config command help")
	fmt.Fprintln(w, "  ship i18n --help                         i18n command help")
	fmt.Fprintln(w, "  ship agent --help                        Agent workflow command help")
	fmt.Fprintln(w, "  ship db --help                           Database command help")
	fmt.Fprintln(w, "  ship make --help                         Generator command help")
	fmt.Fprintln(w, "  ship infra --help                        Local infrastructure command help")
	fmt.Fprintln(w, "  ship templ --help                        Templ command help")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  ship new demo")
	fmt.Fprintln(w, "  ship dev --all")
	fmt.Fprintln(w, "  ship db:migrate")
	fmt.Fprintln(w, "  ship make:resource contact")
}

func PrintTemplHelp(w io.Writer) {
	fmt.Fprintln(w, "ship templ commands:")
	fmt.Fprintln(w, "  ship templ generate [--path <dir>] [--file <file.templ>]  Generate templ code and relocate outputs to per-package gen/ directories")
}

func PrintMakeHelp(w io.Writer) {
	fmt.Fprintln(w, "ship make commands:")
	fmt.Fprintln(w, "  ship make:scaffold <Name> [fields...] [--path app] [--views templ|none] [--auth public|auth] [--api] [--migrate] [--dry-run] [--force]  Generate model + migration + controller/resource wiring")
	fmt.Fprintln(w, "  ship make:controller <Name|NameController> [--actions index,show,create,update,destroy] [--auth public|auth] [--wire]                    Generate a controller with optional route wiring")
	fmt.Fprintln(w, "  ship make:resource <name> [--path app] [--auth public|auth] [--views templ|none] [--wire] [--dry-run]                                  Generate a route handler and optional page template")
	fmt.Fprintln(w, "  ship make:model <Name> [fields...]                                                                                                          Generate a DB query/model scaffold")
	fmt.Fprintln(w, "  ship make:factory <Name>                                                                                                                     Generate a test data factory")
	fmt.Fprintln(w, "  ship make:locale <code>                                                                                                                      Generate locale file from baseline keys")
	fmt.Fprintln(w, "  ship make:event <TypeName> [--force]                                                                                                        Generate a domain event type")
	fmt.Fprintln(w, "  ship make:schedule <Name> --cron \"<expr>\"                                                                                                   Insert a scheduled job entry")
	fmt.Fprintln(w, "  ship make:command <Name>                                                                                                                     Generate an app CLI command")
	fmt.Fprintln(w, "  ship make:module <Name> [--path modules] [--module-base github.com/leomorpho/goship-modules] [--dry-run] [--force]                     Generate a standalone module scaffold")
}
