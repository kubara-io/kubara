package catalog

import (
	"embed"
	"io/fs"
)

const builtInRootDirectory = "built-in"

//go:embed all:built-in
var builtInFS embed.FS

func BuiltInFS() fs.FS {
	return builtInFS
}
