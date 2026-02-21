package templates

import "embed"

//go:embed domain/*.tmpl project/*.tmpl
var FS embed.FS
