package commands

import (
	"fmt"
	"io"
)

func PrintRootHelp(w io.Writer) {
	fmt.Fprintln(w, "ship - GoShip CLI")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  ship new <app> [--module <module-path>] [--dry-run] [--force]")
	fmt.Fprintln(w, "  ship dev [worker|all] [--worker|--all]")
	fmt.Fprintln(w, "  ship check")
	fmt.Fprintln(w, "  ship doctor [--json]")
	fmt.Fprintln(w, "  ship config:validate [--json]")
	fmt.Fprintln(w, "  ship routes [--json]")
	fmt.Fprintln(w, "  ship describe [--pretty]")
	fmt.Fprintln(w, "  ship verify [--skip-tests] [--json]")
	fmt.Fprintln(w, "  ship agent:<setup|check>              (or ship agent for help)")
	fmt.Fprintln(w, "  ship upgrade --to <version> [--dry-run]")
	fmt.Fprintln(w, "  ship test [--integration]")
	fmt.Fprintln(w, "  ship db:<create|generate|make|migrate|status|reset|drop|rollback|seed>  (or ship db for help)")
	fmt.Fprintln(w, "  ship infra:<up|down>                  (or ship infra for help)")
	fmt.Fprintln(w, "  ship templ <generate>")
	fmt.Fprintln(w, "  ship make:<scaffold|controller|resource|model|module>  (or ship make for help)")
	fmt.Fprintln(w, "  ship module:add <name> [--dry-run]")
	fmt.Fprintln(w, "  ship module:remove <name> [--dry-run]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  ship new demo")
	fmt.Fprintln(w, "  ship dev")
	fmt.Fprintln(w, "  ship check")
	fmt.Fprintln(w, "  ship doctor [--json]")
	fmt.Fprintln(w, "  ship config:validate [--json]")
	fmt.Fprintln(w, "  ship routes [--json]")
	fmt.Fprintln(w, "  ship describe --pretty")
	fmt.Fprintln(w, "  ship verify --skip-tests")
	fmt.Fprintln(w, "  ship agent:setup")
	fmt.Fprintln(w, "  ship agent:status")
	fmt.Fprintln(w, "  ship dev worker")
	fmt.Fprintln(w, "  ship dev --all")
	fmt.Fprintln(w, "  ship test --integration")
	fmt.Fprintln(w, "  ship upgrade --to v3.27.0")
	fmt.Fprintln(w, "  ship db:create")
	fmt.Fprintln(w, "  ship db:generate")
	fmt.Fprintln(w, "  ship db:make add_posts")
	fmt.Fprintln(w, "  ship db:migrate")
	fmt.Fprintln(w, "  ship db:status")
	fmt.Fprintln(w, "  ship db:reset [--seed] [--force] [--yes]")
	fmt.Fprintln(w, "  ship db:drop [--force] [--yes]")
	fmt.Fprintln(w, "  ship db:rollback 1")
	fmt.Fprintln(w, "  ship infra:up")
	fmt.Fprintln(w, "  ship templ generate --path app")
	fmt.Fprintln(w, "  ship make:resource contact")
	fmt.Fprintln(w, "  ship make:model Post title:string")
	fmt.Fprintln(w, "  ship make:module EmailSubscriptions")
}

func PrintTemplHelp(w io.Writer) {
	fmt.Fprintln(w, "ship templ commands:")
	fmt.Fprintln(w, "  ship templ generate [--path <dir>] [--file <file.templ>]")
	fmt.Fprintln(w, "    (generated files are moved to a child gen/ directory per templ package)")
}

func PrintMakeHelp(w io.Writer) {
	fmt.Fprintln(w, "ship make commands:")
	fmt.Fprintln(w, "  ship make:scaffold <Name> [fields...] [--path app] [--views templ|none] [--auth public|auth] [--api] [--migrate] [--dry-run] [--force]")
	fmt.Fprintln(w, "  ship make:controller <Name|NameController> [--actions index,show,create,update,destroy] [--auth public|auth] [--wire]")
	fmt.Fprintln(w, "  ship make:resource <name> [--path app] [--auth public|auth] [--views templ|none] [--wire] [--dry-run]")
	fmt.Fprintln(w, "  ship make:model <Name> [fields...]")
	fmt.Fprintln(w, "  ship make:module <Name> [--path modules] [--module-base github.com/leomorpho/goship-modules] [--dry-run] [--force]")
}
