package hash

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// MD5File returns the hex-encoded SHA-256 of a file's contents.
// Named MD5File for historical reasons (manifest JSON keys are "md5").
func MD5File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// MD5Bytes returns the hex-encoded SHA-256 of a byte slice.
// Named MD5Bytes for historical reasons (manifest JSON keys are "md5").
func MD5Bytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// IsBinary returns true if the file appears to be binary. Heuristic: NUL byte
// in the first 8 KiB, matching what git and most diff tools use.
func IsBinary(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	buf := make([]byte, 8192)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}
	return bytes.IndexByte(buf[:n], 0) >= 0, nil
}
