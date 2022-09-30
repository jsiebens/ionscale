package util

import (
	"crypto/rsa"
	"math/rand"
	"time"
)

var entropy *rand.Rand

func init() {
	seed := time.Now().UnixNano()
	source := rand.NewSource(seed)
	entropy = rand.New(source)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func RandUint64(n uint64) uint64 {
	return entropy.Uint64() % n
}

func RandomBytes(size int) ([]byte, error) {
	buf := make([]byte, size)
	if _, err := entropy.Read(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func NewPrivateKey() (*rsa.PrivateKey, string, error) {
	id := RandStringBytes(22)

	privateKey, err := rsa.GenerateKey(entropy, 2048)
	if err != nil {
		return nil, "", err
	}

	return privateKey, id, nil
}
