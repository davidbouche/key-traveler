package hash

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// MD5File returns the hex-encoded md5 of a file's contents.
func MD5File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// MD5Bytes returns the hex-encoded md5 of a byte slice.
func MD5Bytes(b []byte) string {
	sum := md5.Sum(b)
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
