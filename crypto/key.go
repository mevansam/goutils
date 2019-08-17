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

func KeyFromPassphrase(passphrase string, xor int64) []byte {

	key := sha256.Sum256([]byte(passphrase))
	if xor != int64(0) {
		// XOR 8 bytes of key with given value to scramble
		// further the key generated from the passphrase to
		// prevent attacks that creating keys using well
		// known passphrases

		xorBytes := make([]byte, 8)
		for i := 0; i < 8; i++ {
			xorBytes[i] = byte(xor >> uint(i*8))
		}
		xorKey := make([]byte, 32)
		for i := 0; i < 32; i++ {
			xorKey[i] = key[i] ^ xorBytes[i%8]
		}
		return xorKey

	} else {
		return key[0:32]
	}
}
