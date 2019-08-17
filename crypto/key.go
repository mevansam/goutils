package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"io"
)

func RandomKey(len int) ([]byte, error) {

	var (
		err error
	)

	key := make([]byte, len)
	if _, err = io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func KeyFromPassphrase(passphrase string) []byte {

	phraseSum := sha256.Sum256([]byte(passphrase))
	return phraseSum[0:32]
}
