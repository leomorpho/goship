package contracts

import "github.com/leomorpho/goship/framework/backup"

// Route: POST /managed/backup
type ManagedBackupRequest struct {
	ObjectKey string `json:"object_key"`
}

// Route: POST /managed/restore
type ManagedRestoreRequest struct {
	Manifest backup.Manifest `json:"manifest"`
}
