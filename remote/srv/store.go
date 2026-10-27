package srv

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/TheEmeraldPig/mobileClaude/remote/mmo"
)

// Characters are kept as one JSON file per account in DataDir/accounts.
//
// Render's free plan wipes the disk whenever the server restarts, so the server also
// hands each game a signed copy of its character (a "backup"). The game sends it back
// when it connects; if the server has lost the character, or has an older one, it
// takes the backup. The signature (HMAC with the server's secret) means a backup can't
// be edited or made up.

type Store struct{ dir string }

func NewStore(dir string) (*Store, error) {
	d := filepath.Join(dir, "accounts")
	if err := os.MkdirAll(d, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: d}, nil
}

func (s *Store) path(acct string) string { return filepath.Join(s.dir, acct+".json") }

func (s *Store) Load(acct string) *mmo.Char {
	b, err := os.ReadFile(s.path(acct))
	if err != nil {
		return nil
	}
	var c mmo.Char
	if json.Unmarshal(b, &c) != nil || c.ID != acct {
		return nil
	}
	c.Normalize()
	return &c
}

func (s *Store) Save(c *mmo.Char) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	tmp := s.path(c.ID) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path(c.ID))
}

// AccountID turns a device's secret key into its account id (the key itself is never stored)
func AccountID(key string) string {
	h := sha256.Sum256([]byte("remote-account:" + key))
	return hex.EncodeToString(h[:16])
}

func validKey(key string) bool {
	if len(key) < 16 || len(key) > 128 {
		return false
	}
	for _, r := range key {
		if r < '!' || r > '~' {
			return false
		}
	}
	return true
}

func signBackup(secret []byte, c *mmo.Char) string {
	b, _ := json.Marshal(c)
	m := hmac.New(sha256.New, secret)
	m.Write(b)
	enc := base64.RawURLEncoding
	return enc.EncodeToString(b) + "." + enc.EncodeToString(m.Sum(nil))
}

func readBackup(secret []byte, s string) (*mmo.Char, error) {
	body, sig, ok := strings.Cut(s, ".")
	if !ok {
		return nil, errors.New("bad backup")
	}
	enc := base64.RawURLEncoding
	b, err1 := enc.DecodeString(body)
	want, err2 := enc.DecodeString(sig)
	if err1 != nil || err2 != nil {
		return nil, errors.New("bad backup")
	}
	m := hmac.New(sha256.New, secret)
	m.Write(b)
	if !hmac.Equal(m.Sum(nil), want) {
		return nil, errors.New("backup signature doesn't match")
	}
	var c mmo.Char
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	c.Normalize()
	return &c, nil
}
