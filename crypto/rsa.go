package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/youmark/pkcs8"
)

type RSAKey struct {
	key *rsa.PrivateKey
}

type RSAPublicKey struct {
	key *rsa.PublicKey

	// a key id used to identify the private key
	// required to decrypt ciphertext encrypted
	// using this public key.
	KeyID string
}

// create a new RSA key
func NewRSAKey() (*RSAKey, error) {

	var (
		err error
		key *rsa.PrivateKey
	)

	// create rsa key pair
	if key, err = rsa.GenerateKey(rand.Reader, 4096); err != nil {
		return nil, err
	}
	return &RSAKey{
		key: key,
	}, nil
}

// creates a new RSA key from PEM encoded data
func NewRSAKeyFromPEM(privateKeyPEM string, password []byte) (*RSAKey, error) {
	return newRSAKeyFromPEM([]byte(privateKeyPEM), password)
}

func NewRSAKeyFromFile(pemFilePath string, password []byte) (*RSAKey, error) {

	var (
		err     error
		pemData []byte
	)

	if pemData, err = ioutil.ReadFile(pemFilePath); err != nil {
		return nil, err
	}
	return newRSAKeyFromPEM(pemData, password)
}

func newRSAKeyFromPEM(privateKeyPEM []byte, password []byte) (*RSAKey, error) {

	var (
		err error
		ok  bool

		pk         interface{}
		privateKey *rsa.PrivateKey
	)

	// retrieve private key from pem encoded string
	privateKeyBlock, _ := pem.Decode(privateKeyPEM)
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

	return &RSAKey{
		key: privateKey,
	}, nil
}

// returns the encapsulated public key
func (k *RSAKey) PublicKey() *RSAPublicKey {
	return &RSAPublicKey{
		key: &k.key.PublicKey,
	}
}

// returns the PEM encoded public key
func (k *RSAKey) GetPublicKeyPEM() (string, error) {

	var (
		err error

		publicKey []byte
		publicKeyPEM strings.Builder
	)

	if publicKey, err = x509.MarshalPKIXPublicKey(k.key.Public()); err != nil {
		return "", err
	}
	if err := pem.Encode(
		&publicKeyPEM, 
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKey,
		},
	); err != nil {
		return "", err
	}

	return publicKeyPEM.String(), err
}

// decrypts cipher text encrypted with the public key with the private key
func (k *RSAKey) Decrypt(ciphertext []byte) ([]byte, error) {

	var (
		err error

		plaintext []byte
	)
	
	if plaintext, err = rsa.DecryptOAEP(sha256.New(), rand.Reader, k.key, ciphertext, nil); err != nil {
		return nil, err
	}
	return plaintext, nil
}

// decrypts a base64 encoded encrypted string
func (k *RSAKey) DecryptBase64(ciphertext string) ([]byte, error) {

	var (
		err        error
		decoded [] byte
	)

	if decoded, err = base64.StdEncoding.DecodeString(ciphertext); err != nil {
		return nil, nil
	}
	return k.Decrypt(decoded)
}

// returns the PEM encoded private key
func (k *RSAKey) GetPrivateKeyPEM() (string, error) {
	
	var (
		err error

		privateKey []byte
		privateKeyPEM strings.Builder
	)

	if privateKey, err = x509.MarshalPKCS8PrivateKey(k.key); err  != nil {
		return "", err
	}	
	if err := pem.Encode(
		&privateKeyPEM, 
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privateKey,
		},
	); err != nil {
		return "", err
	}
	return privateKeyPEM.String(), err
}

// returns the PEM encoded private key
func (k *RSAKey) GetEncryptedPrivateKeyPEM(password []byte) (string, error) {
		
	var (
		err error

		privateKey []byte
		privateKeyPEM strings.Builder
	)

	if privateKey, err = pkcs8.MarshalPrivateKey(k.key, password,
		&pkcs8.Opts{
			Cipher: pkcs8.AES256GCM,
			KDFOpts: pkcs8.PBKDF2Opts{
				SaltSize: 16, IterationCount: 512, HMACHash: crypto.SHA256,
			},
		},
	); err  != nil {
		return "", err
	}	
	if err := pem.Encode(
		&privateKeyPEM, 
		&pem.Block{
			Type:  "ENCRYPTED PRIVATE KEY",
			Bytes: privateKey,
		},
	); err != nil {
		return "", err
	}
	return privateKeyPEM.String(), err
}

// creates a new RSA key from PEM encoded data
func NewPublicKeyFromPEM(publicKeyPEM string) (*RSAPublicKey, error) {
	return newPublicKeyFromPEM([]byte(publicKeyPEM))
}

func NewPublicKeyFromFile(pemFilePath string) (*RSAPublicKey, error) {

	var (
		err     error
		pemData []byte
	)

	if pemData, err = ioutil.ReadFile(pemFilePath); err != nil {
		return nil, err
	}
	return newPublicKeyFromPEM(pemData)
}

func newPublicKeyFromPEM(publicKeyPEM []byte) (*RSAPublicKey, error) {

	var (
		err error
		ok  bool

		pk        interface{}
		publicKey *rsa.PublicKey
	)

	// retrieve private key from pem encoded string
	publicKeyBlock, _ := pem.Decode(publicKeyPEM)
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
	return &RSAPublicKey{
		key: publicKey,
	}, nil
}

// sets the key id
func (k *RSAPublicKey) SetKeyID(keyID string) {
	k.KeyID = keyID
}

// encrypts plain text using an RSA public key
func (k *RSAPublicKey) Encrypt(plaintext []byte) ([]byte, error) {

	var (
		err error

		ciphertext []byte
	)

	if ciphertext, err = rsa.EncryptOAEP(sha256.New(), rand.Reader, k.key, plaintext, nil); err != nil {
		return nil, err
	}

	return ciphertext, nil
}

// encrypt plain text and return the cipher text as base64 encoded text
func (k *RSAPublicKey) EncryptBase64(plaintext []byte) (string, error) {

	var (
		err         error
		ciphertext  []byte
		encodeddata string
	)

	if ciphertext, err = k.Encrypt(plaintext); err != nil {
		return "", err
	}
	encodeddata = base64.StdEncoding.EncodeToString([]byte(ciphertext))

	if len(k.KeyID) > 0 {
		return k.KeyID + "|" + encodeddata, nil
	} else {
		return encodeddata, nil
	}
}

// creates a private key, public key pem encoded pair
func CreateRSAKeyPair(password []byte) (string, string, error) {
	
	var (
		err error
		key *RSAKey

		privateKeyPEM, publicKeyPEM string
	)

	// create rsa key	
	if key, err = NewRSAKey(); err != nil {
		return "", "", err
	}
	// get pem encoded public key
	if publicKeyPEM, err = key.GetPublicKeyPEM(); err != nil {
		return "", "", err
	}
	// pem encoded private key
	if password == nil {
		if privateKeyPEM, err = key.GetPrivateKeyPEM(); err != nil {
			return "", "", err
		}
	} else {
		if privateKeyPEM, err = key.GetEncryptedPrivateKeyPEM(password); err != nil {
			return "", "", err
		}
	}

	return privateKeyPEM, publicKeyPEM, err
}
