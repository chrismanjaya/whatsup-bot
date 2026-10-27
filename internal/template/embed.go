package template

import "embed"

// FS holds every message template, embedded into the binary at build time.
//
//go:embed *.tmpl
var FS embed.FS
