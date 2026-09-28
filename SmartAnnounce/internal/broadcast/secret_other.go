//go:build !windows

package broadcast

import "errors"

func protectSecret([]byte) ([]byte, error) {
	return nil, errors.New("本系统不支持 Windows DPAPI，无法保存 API Key")
}

func unprotectSecret([]byte) ([]byte, error) {
	return nil, errors.New("本系统不支持 Windows DPAPI，无法读取 API Key")
}
