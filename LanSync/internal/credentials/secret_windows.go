package credentials

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Rule：使用当前 Windows 用户的 DPAPI，不以明文或可移植密钥保存凭证。
func Protect(secret []byte) ([]byte, error) {
	return transform(secret, false)
}

func Unprotect(ciphertext []byte) ([]byte, error) {
	return transform(ciphertext, true)
}

func transform(input []byte, decrypt bool) ([]byte, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("凭证内容为空")
	}
	in := windows.DataBlob{Size: uint32(len(input)), Data: &input[0]}
	var out windows.DataBlob
	var err error
	if decrypt {
		err = windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	} else {
		err = windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	}
	if err != nil {
		return nil, fmt.Errorf("Windows 凭证保护失败：%w", err)
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.Data))))
	return append([]byte(nil), unsafe.Slice(out.Data, int(out.Size))...), nil
}
