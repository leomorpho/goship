package migrate

import "embed"

//go:embed migrations/*.sql
var migrationFS embed.FS

func Migrations() embed.FS {
	return migrationFS
}
