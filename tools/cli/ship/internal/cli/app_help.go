package ship

import (
	"fmt"
	"io"
)

func printRootHelp(w io.Writer) {
	fmt.Fprintln(w, "ship - GoShip CLI")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  ship new <app> [--module <module-path>] [--dry-run] [--force]")
	fmt.Fprintln(w, "  ship dev [worker|all] [--worker|--all]")
	fmt.Fprintln(w, "  ship check")
	fmt.Fprintln(w, "  ship doctor")
	fmt.Fprintln(w, "  ship agent:<setup|check>              (or ship agent for help)")
	fmt.Fprintln(w, "  ship upgrade --to <version> [--dry-run]")
	fmt.Fprintln(w, "  ship test [--integration]")
	fmt.Fprintln(w, "  ship db:<create|make|migrate|status|reset|drop|rollback|seed>  (or ship db for help)")
	fmt.Fprintln(w, "  ship infra:<up|down>                  (or ship infra for help)")
	fmt.Fprintln(w, "  ship templ <generate>")
	fmt.Fprintln(w, "  ship make:<scaffold|controller|resource|model|module>  (or ship make for help)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  ship new demo")
	fmt.Fprintln(w, "  ship dev")
	fmt.Fprintln(w, "  ship check")
	fmt.Fprintln(w, "  ship doctor")
	fmt.Fprintln(w, "  ship agent:setup")
	fmt.Fprintln(w, "  ship agent:status")
	fmt.Fprintln(w, "  ship dev worker")
	fmt.Fprintln(w, "  ship dev --all")
	fmt.Fprintln(w, "  ship test --integration")
	fmt.Fprintln(w, "  ship upgrade --to v0.27.1")
	fmt.Fprintln(w, "  ship db:create")
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

func printDevHelp(w io.Writer) {
	fmt.Fprintln(w, "ship dev commands:")
	fmt.Fprintln(w, "  ship dev")
	fmt.Fprintln(w, "  ship dev worker")
	fmt.Fprintln(w, "  ship dev all")
	fmt.Fprintln(w, "  ship dev --worker")
	fmt.Fprintln(w, "  ship dev --all")
	fmt.Fprintln(w, "  (default runs web; use --all to run web + worker concurrently)")
}

func printDBHelp(w io.Writer) {
	fmt.Fprintln(w, "ship db commands:")
	fmt.Fprintln(w, "  ship db:create [--dry-run]")
	fmt.Fprintln(w, "  ship db:make <migration_name>")
	fmt.Fprintln(w, "  ship db:migrate")
	fmt.Fprintln(w, "  ship db:status")
	fmt.Fprintln(w, "  ship db:reset [--seed] [--force] [--yes] [--dry-run]")
	fmt.Fprintln(w, "  ship db:drop [--force] [--yes] [--dry-run]")
	fmt.Fprintln(w, "  ship db:rollback [amount]")
	fmt.Fprintln(w, "  ship db:seed")
}

func printInfraHelp(w io.Writer) {
	fmt.Fprintln(w, "ship infra commands:")
	fmt.Fprintln(w, "  ship infra:up")
	fmt.Fprintln(w, "  ship infra:down")
}

func printTestHelp(w io.Writer) {
	fmt.Fprintln(w, "ship test commands:")
	fmt.Fprintln(w, "  ship test")
	fmt.Fprintln(w, "  ship test --integration")
}

func printCheckHelp(w io.Writer) {
	fmt.Fprintln(w, "ship check commands:")
	fmt.Fprintln(w, "  ship check")
}

func printDoctorHelp(w io.Writer) {
	fmt.Fprintln(w, "ship doctor commands:")
	fmt.Fprintln(w, "  ship doctor")
	fmt.Fprintln(w, "  (validates canonical app structure and LLM/DX conventions)")
}

func printUpgradeHelp(w io.Writer) {
	fmt.Fprintln(w, "ship upgrade commands:")
	fmt.Fprintln(w, "  ship upgrade --to <version> [--dry-run]")
	fmt.Fprintln(w, "  (currently upgrades atlas pin only; no auto-latest)")
}

func printTemplHelp(w io.Writer) {
	fmt.Fprintln(w, "ship templ commands:")
	fmt.Fprintln(w, "  ship templ generate [--path <dir>] [--file <file.templ>]")
	fmt.Fprintln(w, "    (generated files are moved to a child gen/ directory per templ package)")
}

func printMakeHelp(w io.Writer) {
	fmt.Fprintln(w, "ship make commands:")
	fmt.Fprintln(w, "  ship make:scaffold <Name> [fields...] [--path apps/site] [--views templ|none] [--auth public|auth] [--api] [--migrate] [--dry-run] [--force]")
	fmt.Fprintln(w, "  ship make:controller <Name|NameController> [--actions index,show,create,update,destroy] [--auth public|auth] [--wire]")
	fmt.Fprintln(w, "  ship make:resource <name> [--path apps/site] [--auth public|auth] [--views templ|none] [--wire] [--dry-run]")
	fmt.Fprintln(w, "  ship make:model <Name> [fields...]")
	fmt.Fprintln(w, "  ship make:module <Name> [--path modules] [--module-base github.com/leomorpho/goship-modules] [--dry-run] [--force]")
}
