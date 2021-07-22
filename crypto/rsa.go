package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/youmark/pkcs8"
)

// creates a private key, public key pair
func CreateRSAKeyPair(password []byte) (string, string, error) {
	
	var (
		err error
		key *rsa.PrivateKey

		privateKey, publicKey []byte
		privateKeyPEM, publicKeyPEM strings.Builder
	)

	// create rsa key pair
	if key, err = rsa.GenerateKey(rand.Reader, 4096); err != nil {
		return "", "", err
	}
	// pem encoded private key
	if password == nil {
		if privateKey, err = x509.MarshalPKCS8PrivateKey(key); err  != nil {
			return "", "", err
		}	
		if err := pem.Encode(
			&privateKeyPEM, 
			&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: privateKey,
			},
		); err != nil {
			return "", "", err
		}
	} else {
		if privateKey, err = pkcs8.MarshalPrivateKey(key, password,
			&pkcs8.Opts{
				Cipher: pkcs8.AES256GCM,
				KDFOpts: pkcs8.PBKDF2Opts{
					SaltSize: 16, IterationCount: 512, HMACHash: crypto.SHA256,
				},
			},
		); err  != nil {
			return "", "", err
		}	
		if err := pem.Encode(
			&privateKeyPEM, 
			&pem.Block{
				Type:  "ENCRYPTED PRIVATE KEY",
				Bytes: privateKey,
			},
		); err != nil {
			return "", "", err
		}
	}
	// pem encoded public key
	if publicKey, err = x509.MarshalPKIXPublicKey(key.Public()); err != nil {
		return "", "", err
	}
	if err := pem.Encode(
		&publicKeyPEM, 
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKey,
		},
	); err != nil {
		return "", "", err
	}

	return privateKeyPEM.String(), publicKeyPEM.String(), err
}

// unmarshals pem encoded string to rsa private key
func PrivateKeyFromPEM(privateKeyPEM string, password []byte) (*rsa.PrivateKey, error) {

	var (
		err error
		ok  bool

		pk         interface{}
		privateKey *rsa.PrivateKey
	)

	// retrieve private key from pem encoded string
	privateKeyBlock, _ := pem.Decode([]byte(privateKeyPEM))
	if privateKeyBlock.Type == "RSA PRIVATE KEY" {
		// extract rsa private key pem encoded data
		if pk, err = x509.ParsePKCS8PrivateKey(privateKeyBlock.Bytes); err != nil {
			return nil, err
		}
	} else if privateKeyBlock.Type == "ENCRYPTED PRIVATE KEY" {
		// extract encrypted rsa private key pem encoded data
		if pk, err = pkcs8.ParsePKCS8PrivateKey(privateKeyBlock.Bytes, password); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unable to load private key block from pem encoded data")
	}
	if privateKey, ok = pk.(*rsa.PrivateKey); !ok {
		return nil, fmt.Errorf("pem encoded private key was not an rsa private key")
	}
	return privateKey, nil
}

func PublickKeyFromPEM(publicKeyPEM string) (*rsa.PublicKey, error) {

	var (
		err error
		ok  bool

		pk        interface{}
		publicKey *rsa.PublicKey
	)

	// retrieve private key from pem encoded string
	publicKeyBlock, _ := pem.Decode([]byte(publicKeyPEM))
	if publicKeyBlock.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("unable to load public key block from pem encoded data")
	}
	// extract public key from pem encoded data
	if pk, err = x509.ParsePKIXPublicKey(publicKeyBlock.Bytes); err != nil {
		return nil, err
	}
	if publicKey, ok = pk.(*rsa.PublicKey); !ok {
		return nil, fmt.Errorf("pem encoded public key was not an rsa public key")
	}
	return publicKey, nil
}

// encrypts data with public key
func EncryptWithPublicKey(plaintext []byte, pub *rsa.PublicKey) ([]byte, error) {

	var (
		err error

		ciphertext []byte
	)

	if ciphertext, err = rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, plaintext, nil); err != nil {
		return nil, err
	}
	return ciphertext, nil
}

// decrypts data with private key
func DecryptWithPrivateKey(ciphertext []byte, priv *rsa.PrivateKey) ([]byte, error) {

	var (
		err error

		plaintext []byte
	)
	
	if plaintext, err = rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, ciphertext, nil); err != nil {
		return nil, err
	}
	return plaintext, nil
}
