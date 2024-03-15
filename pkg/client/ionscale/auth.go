package ionscale

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/99designs/keyring"
	"github.com/jsiebens/ionscale/internal/key"
	"github.com/jsiebens/ionscale/internal/token"
	"os"
	"strconv"
)

const (
	defaultDir string = "~/.ionscale"
)

func LoadClientAuth(addr string, systemAdminKey string) (ClientAuth, error) {
	if systemAdminKey != "" {
		k, err := key.ParsePrivateKey(systemAdminKey)
		if err != nil {
			return nil, fmt.Errorf("invalid system admin key")
		}
		tid := getEnvUint64("IONSCALE_SYSTEM_ADMIN_DEFAULT_TAILNET_ID", 0)
		return systemAdminTokenSession{key: *k, tid: tid}, nil
	}

	ring, err := openKeyring()
	if err != nil {
		return nil, err
	}

	data, err := ring.Get(createKeyName(addr))
	if err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return nil, err
	}

	if errors.Is(err, keyring.ErrKeyNotFound) {
		return Anonymous{}, nil
	}

	var ds defaultSession
	if err := json.Unmarshal(data.Data, &ds); err != nil {
		return nil, err
	}

	return ds, nil
}

func StoreAuthToken(addr, token string, tailnetID uint64) error {
	ring, err := openKeyring()
	if err != nil {
		return err
	}

	ds := defaultSession{
		TK:  token,
		TID: tailnetID,
	}

	data, err := json.Marshal(&ds)
	if err != nil {
		return err
	}

	return ring.Set(keyring.Item{
		Key:  createKeyName(addr),
		Data: data,
	})
}

type ClientAuth interface {
	GetToken() (string, error)
	TailnetID() uint64
}

type defaultSession struct {
	TK  string
	TID uint64
}

func (m defaultSession) GetToken() (string, error) {
	return m.TK, nil
}

func (m defaultSession) TailnetID() uint64 {
	return m.TID
}

type systemAdminTokenSession struct {
	key key.ServerPrivate
	tid uint64
}

func (m systemAdminTokenSession) GetToken() (string, error) {
	return token.GenerateSystemAdminToken(m.key)
}

func (m systemAdminTokenSession) TailnetID() uint64 {
	return m.tid
}

type Anonymous struct {
}

func (m Anonymous) GetToken() (string, error) {
	return "", nil
}

func (m Anonymous) TailnetID() uint64 {
	return 0
}

func openKeyring() (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{
		LibSecretCollectionName: "login",
		PassPrefix:              "ionscale",
		FileDir:                 defaultDir,
		FilePasswordFunc:        keyring.FixedStringPrompt(""),
		AllowedBackends: []keyring.BackendType{
			keyring.FileBackend,
		},
	})
}

func createKeyName(addr string) string {
	sum := md5.Sum([]byte(addr))
	x := hex.EncodeToString(sum[:])
	return fmt.Sprintf("ionscale:%s", x)
}

func getEnvUint64(key string, defaultValue uint64) uint64 {
	v := os.Getenv(key)
	if v != "" {
		vi, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return defaultValue
		}
		return vi
	}
	return defaultValue
}
