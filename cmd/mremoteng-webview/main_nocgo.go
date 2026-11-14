//go:build !linux || !cgo

// Without cgo there is no GTK, so the helper only explains itself. The application then
// falls back to the system browser.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("mremoteng-webview (built without cgo, no embedded view)")
		return
	}
	fmt.Fprintln(os.Stderr, "mremoteng-webview: this build has no cgo, so the embedded view is unavailable")
	os.Exit(1)
}
