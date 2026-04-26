package secrets

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestVaultPutGetDelete(t *testing.T) {
	v := New(t.TempDir())
	v.now = func() time.Time { return time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC) }
	secret := Secret{
		Username:   "root",
		Password:   "s3cret",
		PrivateKey: "private-key",
		Passphrase: "key-pass",
		Token:      "token",
	}
	if err := v.Put(context.Background(), "target_1", []byte("vault-pass"), secret); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.root, "note.txt"), []byte("ignore"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(v.root, "nested.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	ids, err := v.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "target_1" {
		t.Fatalf("ids=%v", ids)
	}
	if data, err := os.ReadFile(v.path("target_1")); err != nil || bytes.Contains(data, []byte("s3cret")) {
		t.Fatalf("vault file leaked plaintext or read failed data=%s err=%v", data, err)
	}
	got, err := v.Get(context.Background(), "target_1", []byte("vault-pass"))
	if err != nil {
		t.Fatal(err)
	}
	if got != secret {
		t.Fatalf("secret=%+v", got)
	}
	if _, err := v.Get(context.Background(), "target_1", []byte("wrong-pass")); err == nil || err.Error() != "secret decrypt failed" {
		t.Fatalf("expected decrypt failure, got %v", err)
	}
	if err := v.Delete(context.Background(), "target_1"); err != nil {
		t.Fatal(err)
	}
	ids, err = v.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("ids after delete=%v", ids)
	}
	if _, err := os.Stat(v.path("target_1")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected deleted secret, got %v", err)
	}
	if err := v.Delete(context.Background(), "target_1"); err != nil {
		t.Fatal(err)
	}
	missingIDs, err := New(filepath.Join(t.TempDir(), "missing")).List(context.Background())
	if err != nil || len(missingIDs) != 0 {
		t.Fatalf("missing list ids=%v err=%v", missingIDs, err)
	}
	rootFile := filepath.Join(t.TempDir(), "root-file")
	if err := os.WriteFile(rootFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(rootFile).List(context.Background()); err == nil {
		t.Fatal("expected root file list error")
	}
}

func TestVaultValidationAndContext(t *testing.T) {
	v := New(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := v.Put(ctx, "target_1", []byte("pass"), Secret{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("put canceled err=%v", err)
	}
	if _, err := v.Get(ctx, "target_1", []byte("pass")); !errors.Is(err, context.Canceled) {
		t.Fatalf("get canceled err=%v", err)
	}
	if err := v.Delete(ctx, "target_1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("delete canceled err=%v", err)
	}
	if _, err := v.List(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("list canceled err=%v", err)
	}
	for _, id := range []string{"", "../x", "a/b", `a\b`, "a..b"} {
		if err := v.Put(context.Background(), id, []byte("pass"), Secret{}); err == nil {
			t.Fatalf("expected put id error for %q", id)
		}
		if _, err := v.Get(context.Background(), id, []byte("pass")); err == nil {
			t.Fatalf("expected get id error for %q", id)
		}
		if err := v.Delete(context.Background(), id); err == nil {
			t.Fatalf("expected delete id error for %q", id)
		}
	}
	if err := v.Put(context.Background(), "target_1", nil, Secret{}); err == nil {
		t.Fatal("expected put passphrase error")
	}
	if _, err := v.Get(context.Background(), "target_1", nil); err == nil {
		t.Fatal("expected get passphrase error")
	}
}

func TestVaultEnvelopeErrors(t *testing.T) {
	root := t.TempDir()
	v := New(root)
	if _, err := v.Get(context.Background(), "missing", []byte("pass")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected missing secret error, got %v", err)
	}
	write := func(id string, env any) {
		t.Helper()
		data, err := json.Marshal(env)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(v.path(id), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(v.path("bad_json"), []byte("{bad json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Get(context.Background(), "bad_json", []byte("pass")); err == nil {
		t.Fatal("expected bad json error")
	}
	write("bad_version", envelope{Version: 99, KDF: "argon2id"})
	if _, err := v.Get(context.Background(), "bad_version", []byte("pass")); err == nil || err.Error() != "unsupported secret envelope" {
		t.Fatalf("expected unsupported envelope, got %v", err)
	}
	write("bad_salt", envelope{Version: fileVersion, KDF: "argon2id", Salt: "!"})
	if _, err := v.Get(context.Background(), "bad_salt", []byte("pass")); err == nil {
		t.Fatal("expected bad salt error")
	}
	write("bad_nonce", envelope{Version: fileVersion, KDF: "argon2id", Salt: base64.StdEncoding.EncodeToString([]byte("salt")), Nonce: "!"})
	if _, err := v.Get(context.Background(), "bad_nonce", []byte("pass")); err == nil {
		t.Fatal("expected bad nonce error")
	}
	write("bad_ciphertext", envelope{Version: fileVersion, KDF: "argon2id", Salt: base64.StdEncoding.EncodeToString([]byte("salt")), Nonce: base64.StdEncoding.EncodeToString([]byte("nonce")), Ciphertext: "!"})
	if _, err := v.Get(context.Background(), "bad_ciphertext", []byte("pass")); err == nil {
		t.Fatal("expected bad ciphertext error")
	}
	write("short_nonce", envelope{Version: fileVersion, KDF: "argon2id", Salt: base64.StdEncoding.EncodeToString([]byte("salt")), Nonce: base64.StdEncoding.EncodeToString([]byte("short")), Ciphertext: base64.StdEncoding.EncodeToString([]byte("cipher"))})
	if _, err := v.Get(context.Background(), "short_nonce", []byte("pass")); err == nil || err.Error() != "invalid secret nonce" {
		t.Fatal("expected short nonce decrypt error")
	}
	salt := []byte("0123456789abcdef")
	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	aead := newGCM(DeriveKey([]byte("pass"), salt))
	write("bad_plaintext", envelope{
		Version:    fileVersion,
		KDF:        "argon2id",
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(aead.Seal(nil, nonce, []byte("{bad json"), []byte("bad_plaintext"))),
	})
	if _, err := v.Get(context.Background(), "bad_plaintext", []byte("pass")); err == nil {
		t.Fatal("expected plaintext json error")
	}
}

func TestVaultRandomAndAtomicWriteErrors(t *testing.T) {
	v := New(t.TempDir())
	v.rand = errReader{}
	if err := v.Put(context.Background(), "target_1", []byte("pass"), Secret{}); err == nil {
		t.Fatal("expected salt random error")
	}
	v.rand = bytes.NewReader(make([]byte, saltLen))
	if err := v.Put(context.Background(), "target_1", []byte("pass"), Secret{}); err == nil {
		t.Fatal("expected nonce random error")
	}
	rootFile := filepath.Join(t.TempDir(), "root-file")
	if err := os.WriteFile(rootFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := New(rootFile).Put(context.Background(), "target_1", []byte("pass"), Secret{}); err == nil {
		t.Fatal("expected root file write error")
	}
	if _, err := randomBytes(bytes.NewReader([]byte{1}), 2); err == nil {
		t.Fatal("expected random read error")
	}
	if got := DeriveKey([]byte("pass"), []byte("salt")); len(got) != keyLen {
		t.Fatalf("key len=%d", len(got))
	}
}

func TestVaultInjectedFileErrors(t *testing.T) {
	oldMkdirAll, oldReadDir, oldReadFile, oldRemove, oldRename, oldStatPath, oldWriteFile := mkdirAll, readDir, readFile, remove, rename, statPath, writeFile
	t.Cleanup(func() {
		mkdirAll, readDir, readFile, remove, rename, statPath, writeFile = oldMkdirAll, oldReadDir, oldReadFile, oldRemove, oldRename, oldStatPath, oldWriteFile
	})
	v := New(t.TempDir())
	mkdirAll = func(string, os.FileMode) error { return errors.New("mkdir failed") }
	if err := v.Put(context.Background(), "target_1", []byte("pass"), Secret{}); err == nil || err.Error() != "mkdir failed" {
		t.Fatalf("expected mkdir error, got %v", err)
	}
	mkdirAll = oldMkdirAll
	writeFile = func(string, []byte, os.FileMode) error { return errors.New("write failed") }
	if err := v.Put(context.Background(), "target_1", []byte("pass"), Secret{}); err == nil || err.Error() != "write failed" {
		t.Fatalf("expected write error, got %v", err)
	}
	writeFile = oldWriteFile
	rename = func(string, string) error { return errors.New("rename failed") }
	if err := v.Put(context.Background(), "target_1", []byte("pass"), Secret{}); err == nil || err.Error() != "rename failed" {
		t.Fatalf("expected rename error, got %v", err)
	}
	rename = oldRename
	readFile = func(string) ([]byte, error) { return nil, errors.New("read failed") }
	if _, err := v.Get(context.Background(), "target_1", []byte("pass")); err == nil || err.Error() != "read failed" {
		t.Fatalf("expected read error, got %v", err)
	}
	readFile = oldReadFile
	remove = func(string) error { return errors.New("remove failed") }
	if err := v.Delete(context.Background(), "target_1"); err == nil || err.Error() != "remove failed" {
		t.Fatalf("expected remove error, got %v", err)
	}
	remove = oldRemove
	statPath = func(string) (os.FileInfo, error) { return nil, errors.New("stat failed") }
	if _, err := v.List(context.Background()); err == nil || err.Error() != "stat failed" {
		t.Fatalf("expected stat error, got %v", err)
	}
	statPath = oldStatPath
	readDir = func(string) ([]os.DirEntry, error) { return nil, os.ErrNotExist }
	ids, err := v.List(context.Background())
	if err != nil || len(ids) != 0 {
		t.Fatalf("expected empty missing list ids=%v err=%v", ids, err)
	}
	readDir = func(string) ([]os.DirEntry, error) { return nil, errors.New("read dir failed") }
	if _, err := v.List(context.Background()); err == nil || err.Error() != "read dir failed" {
		t.Fatalf("expected read dir error, got %v", err)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}
