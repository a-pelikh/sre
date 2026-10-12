package web

import "embed"

// Files contains the browser client bundled into the server binary.
//
//go:embed index.html app.js styles.css
var Files embed.FS
