//go:build linux && cgo

// Command mremoteng-webview hosts a WebKit view inside an X11 window owned by the
// application.
//
// The application cannot host a foreign toolkit: Fyne draws into its own window, and
// WebKitGTK needs a GTK main loop. This helper is launched with the window id of the
// area the application created — the same kind of window it gives FreeRDP — and embeds a
// GtkPlug carrying a WebKitWebView into it. Nothing GTK or WebKit is linked into the
// application itself, which keeps the cgo boundary in one small binary.
//
// GTK must use its X11 backend even when the desktop is Wayland: the window belongs to
// X11 (through XWayland), and GtkPlug refuses to embed under the Wayland backend.
package main

/*
#cgo pkg-config: gtk+-3.0 webkit2gtk-4.1
#include <gtk/gtk.h>
#include <gtk/gtkx.h>
#include <webkit2/webkit2.h>

static GtkWidget *plug_view = NULL;

// start creates the plug inside the given window and loads url. It returns a non-zero
// code when GTK cannot even start, which the Go side reports on stderr.
static int start(unsigned long window, const char *url) {
	if (!gtk_init_check(NULL, NULL)) {
		return 1;
	}
	GtkWidget *plug = gtk_plug_new((Window)window);
	if (plug == NULL) {
		return 2;
	}
	gtk_window_set_title(GTK_WINDOW(plug), "mRemoteNG Go");
	plug_view = webkit_web_view_new();
	gtk_container_add(GTK_CONTAINER(plug), plug_view);
	gtk_widget_show_all(plug);
	webkit_web_view_load_uri(WEBKIT_WEB_VIEW(plug_view), url);
	return 0;
}

static void run(void) { gtk_main(); }
*/
import "C"

import (
	"flag"
	"fmt"
	"os"
	"unsafe"
)

var version = "devel"

func main() {
	window := flag.Uint64("window", 0, "X11 window id of the area to embed into")
	url := flag.String("url", "", "address to open")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("mremoteng-webview %s\n", version)
		return
	}
	if *window == 0 || *url == "" {
		fmt.Fprintln(os.Stderr, "mremoteng-webview: -window and -url are required")
		os.Exit(2)
	}
	address := C.CString(*url)
	defer C.free(unsafe.Pointer(address))
	switch code := C.start(C.ulong(*window), address); code {
	case 0:
	case 1:
		fmt.Fprintln(os.Stderr, "mremoteng-webview: the display refused to start")
		os.Exit(1)
	default:
		fmt.Fprintln(os.Stderr, "mremoteng-webview: could not create the embedded view")
		os.Exit(1)
	}
	C.run()
}
