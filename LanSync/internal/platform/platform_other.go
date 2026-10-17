//go:build !windows

package platform

import "fmt"

func StartTray(icon []byte, ready, show, syncAll, quit func()) {}
func StopTray()                                                {}
func SetAutoStart(enabled bool) error                          { return fmt.Errorf("当前仅支持 Windows 开机启动") }
func AllowLAN() error {
	return fmt.Errorf("当前仅支持 Windows 防火墙授权，请在系统中允许本应用")
}
func OpenFolder(path string) error { return fmt.Errorf("当前仅支持 Windows 打开目录") }
