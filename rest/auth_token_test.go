package rest_test

import (
	"github.com/mevansam/goutils/rest"

	test_mocks "github.com/mevansam/goutils/test/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Auth Token", func() {

	It("Creates and auth token and validates it", func() {

		var (
			err error
	
			mockAuthCrypt rest.AuthCrypt
		)

		mockAuthCrypt, err = test_mocks.NewMockAuthCrypt("some key")
		Expect(err).ToNot(HaveOccurred())
		authToken := rest.NewAuthToken(mockAuthCrypt)
		Expect(authToken).ToNot(BeNil())

		encryptedReqToken, err := authToken.GetEncryptedToken()
		Expect(err).ToNot(HaveOccurred())
		Expect(len(encryptedReqToken)).To(BeNumerically(">", 0))

		respAuthToken, err := rest.NewValidatedResponseToken(encryptedReqToken, mockAuthCrypt)
		Expect(err).ToNot(HaveOccurred())
		encryptedRespToken, err := respAuthToken.GetEncryptedToken()
		Expect(err).ToNot(HaveOccurred())

		isValid, err := authToken.IsTokenResponseValid(encryptedRespToken)
		Expect(err).ToNot(HaveOccurred())
		Expect(isValid).To(BeTrue())
	})
})
