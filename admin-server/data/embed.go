package data

import "embed"

// FS embeds all files and directories in data/ into the binary.
//
//go:embed *
var FS embed.FS
