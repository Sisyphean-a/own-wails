package credentials

import (
	"bytes"
	"testing"
)

func TestUserProtectedSecret(t *testing.T) {
	secret := []byte("sk-lansync-test-secret")
	ciphertext, err := Protect(secret)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ciphertext, secret) {
		t.Fatal("凭证明文泄露")
	}
	plain, err := Unprotect(ciphertext)
	if err != nil || !bytes.Equal(plain, secret) {
		t.Fatalf("凭证恢复失败：%v", err)
	}
	ciphertext[0] ^= 0xff
	if _, err = Unprotect(ciphertext); err == nil {
		t.Fatal("损坏凭证被接受")
	}
}
