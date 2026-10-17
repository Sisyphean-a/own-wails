package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBindingConstructionDoesNotTouchUserData(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "not-created")
	app := NewApp(dir, "127.0.0.1:0", false)
	if app.engine != nil {
		t.Fatal("绑定构造期间初始化了引擎")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("绑定构造期间创建了用户数据")
	}
}
