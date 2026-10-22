//go:build !windows

package credentials

import "fmt"

func Protect([]byte) ([]byte, error) {
	return nil, fmt.Errorf("DeepSeek 密钥安全存储仅支持 Windows")
}

func Unprotect([]byte) ([]byte, error) {
	return nil, fmt.Errorf("DeepSeek 密钥安全存储仅支持 Windows")
}
