package vault

import (
	"encoding/json"
	"fmt"
	"os"
)

type plaintextVault struct {
	Entries []Entry
}

type Vault struct {
	Filename string
	KDF      *KDFParams
	FEK      []byte
	Locked   bool
	syncer   Syncer
	data     plaintextVault
}

func NewVault(filename string, kdf *KDFParams) *Vault {
	if kdf == nil {
		kdf = DefaultKDFParams()
	}
	return &Vault{Filename: filename, KDF: kdf}
}

func (v *Vault) SetSyncer(s Syncer) {
	v.syncer = s
}

func (v *Vault) SyncPull() error {
	if v.syncer == nil {
		return fmt.Errorf("no syncer configured")
	}

	return v.syncer.Pull(v.Filename)
}

func (v *Vault) SyncPush() error {
	if v.syncer == nil {
		return fmt.Errorf("no syncer configured")
	}

	return v.syncer.Push(v.Filename)
}

func (v *Vault) Create(passphrase []byte) error {
	if len(v.KDF.Salt) == 0 {
		v.KDF.Salt, _ = randBytes(16)
	}

	fek, err := DeriveFEKFromPassphrase(passphrase, v.KDF, []byte("vault v1"))
	if err != nil {
		return err
	}
	v.FEK = fek
	v.data = plaintextVault{Entries: []Entry{}}

	// Marshal plaintext
	pt, _ := json.Marshal(v.data)

	// AEAD encryption
	nonce, ct, err := AEADSeal(v.FEK, pt, []byte(Magic))
	if err != nil {
		return err
	}

	// Build file header
	header := fileHeader{
		Flags:        0,
		KDFAlgo:      0x01, // Argon2id
		ArgonTime:    v.KDF.Time,
		ArgonMemory:  v.KDF.Memory,
		ArgonThreads: v.KDF.Threads,
		Salt:         v.KDF.Salt,
		Nonce:        nonce,
	}

	hdrBytes, err := encodeHeader(header)
	if err != nil {
		return err
	}

	// Combine header + ciphertext
	raw := append(hdrBytes, ct...)

	return atomicWriteFile(v.Filename, raw, 0600)
}

func (v *Vault) Open(passphrase []byte) error {
	raw, err := os.ReadFile(v.Filename)
	if err != nil {
		return err
	}

	// Decode header
	header, ct, err := decodeHeader(raw)
	if err != nil {
		return ErrCorrupt
	}

	// Restore KDF params from header
	v.KDF.Time = header.ArgonTime
	v.KDF.Memory = header.ArgonMemory
	v.KDF.Threads = header.ArgonThreads
	v.KDF.Salt = header.Salt

	fek, err := DeriveFEKFromPassphrase(passphrase, v.KDF, []byte("vault v1"))
	if err != nil {
		return err
	}
	v.FEK = fek

	pt, err := AEADOpen(v.FEK, header.Nonce, []byte(Magic), ct)
	if err != nil {
		return ErrAuthFailed
	}

	return json.Unmarshal(pt, &v.data)
}

func (v *Vault) Save() error {
	pt, _ := json.Marshal(v.data)
	nonce, ct, err := AEADSeal(v.FEK, pt, []byte(Magic))
	if err != nil {
		return err
	}

	header := fileHeader{
		Flags:        0,
		KDFAlgo:      0x01,
		ArgonTime:    v.KDF.Time,
		ArgonMemory:  v.KDF.Memory,
		ArgonThreads: v.KDF.Threads,
		Salt:         v.KDF.Salt,
		Nonce:        nonce,
	}

	hdrBytes, err := encodeHeader(header)
	if err != nil {
		return err
	}

	raw := append(hdrBytes, ct...)
	return atomicWriteFile(v.Filename, raw, 0600)
}

// CRUD operations
func (v *Vault) List() []Entry { return v.data.Entries }
func (v *Vault) Add(e Entry)   { v.data.Entries = append(v.data.Entries, e) }
func (v *Vault) Get(id string) *Entry {
	for _, e := range v.data.Entries {
		if e.ID == id {
			return &e
		}
	}
	return nil
}
func (v *Vault) Delete(id string) {
	for i, e := range v.data.Entries {
		if e.ID == id {
			v.data.Entries = append(v.data.Entries[:i], v.data.Entries[i+1:]...)
			return
		}
	}
}

// Zero securely wipes a byte slice from memory.
// It calls the package-level Zero function from crypto.go.
func Zero(b []byte) {
	zero(b) // calls vault/crypto.go zero()
}
