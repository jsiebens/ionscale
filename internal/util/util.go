package util

import (
	"crypto/rand"
	"crypto/rsa"
	"math/big"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			panic(err)
		}
		b[i] = letterBytes[idx.Int64()]
	}
	return string(b)
}

func RandUint64(n uint64) uint64 {
	val, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(err)
	}
	return val.Uint64()
}

func RandomBytes(size int) ([]byte, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func NewPrivateKey() (*rsa.PrivateKey, string, error) {
	id := RandStringBytes(22)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", err
	}

	return privateKey, id, nil
}
