package vault

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"filippo.io/age"

	"github.com/david/key-traveler/internal/paths"
)

// EncryptToFile encrypts plaintext to dst, recipients is a list of age pubkey strings.
func EncryptToFile(dst string, plaintext []byte, recipients []string) error {
	rcpts, err := parseRecipients(recipients)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, rcpts...)
	if err != nil {
		return err
	}
	if _, err := w.Write(plaintext); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return paths.WriteAtomic(dst, buf.Bytes(), 0o644)
}

// DecryptFile decrypts src with the given identity and returns the plaintext.
func DecryptFile(src string, id age.Identity) ([]byte, error) {
	f, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return decryptReader(f, id)
}

func decryptReader(r io.Reader, id age.Identity) ([]byte, error) {
	dr, err := age.Decrypt(r, id)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return io.ReadAll(dr)
}

// ReEncrypt rewrites src in-place, decrypting with id and re-encrypting for
// the new list of recipients. Used when a host is enrolled or revoked.
func ReEncrypt(src string, id age.Identity, recipients []string) error {
	plaintext, err := DecryptFile(src, id)
	if err != nil {
		return fmt.Errorf("%s: %w", src, err)
	}
	return EncryptToFile(src, plaintext, recipients)
}

func parseRecipients(list []string) ([]age.Recipient, error) {
	if len(list) == 0 {
		return nil, fmt.Errorf("no recipients provided")
	}
	out := make([]age.Recipient, 0, len(list))
	for _, s := range list {
		r, err := age.ParseX25519Recipient(s)
		if err != nil {
			return nil, fmt.Errorf("parsing recipient %q: %w", s, err)
		}
		out = append(out, r)
	}
	return out, nil
}
