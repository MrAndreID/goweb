package user

import "embed"

// Views menyematkan template CONTENT milik feature user.
//
// Feature hanya menyumbang blok "content" (dan opsional "title"); layout
// global dimiliki lapisan application (lihat internal/application/view).
// Renderer global meng-compose layout + content ini pada saat startup.
//
//go:embed templates/*.html
var Views embed.FS
