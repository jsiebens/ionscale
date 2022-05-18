package token

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jsiebens/ionscale/internal/key"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/mr-tron/base58"
	"strings"
	"time"
)

const (
	nonceLength            = 16
	systemAdminTokenPrefix = "st_"
)

var driftCompensation = time.Minute

type Info struct {
	Nonce        string    `json:"nonce"`
	NonceBytes   []byte    `json:"-"`
	CreationTime time.Time `json:"creation_time"`
}

func IsSystemAdminToken(token string) bool {
	return strings.HasPrefix(token, systemAdminTokenPrefix)
}

func ParseSystemAdminToken(privKey key.ServerPrivate, versionedToken string) (*Info, error) {
	versionedToken = strings.TrimSpace(versionedToken)
	if versionedToken == "" {
		return nil, errors.New("empty token")
	}

	if !strings.HasPrefix(versionedToken, systemAdminTokenPrefix) {
		return nil, errors.New("token has wrong format")
	}
	token := strings.TrimPrefix(versionedToken, systemAdminTokenPrefix)

	marshaledBlob, err := base58.FastBase58Decoding(token)
	if err != nil {
		return nil, fmt.Errorf("error base58-decoding token: %w", err)
	}
	if len(marshaledBlob) == 0 {
		return nil, fmt.Errorf("length zero after base58-decoding token")
	}

	info := new(Info)

	if err := unmarshal(marshaledBlob, info, privKey); err != nil {
		return nil, fmt.Errorf("error unmarshaling token info: %w", err)
	}

	info.NonceBytes, err = base64.RawStdEncoding.DecodeString(info.Nonce)
	if err != nil {
		return nil, fmt.Errorf("error decoding nonce bytes: %w", err)
	}
	if len(info.NonceBytes) != nonceLength {
		return nil, errors.New("nonce has incorrect length, must be 32 bytes")
	}

	if info.CreationTime.IsZero() {
		return nil, errors.New("token creation time is zero")
	}

	if info.CreationTime.After(time.Now().Add(driftCompensation)) {
		return nil, errors.New("token creation time is invalid")
	}

	if info.CreationTime.Before(time.Now().Add(-driftCompensation)) {
		return nil, errors.New("token creation time is expired")
	}

	return info, nil
}

func GenerateSystemAdminToken(privKey key.ServerPrivate) (string, error) {
	b, err := util.RandomBytes(nonceLength)
	if err != nil {
		return "", fmt.Errorf("error generating random bytes for token nonce: %w", err)
	}
	info := &Info{
		Nonce:        base64.RawStdEncoding.EncodeToString(b),
		CreationTime: time.Now(),
	}

	return formatToken(privKey, systemAdminTokenPrefix, info)
}

func formatToken(privKey key.ServerPrivate, prefix string, v interface{}) (string, error) {
	blobInfo, err := marshal(v, privKey)
	if err != nil {
		return "", fmt.Errorf("error encrypting info: %w", err)
	}

	encodedMarshaledBlob := base58.FastBase58Encoding(blobInfo)

	return fmt.Sprintf("%s%s", prefix, encodedMarshaledBlob), nil
}

func marshal(v interface{}, privKey key.ServerPrivate) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return privKey.Seal(b), nil
}

func unmarshal(msg []byte, v interface{}, privateKey key.ServerPrivate) error {
	decrypted, ok := privateKey.Open(msg)
	if !ok {
		return fmt.Errorf("unable to decrypt payload")
	}

	if err := json.Unmarshal(decrypted, v); err != nil {
		return err
	}

	return nil
}
