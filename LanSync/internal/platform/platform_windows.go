package platform

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"unicode/utf16"

	"github.com/getlantern/systray"
	"golang.org/x/sys/windows/registry"
)

var trayOnce sync.Once
var trayDone = make(chan struct{})

func StartTray(icon []byte, ready, show, syncAll, quit func()) {
	go systray.Run(func() {
		systray.SetIcon(icon)
		systray.SetTooltip("局域同步")
		open := systray.AddMenuItem("打开", "")
		syncItem := systray.AddMenuItem("立即同步全部", "")
		systray.AddSeparator()
		exit := systray.AddMenuItem("退出", "")
		ready()
		go func() {
			for {
				select {
				case <-trayDone:
					return
				case <-open.ClickedCh:
					show()
				case <-syncItem.ClickedCh:
					syncAll()
				case <-exit.ClickedCh:
					quit()
					return
				}
			}
		}()
	}, func() {})
}
func StopTray() { trayOnce.Do(func() { close(trayDone); systray.Quit() }) }
func SetAutoStart(enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if !enabled {
		err = key.DeleteValue("LanSync")
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return key.SetStringValue("LanSync", `"`+exe+`"`)
}
func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func encoded(s string) string {
	chars := utf16.Encode([]rune(s))
	b := make([]byte, len(chars)*2)
	for i, c := range chars {
		b[i*2] = byte(c)
		b[i*2+1] = byte(c >> 8)
	}
	return base64.StdEncoding.EncodeToString(b)
}
func AllowLAN() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// Rule：应用路径限定 + TCP + 专用网络 + 本地子网，适应监听端口改变，不放开其他程序。
	script := "$ErrorActionPreference='Stop'; $existing=Get-NetFirewallRule -Name 'LanSync-Private' -ErrorAction SilentlyContinue; if($existing){Remove-NetFirewallRule -Name 'LanSync-Private'}; New-NetFirewallRule -Name 'LanSync-Private' -DisplayName 'LanSync 局域网同步' -Direction Inbound -Action Allow -Program " + psQuote(exe) + " -Protocol TCP -Profile Private -RemoteAddress LocalSubnet | Out-Null; $discovery=Get-NetFirewallRule -Name 'LanSync-Discovery' -ErrorAction SilentlyContinue; if($discovery){Remove-NetFirewallRule -Name 'LanSync-Discovery'}; New-NetFirewallRule -Name 'LanSync-Discovery' -DisplayName 'LanSync 设备发现' -Direction Inbound -Action Allow -Program " + psQuote(exe) + " -Protocol UDP -LocalPort 5353 -Profile Private -RemoteAddress LocalSubnet | Out-Null"
	outer := "$p=Start-Process powershell.exe -Verb RunAs -Wait -PassThru -ArgumentList '-NoProfile -NonInteractive -EncodedCommand " + encoded(script) + "'; if($p.ExitCode -ne 0){exit $p.ExitCode}"
	output, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", encoded(outer)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("未能完整配置防火墙规则（可能取消授权，请检查系统规则）：%v %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
func OpenFolder(path string) error { return exec.Command("explorer.exe", path).Start() }
