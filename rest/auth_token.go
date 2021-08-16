package rest

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/mevansam/goutils/logger"
)

type AuthToken struct {
	authCrypt AuthCrypt

	// indicates if token is an 
	// authenticated request token
	isRequestToken bool

	// validates that the request token was validated
	// by the service
	requestValue1, 
	requestValue2, 
	expectedResponseValue int
}

// creates an authenticated token to send with a request
func NewAuthToken(authCrypt AuthCrypt) *AuthToken {

	authToken := &AuthToken{
		authCrypt:         authCrypt,
		requestValue1:   rand.Intn(65535),
		requestValue2:   rand.Intn(65535),
		isRequestToken: true,
	}
	authToken.expectedResponseValue = authToken.requestValue1 ^ authToken.requestValue2
	return authToken
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
	if len(tokenParts) != 3 || tokenParts[0] != authCrypt.AuthTokenKey() {
		return nil, fmt.Errorf("invalid request token")
	}
	if authToken.requestValue1, err = strconv.Atoi(tokenParts[1]); err != nil {
		return nil, fmt.Errorf("invalid request token. error parsing request value1: %s", err.Error())
	}
	if authToken.requestValue2, err = strconv.Atoi(tokenParts[2]); err != nil {
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
			token.WriteString(strconv.Itoa(t.requestValue1))
			token.WriteByte('|')
			token.WriteString(strconv.Itoa(t.requestValue2))	
		} else {
			token.WriteString(t.authCrypt.AuthTokenKey())
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
		if len(tokenParts) == 2 && tokenParts[0] == t.authCrypt.AuthTokenKey() {
			if valResp, err = strconv.Atoi(tokenParts[1]); err != nil {
				return false, err
			}
			return valResp == t.expectedResponseValue, nil
		}	
		return false, nil
	} else {
		return false, fmt.Errorf("invalid token test for response token")
	}
}
