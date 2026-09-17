package security

import (
	"strings"
	"testing"
)

func TestEncryptDecryptString_RoundTrip(t *testing.T) {
	plaintext := "PM_Secret_Data"
	password := "strong-password-123"

	ciphertext, err := EncryptString(plaintext, password)
	if err != nil {
		t.Fatalf("EncryptString returned error: %v", err)
	}
	if ciphertext == plaintext {
		t.Fatalf("ciphertext should not equal plaintext")
	}

	t.Logf("ciphertext: %s", ciphertext)

	decrypted, err := DecryptString(ciphertext, password)
	if err != nil {
		t.Fatalf("DecryptString returned error: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("decrypted text mismatch: got %q want %q", decrypted, plaintext)
	}
}

func TestDecryptString_WrongPassword(t *testing.T) {
	plaintext := "PM_Secret_Data"
	password := "correct-password"

	ciphertext, err := EncryptString(plaintext, password)
	if err != nil {
		t.Fatalf("EncryptString returned error: %v", err)
	}

	_, err = DecryptString(ciphertext, "wrong-password")
	if err == nil {
		t.Fatalf("expected decryption error with wrong password")
	}
}

func TestEncryptString_EmptyPassword(t *testing.T) {
	_, err := EncryptString("data", "")
	if err == nil {
		t.Fatalf("expected error for empty password")
	}
	if !strings.Contains(err.Error(), "password") {
		t.Fatalf("expected password-related error, got: %v", err)
	}
}
