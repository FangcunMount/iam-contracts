package openapi

import (
	"embed"
	"io/fs"
)

// FS embeds OpenAPI specs under api/rest for serving via Swagger UI.
//
//go:embed rest/*.yaml
var FS embed.FS

// RestFS is a sub filesystem rooted at rest for serving via HTTP.
var RestFS fs.FS

func init() {
	sub, err := fs.Sub(FS, "rest")
	if err != nil {
		panic(err)
	}
	RestFS = sub
}
