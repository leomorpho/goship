package starter

import "embed"

// Files contains the starter scaffold template used by `ship new`.
//
//go:embed testdata/scaffold/**
var Files embed.FS
