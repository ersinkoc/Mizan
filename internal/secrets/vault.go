package secrets

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

const (
	fileVersion  = 1
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	keyLen       = 32
	saltLen      = 16
	nonceLen     = 12
)

type Secret struct {
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
	Token      string `json:"token,omitempty"`
}

type Vault struct {
	root string
	rand io.Reader
	now  func() time.Time
}

type envelope struct {
	Version    int       `json:"version"`
	KDF        string    `json:"kdf"`
	Time       uint32    `json:"time"`
	MemoryKiB  uint32    `json:"memory_kib"`
	Threads    uint8     `json:"threads"`
	Salt       string    `json:"salt"`
	Nonce      string    `json:"nonce"`
	Ciphertext string    `json:"ciphertext"`
	UpdatedAt  time.Time `json:"updated_at"`
}

var (
	mkdirAll  = os.MkdirAll
	readFile  = os.ReadFile
	remove    = os.Remove
	rename    = os.Rename
	writeFile = os.WriteFile
)

func New(root string) *Vault {
	return &Vault{root: root, rand: rand.Reader, now: func() time.Time { return time.Now().UTC() }}
}

func DeriveKey(passphrase, salt []byte) []byte {
	return argon2.IDKey(passphrase, salt, argonTime, argonMemory, argonThreads, keyLen)
}

func (v *Vault) Put(ctx context.Context, id string, passphrase []byte, secret Secret) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateID(id); err != nil {
		return err
	}
	if len(passphrase) == 0 {
		return errors.New("passphrase is required")
	}
	plain, _ := json.Marshal(secret)
	salt, err := randomBytes(v.rand, saltLen)
	if err != nil {
		return err
	}
	nonce, err := randomBytes(v.rand, nonceLen)
	if err != nil {
		return err
	}
	aead := newGCM(DeriveKey(passphrase, salt))
	env := envelope{
		Version:    fileVersion,
		KDF:        "argon2id",
		Time:       argonTime,
		MemoryKiB:  argonMemory,
		Threads:    argonThreads,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(aead.Seal(nil, nonce, plain, []byte(id))),
		UpdatedAt:  v.now(),
	}
	data, _ := json.MarshalIndent(env, "", "  ")
	return atomicWrite(v.path(id), data, 0o600)
}

func (v *Vault) Get(ctx context.Context, id string, passphrase []byte) (Secret, error) {
	if err := ctx.Err(); err != nil {
		return Secret{}, err
	}
	if err := validateID(id); err != nil {
		return Secret{}, err
	}
	if len(passphrase) == 0 {
		return Secret{}, errors.New("passphrase is required")
	}
	data, err := readFile(v.path(id))
	if err != nil {
		return Secret{}, err
	}
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return Secret{}, err
	}
	if env.Version != fileVersion || env.KDF != "argon2id" {
		return Secret{}, errors.New("unsupported secret envelope")
	}
	salt, err := base64.StdEncoding.DecodeString(env.Salt)
	if err != nil {
		return Secret{}, err
	}
	nonce, err := base64.StdEncoding.DecodeString(env.Nonce)
	if err != nil {
		return Secret{}, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(env.Ciphertext)
	if err != nil {
		return Secret{}, err
	}
	aead := newGCM(DeriveKey(passphrase, salt))
	if len(nonce) != aead.NonceSize() {
		return Secret{}, errors.New("invalid secret nonce")
	}
	plain, err := aead.Open(nil, nonce, ciphertext, []byte(id))
	if err != nil {
		return Secret{}, errors.New("secret decrypt failed")
	}
	var secret Secret
	if err := json.Unmarshal(plain, &secret); err != nil {
		return Secret{}, err
	}
	return secret, nil
}

func (v *Vault) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateID(id); err != nil {
		return err
	}
	if err := remove(v.path(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func newGCM(key []byte) cipher.AEAD {
	block, _ := aes.NewCipher(key)
	aead, _ := cipher.NewGCM(block)
	return aead
}

func (v *Vault) path(id string) string {
	return filepath.Join(v.root, id+".json")
}

func validateID(id string) error {
	if id == "" {
		return errors.New("secret id is required")
	}
	if id != filepath.Base(id) || strings.ContainsAny(id, `/\`) || strings.Contains(id, "..") {
		return fmt.Errorf("invalid secret id %q", id)
	}
	return nil
}

func randomBytes(r io.Reader, n int) ([]byte, error) {
	data := make([]byte, n)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data, nil
}

func atomicWrite(path string, data []byte, perm os.FileMode) error {
	if err := mkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmpName := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".tmp")
	if err := writeFile(tmpName, data, perm); err != nil {
		return err
	}
	if err := rename(tmpName, path); err != nil {
		_ = remove(tmpName)
		return err
	}
	return nil
}
