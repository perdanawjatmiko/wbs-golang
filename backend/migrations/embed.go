package migrations

import "embed"

// Files are embedded so CLI execution never depends on its working directory.
//
//go:embed *.sql
var Files embed.FS
