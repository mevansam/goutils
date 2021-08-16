package mocks

import (
	"crypto/rand"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/mevansam/goutils/crypto"
	"github.com/mevansam/goutils/rest"

	. "github.com/onsi/gomega"
)

type MockAuthCrypt struct {
	c *crypto.Crypt
	l sync.Mutex

	k string
}

func NewMockAuthCrypt(key string) (*MockAuthCrypt, error) {

	var (
		err error
	)

	restApiAuth := &MockAuthCrypt{
		k: key,
	}
	encryptionKey := make([]byte, 32)
	if _, err = io.ReadFull(rand.Reader, encryptionKey); err != nil {
		return nil, err
	}
	if restApiAuth.c, err = crypto.NewCrypt(encryptionKey); err != nil {
		return nil, err
	}
	return restApiAuth, nil
}

func (a *MockAuthCrypt) WaitForAuth() bool {
	return true
}

func (a *MockAuthCrypt) IsAuthenticated() bool {
	return true
}

func (a *MockAuthCrypt) Crypt() (*crypto.Crypt, *sync.Mutex) {
	return a.c, &a.l
}

func (a *MockAuthCrypt) AuthTokenKey() string {
	return a.k
}

func HandleAuthHeaders(mockAuthCrypt rest.AuthCrypt, request, response string) (func(w http.ResponseWriter, r *http.Request, body string) *string) {
	
	return func(w http.ResponseWriter, r *http.Request, body string) *string {
		encryptedAuthToken := r.Header["X-Auth-Token"]
		Expect(encryptedAuthToken).NotTo(BeNil())
		Expect(len(encryptedAuthToken)).To(BeNumerically(">", 0))
		
		authRespToken, err := rest.NewValidatedResponseToken(encryptedAuthToken[0], mockAuthCrypt)
		Expect(err).NotTo(HaveOccurred())

		// retrieve decrypted request payload
		payload := []byte{}
		if len(body) > 0 {
			payloadReader, err := authRespToken.DecryptPayload(strings.NewReader(body))
			Expect(err).ToNot(HaveOccurred())
			payload, err = io.ReadAll(payloadReader.(io.Reader))
			Expect(err).ToNot(HaveOccurred())	
		} 
		Expect(string(payload)).To(Equal(request))

		// get encrypted response body
		responseBody := []byte{}
		if len(response) > 0 {
			bodyReader, err := authRespToken.EncryptPayload(strings.NewReader(response))
			Expect(err).ToNot(HaveOccurred())
			responseBody, err = io.ReadAll(bodyReader)
			Expect(err).ToNot(HaveOccurred())	
		}

		encryptedRespAuthToken, err := authRespToken.GetEncryptedToken()
		Expect(err).NotTo(HaveOccurred())
	
		w.Header()["X-Auth-Token-Response"] = []string{ encryptedRespAuthToken }

		if (len(responseBody) > 0) {
			respBody := string(responseBody)
			return &respBody	
		} else {
			return nil
		}
	}
}
