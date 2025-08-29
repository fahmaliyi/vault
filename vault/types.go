package vault

import "errors"

const (
	MasterKeyLen = 32
	FEKLen       = 32
	NonceLen     = 24
	Magic        = "GVLT"
	Version      = 0x01
)

var (
	ErrLocked     = errors.New("vault: locked")
	ErrCorrupt    = errors.New("vault: corrupt file")
	ErrAuthFailed = errors.New("vault: authentication failed")
)

type Entry struct {
	ID       string
	Title    string
	Username string
	Secret   []byte
	Notes    string
}

type KDFParams struct {
	Time, Memory uint32
	Threads      uint8
	Salt         []byte
}

type fileHeader struct {
	Flags        uint16
	KDFAlgo      uint8
	ArgonTime    uint32
	ArgonMemory  uint32
	ArgonThreads uint8
	Salt         []byte
	Nonce        []byte
}
