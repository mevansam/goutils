package rest

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"strconv"
	"strings"
	"sync"

	crypto_rand "crypto/rand"
	math_rand "math/rand"

	"github.com/mevansam/goutils/logger"
	"github.com/minio/highwayhash"
)

type AuthToken struct {
	authCrypt AuthCrypt

	// indicates if token is an 
	// authenticated request token
	isRequestToken bool

	// hash of the encrypted payload 
	// associated with this token
	hashKey         []byte
	payloadChecksum string

	// validates that the request token was validated
	// by the service
	requestValue1, 
	requestValue2, 
	expectedResponseValue int
}

// an encrypted payload that is hashed for 
// verification on request and response
type encryptedPayload struct {
	Payload string `json:"payload,omitempty"`
}

// creates an authenticated token to send with a request
func NewAuthToken(authCrypt AuthCrypt) (*AuthToken, error) {

	var (
		err error
	)

	authToken := &AuthToken{
		authCrypt: authCrypt,

		requestValue1: math_rand.Intn(65535),
		requestValue2: math_rand.Intn(65535),

		isRequestToken: true,
	}
	authToken.expectedResponseValue = authToken.requestValue1 ^ authToken.requestValue2

	authToken.hashKey = make([]byte, 32)
	if _, err = io.ReadFull(crypto_rand.Reader, authToken.hashKey); err != nil {
		return nil, err
	}

	return authToken, nil
}

// creates and authenticated token to send with a response
func NewValidatedResponseToken(requestAuthToken string, authCrypt AuthCrypt) (*AuthToken, error) {

	var (
		err error

		requestToken string
	)

	authToken := &AuthToken{
		authCrypt: authCrypt,
		isRequestToken: false,
	}

	crypt, cryptLock := authCrypt.Crypt()
	cryptLock.Lock()
	defer cryptLock.Unlock()

	if requestToken, err = crypt.DecryptB64(requestAuthToken); err != nil {
		return nil, err
	}
	logger.TraceMessage(
		"NewValidatedResponseToken: Creating validated response token for request token '%s'", 
		requestToken)

	tokenParts := strings.Split(requestToken, "|")
	if len(tokenParts) != 5 || tokenParts[0] != authCrypt.AuthTokenKey() {
		return nil, fmt.Errorf("invalid request token")
	}
	if authToken.hashKey, err = hex.DecodeString(tokenParts[1]); err != nil {
		return nil, fmt.Errorf("invalid request token. error parsing hash key: %s", err.Error())
	}
	if authToken.payloadChecksum = tokenParts[2]; err != nil {
		return nil, fmt.Errorf("invalid request token. error parsing hash key: %s", err.Error())
	}
	if authToken.requestValue1, err = strconv.Atoi(tokenParts[3]); err != nil {
		return nil, fmt.Errorf("invalid request token. error parsing request value1: %s", err.Error())
	}
	if authToken.requestValue2, err = strconv.Atoi(tokenParts[4]); err != nil {
		return nil, fmt.Errorf("invalid request token. error parsing request value1: %s", err.Error())
	}
	authToken.expectedResponseValue = authToken.requestValue1 ^ authToken.requestValue2
	return authToken, nil
}

func (t *AuthToken) GetEncryptedToken() (string, error) {

	var (
		token strings.Builder
	)

	if t.authCrypt.IsAuthenticated() {
		crypt, cryptLock := t.authCrypt.Crypt()
		cryptLock.Lock()
		defer cryptLock.Unlock()
	
		if t.isRequestToken {
			token.WriteString(t.authCrypt.AuthTokenKey())
			token.WriteByte('|')
			token.WriteString(hex.EncodeToString(t.hashKey))
			token.WriteByte('|')
			token.WriteString(t.payloadChecksum)
			token.WriteByte('|')
			token.WriteString(strconv.Itoa(t.requestValue1))
			token.WriteByte('|')
			token.WriteString(strconv.Itoa(t.requestValue2))	
		} else {
			token.WriteString(t.authCrypt.AuthTokenKey())
			token.WriteByte('|')
			token.WriteString(t.payloadChecksum)
			token.WriteByte('|')
			token.WriteString(strconv.Itoa(t.expectedResponseValue))
		}
	
		return crypt.EncryptB64(token.String())	
	}
	return "", fmt.Errorf("not authenticated")
}

func (t *AuthToken) IsTokenResponseValid(authTokenResponse string) (bool, error) {

	var (
		err error

		respToken string
		valResp   int
	)
	
	if t.isRequestToken {
		crypt, cryptLock := t.authCrypt.Crypt()
		cryptLock.Lock()
		defer cryptLock.Unlock()
	
		if respToken, err = crypt.DecryptB64(authTokenResponse); err != nil {
			return false, err
		}
		logger.TraceMessage("AuthToken.IsTokenResponseValid: Validating response token '%s'", respToken)
	
		tokenParts := strings.Split(respToken, "|")
		if len(tokenParts) == 3 && tokenParts[0] == t.authCrypt.AuthTokenKey() {
			if valResp, err = strconv.Atoi(tokenParts[2]); err != nil {
				return false, err
			}
			t.payloadChecksum = tokenParts[1]
			return valResp == t.expectedResponseValue, nil
		}	
		return false, nil
	} else {
		return false, fmt.Errorf("invalid token test for response token")
	}
}

// encrypts a given payload with the auth tokens auth crypt
func (t *AuthToken) EncryptPayload(payload io.Reader) (io.Reader, error) {

	var (
		err error

		waitForBodyRead sync.WaitGroup

		hash                hash.Hash
		body, encryptedBody []byte
	)

	if hash, err = highwayhash.New(t.hashKey); err != nil {
		return nil, err
	}

	// load payload
	waitForBodyRead.Add(1)
	readerHash, writerHash := io.Pipe()
	readerBody, writerPayload := io.Pipe()

	go func() {
		defer func() {
			writerHash.Close()
			writerPayload.Close()
		}()

		writer := io.MultiWriter(writerHash, writerPayload)
		if _, err := io.Copy(writer, payload); err != nil {
			logger.TraceMessage(
				"AuthToken.EncryptPayload: ERROR! Failed to copy payload for hashing and encryption: %s", 
				err.Error())
		}
	}()
	go func() {
		defer waitForBodyRead.Done()

		// read payload content concurrently with hashing of payload content
		if body, err = io.ReadAll(readerBody); err != nil {
			logger.TraceMessage(
				"AuthToken.EncryptPayload: ERROR! Failed to read body to encrypt: %s", 
				err.Error())
		}
	}()

	// create checksum of payload content
	if _, err = io.Copy(hash, readerHash); err != nil {
		return nil, err
	}
	t.payloadChecksum = hex.EncodeToString(hash.Sum(nil))

	// encrypt payload content
	crypt, cryptLock := t.authCrypt.Crypt()
	cryptLock.Lock()
	defer cryptLock.Unlock()

	waitForBodyRead.Wait()
	if encryptedBody, err = crypt.Encrypt(body); err != nil {
		return nil, err
	}

	encryptedPayload := &encryptedPayload{
		Payload: base64.StdEncoding.EncodeToString(encryptedBody),
	}
	payloadReader, payloadWriter := io.Pipe()
	go func() {
		defer payloadWriter.Close()
		if err := json.NewEncoder(payloadWriter).Encode(encryptedPayload); err != nil {
			logger.TraceMessage(
				"AuthToken.EncryptPayload: ERROR! Failed to encode JSON with encrypted payload: %s", 
				err.Error())
		}
	}()
	return payloadReader, nil
}

// decrypts a given payload with the auth tokens auth crypt
func (t *AuthToken) DecryptPayload(body io.Reader) (io.ReadCloser, error) {

	var (
		err error

		hash hash.Hash

		encryptedPayload encryptedPayload
		decodedBody, 
		decryptedBody,
		payload []byte

		waitForPayloadRead sync.WaitGroup
	)

	if hash, err = highwayhash.New(t.hashKey); err != nil {
		return nil, err
	}

	// unmarshal JSON containing encrypted payload
	if err = json.NewDecoder(body).Decode(&encryptedPayload); err != nil {
		return nil, err
	}

	// decrypt payload
	crypt, cryptLock := t.authCrypt.Crypt()
	cryptLock.Lock()
	defer cryptLock.Unlock()

	if decodedBody, err = base64.StdEncoding.DecodeString(encryptedPayload.Payload); err != nil {
		return nil, err
	}
	if decryptedBody, err = crypt.Decrypt(decodedBody); err != nil {
		return nil, err
	}

	// load body
	waitForPayloadRead.Add(1)
	readerHash, writerHash := io.Pipe()
	readerPayload, writerBody := io.Pipe()

	go func() {
		defer func() {
			writerHash.Close()
			writerBody.Close()
		}()

		writer := io.MultiWriter(writerHash, writerBody)
		if _, err := io.Copy(writer, bytes.NewReader(decryptedBody)); err != nil {
			logger.TraceMessage(
				"AuthToken.DecryptPayload: ERROR! Failed to copy payload for hashing and decryption: %s", 
				err.Error())
		}
	}()
	go func() {
		defer waitForPayloadRead.Done()

		// read payload content concurrently with hashing of payload content
		if payload, err = io.ReadAll(readerPayload); err != nil {
			logger.TraceMessage(
				"AuthToken.DecryptPayload: ERROR! Failed to read decrypted payload: %s", 
				err.Error())
		}
	}()

	// create checksum of payload content
	if _, err = io.Copy(hash, readerHash); err != nil {
		return nil, err
	}
	if hex.EncodeToString(hash.Sum(nil)) != t.payloadChecksum {
		return nil, fmt.Errorf("received payload corrupted")
	}

	waitForPayloadRead.Wait()
	return io.NopCloser(bytes.NewReader(payload)), nil
}
