package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

void TrayCreate(const void *iconData, int iconLen);
void TrayRemove(void);
*/
import "C"
import "unsafe"

import wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

var trayApp *App

func setupTray(app *App, icon []byte) {
	trayApp = app
	C.TrayCreate(unsafe.Pointer(&icon[0]), C.int(len(icon)))
}

func teardownTray() {
	C.TrayRemove()
}

//export goTrayShow
func goTrayShow() {
	if trayApp != nil && trayApp.ctx != nil {
		wailsRuntime.WindowShow(trayApp.ctx)
	}
}

//export goTrayQuit
func goTrayQuit() {
	if trayApp != nil && trayApp.ctx != nil {
		wailsRuntime.Quit(trayApp.ctx)
	}
}
