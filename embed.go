// Package fluxtube embeds the built web UI so the binary serves it directly.
package fluxtube

import (
	"embed"
	"io/fs"
)

//go:embed all:web/dist
var distEmbed embed.FS

// DistFS returns the embedded web/dist filesystem rooted at its contents.
func DistFS() fs.FS {
	sub, err := fs.Sub(distEmbed, "web/dist")
	if err != nil {
		panic(err)
	}
	return sub
}
