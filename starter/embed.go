package starter

import "embed"

// Files contains the starter scaffold template used by `ship new`.
//
//go:embed README.md app/** cmd/** config/**
var Files embed.FS
