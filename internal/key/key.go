package key

import (
	crand "crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
	"io"
)

func NewServerKey() ServerPrivate {
	_, key, err := box.GenerateKey(crand.Reader)
	if err != nil {
		panic(fmt.Sprintf("unable create new key: %v", err))
	}
	return ServerPrivate{k: *key}
}

func ParsePrivateKey(key string) (*ServerPrivate, error) {
	k := new([32]byte)
	err := parseHex(k[:], key)
	if err != nil {
		return nil, err
	}
	return &ServerPrivate{k: *k}, nil
}

func ParsePublicKey(key string) (*ServerPublic, error) {
	k := new([32]byte)
	err := parseHex(k[:], key)
	if err != nil {
		return nil, err
	}
	return &ServerPublic{k: *k}, nil
}

func parseHex(out []byte, v string) error {
	in := []byte(v)

	if want := len(out) * 2; len(in) != want {
		return fmt.Errorf("key hex has the wrong size, got %d want %d", len(in), want)
	}

	_, err := hex.Decode(out[:], in)
	if err != nil {
		return err
	}

	return nil
}

type ServerPrivate struct {
	k [32]byte
}

type ServerPublic struct {
	k [32]byte
}

func (k ServerPrivate) Public() ServerPublic {
	var ret ServerPublic
	curve25519.ScalarBaseMult(&ret.k, &k.k)
	return ret
}

func (k ServerPrivate) Equal(other ServerPrivate) bool {
	return subtle.ConstantTimeCompare(k.k[:], other.k[:]) == 1
}

func (k ServerPrivate) IsZero() bool {
	return k.Equal(ServerPrivate{})
}

func (k ServerPrivate) Seal(cleartext []byte) (ciphertext []byte) {
	if k.IsZero() {
		panic("can't seal with zero keys")
	}
	var nonce [24]byte
	rand(nonce[:])
	p := k.Public()
	return box.Seal(nonce[:], cleartext, &nonce, &p.k, &k.k)
}

func (k ServerPrivate) Open(ciphertext []byte) (cleartext []byte, ok bool) {
	if k.IsZero() {
		panic("can't open with zero keys")
	}
	if len(ciphertext) < 24 {
		return nil, false
	}
	var nonce [24]byte
	copy(nonce[:], ciphertext)
	p := k.Public()
	return box.Open(nil, ciphertext[len(nonce):], &nonce, &p.k, &k.k)
}

func (k ServerPrivate) String() string {
	return hex.EncodeToString(k.k[:])
}

func (k ServerPublic) Equal(other ServerPublic) bool {
	return subtle.ConstantTimeCompare(k.k[:], other.k[:]) == 1
}

func (k ServerPublic) IsZero() bool {
	return k.Equal(ServerPublic{})
}

func (k ServerPublic) String() string {
	return hex.EncodeToString(k.k[:])
}

func rand(b []byte) {
	if _, err := io.ReadFull(crand.Reader, b[:]); err != nil {
		panic(fmt.Sprintf("unable to read random bytes from OS: %v", err))
	}
}
