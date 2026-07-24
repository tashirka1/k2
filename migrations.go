package k2

import "embed"

//go:embed migrations
var EmbeddedMigrations embed.FS
