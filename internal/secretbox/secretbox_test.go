package secretbox

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestSealOpenRoundTrip(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	box, err := New(key)
	if err != nil {
		t.Fatal(err)
	}

	sealed, err := box.Seal("api-secret")
	if err != nil {
		t.Fatal(err)
	}
	if !IsCiphertext(sealed) {
		t.Fatalf("sealed value does not have ciphertext prefix: %q", sealed)
	}
	if strings.Contains(sealed, "api-secret") {
		t.Fatalf("sealed value leaks plaintext: %q", sealed)
	}

	plain, err := box.Open(sealed)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "api-secret" {
		t.Fatalf("plain = %q", plain)
	}
}

func TestRejectsInvalidKey(t *testing.T) {
	if _, err := New("short"); err == nil {
		t.Fatal("expected invalid key error")
	}
}
