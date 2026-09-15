package user

import "embed"

//go:embed templates/*.html
var Views embed.FS
