package js

import "embed"

// FS holds the engine-authored JavaScript files. These are embedded
// in the binary and written to the public directory by press assets.
//
//go:embed *.js
var FS embed.FS
