package vault

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
func randBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, err
	}
	return b, nil
}

func DefaultKDFParams() *KDFParams { return &KDFParams{Time: 3, Memory: 256 * 1024, Threads: 1} }

func DeriveFEKFromPassphrase(passphrase []byte, params *KDFParams, info []byte) ([]byte, error) {
	master := argon2.IDKey(passphrase, params.Salt, params.Time, params.Memory, params.Threads, MasterKeyLen)
	zero(passphrase)
	h := hkdf.New(sha256.New, master, nil, info)
	fek := make([]byte, FEKLen)
	if _, err := io.ReadFull(h, fek); err != nil {
		zero(master)
		return nil, err
	}
	zero(master)
	return fek, nil
}

func AEADSeal(fek, plaintext, aad []byte) ([]byte, []byte, error) {
	aead, _ := chacha20poly1305.NewX(fek)
	nonce, _ := randBytes(NonceLen)
	ct := aead.Seal(nil, nonce, plaintext, aad)
	return nonce, ct, nil
}

func AEADOpen(fek, nonce, aad, ciphertext []byte) ([]byte, error) {
	aead, _ := chacha20poly1305.NewX(fek)
	return aead.Open(nil, nonce, ciphertext, aad)
}

func encodeHeader(h fileHeader) ([]byte, error) {
	buf := &bytes.Buffer{}

	// Magic
	if _, err := buf.WriteString(Magic); err != nil {
		return nil, err
	}

	// Version
	if err := buf.WriteByte(Version); err != nil {
		return nil, err
	}

	// Flags
	if err := binary.Write(buf, binary.BigEndian, h.Flags); err != nil {
		return nil, err
	}

	// KDF Algo
	if err := buf.WriteByte(h.KDFAlgo); err != nil {
		return nil, err
	}

	// Argon2 params
	if err := binary.Write(buf, binary.BigEndian, h.ArgonTime); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, h.ArgonMemory); err != nil {
		return nil, err
	}
	if err := buf.WriteByte(h.ArgonThreads); err != nil {
		return nil, err
	}

	// Salt
	if len(h.Salt) > 255 {
		return nil, errors.New("salt too long")
	}
	if err := buf.WriteByte(uint8(len(h.Salt))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(h.Salt); err != nil {
		return nil, err
	}

	// Nonce
	if len(h.Nonce) > 255 {
		return nil, errors.New("nonce too long")
	}
	if err := buf.WriteByte(uint8(len(h.Nonce))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(h.Nonce); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decodeHeader(raw []byte) (fileHeader, []byte, error) {
	var h fileHeader
	if len(raw) < 4+1+2+1+4+4+1+1+1 { // minimal header check
		return h, nil, ErrCorrupt
	}

	buf := bytes.NewReader(raw)

	// Magic
	magicBytes := make([]byte, 4)
	if _, err := buf.Read(magicBytes); err != nil {
		return h, nil, err
	}
	if string(magicBytes) != Magic {
		return h, nil, ErrCorrupt
	}

	// Version
	var version byte
	if err := binary.Read(buf, binary.BigEndian, &version); err != nil {
		return h, nil, err
	}
	if version != Version {
		return h, nil, ErrCorrupt
	}

	// Flags
	if err := binary.Read(buf, binary.BigEndian, &h.Flags); err != nil {
		return h, nil, err
	}

	// KDF Algo
	if err := binary.Read(buf, binary.BigEndian, &h.KDFAlgo); err != nil {
		return h, nil, err
	}

	// Argon2 params
	if err := binary.Read(buf, binary.BigEndian, &h.ArgonTime); err != nil {
		return h, nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &h.ArgonMemory); err != nil {
		return h, nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &h.ArgonThreads); err != nil {
		return h, nil, err
	}

	// Salt
	var saltLen uint8
	if err := binary.Read(buf, binary.BigEndian, &saltLen); err != nil {
		return h, nil, err
	}
	h.Salt = make([]byte, saltLen)
	if _, err := buf.Read(h.Salt); err != nil {
		return h, nil, err
	}

	// Nonce
	var nonceLen uint8
	if err := binary.Read(buf, binary.BigEndian, &nonceLen); err != nil {
		return h, nil, err
	}
	h.Nonce = make([]byte, nonceLen)
	if _, err := buf.Read(h.Nonce); err != nil {
		return h, nil, err
	}

	// Remaining is ciphertext
	ct := raw[len(raw)-buf.Len():]

	return h, ct, nil
}

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, "gvlt-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}

	_ = syncDir(dir)
	_ = os.Chmod(path, perm)
	return nil
}

func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}
