//go:build windows

package main

import (
	"reflect"
	"unsafe"

	"github.com/jchv/go-webview2"
	"github.com/jchv/go-webview2/pkg/edge"
)

type webviewSettingsAPI interface {
	GetSettings() (*edge.ICoreWebViewSettings, error)
}

func lockStandaloneWebView(w webview2.WebView) {
	if w == nil {
		return
	}
	rv := reflect.ValueOf(w)
	if rv.Kind() == reflect.Interface {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	f := rv.FieldByName("browser")
	if !f.IsValid() {
		return
	}
	browser := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem().Interface()
	api, ok := browser.(webviewSettingsAPI)
	if !ok {
		return
	}
	settings, err := api.GetSettings()
	if err != nil || settings == nil {
		return
	}
	_ = settings.PutIsZoomControlEnabled(false)
	_ = settings.PutIsPinchZoomEnabled(false)
	_ = settings.PutAreBrowserAcceleratorKeysEnabled(false)
	_ = settings.PutIsStatusBarEnabled(false)
	_ = settings.PutIsSwipeNavigationEnabled(false)
	_ = settings.PutAreDefaultContextMenusEnabled(false)
	_ = settings.PutAreDevToolsEnabled(false)
}
