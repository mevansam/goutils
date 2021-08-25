package rest_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
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

		mockAuthCrypt, err = test_mocks.NewMockAuthCrypt("some key", nil)
		Expect(err).ToNot(HaveOccurred())
		authToken, err := rest.NewAuthToken(mockAuthCrypt)
		Expect(err).ToNot(HaveOccurred())
		Expect(authToken).ToNot(BeNil())

		// get encrypted payload
		r, err := authToken.EncryptPayload(strings.NewReader(testRequestPayload))
		Expect(err).ToNot(HaveOccurred())
		body, err := io.ReadAll(r)
		Expect(err).ToNot(HaveOccurred())

		// parse payload
		encryptedPayload := struct {
			Payload string `json:"payload,omitempty"`
		}{}
		err = json.Unmarshal(body, &encryptedPayload)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(encryptedPayload.Payload)).To(BeNumerically(">", 0))

		encryptedReqToken, err := authToken.GetEncryptedToken()
		Expect(err).ToNot(HaveOccurred())
		Expect(len(encryptedReqToken)).To(BeNumerically(">", 0))

		// validate request token and payload checksum
		respAuthToken, err := rest.NewValidatedResponseToken(encryptedReqToken, mockAuthCrypt)
		Expect(err).ToNot(HaveOccurred())
		r, err = respAuthToken.DecryptPayload(bytes.NewReader(body))
		Expect(err).ToNot(HaveOccurred())
		payload, err := io.ReadAll(r)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(payload)).To(Equal(testRequestPayload))

		// get encrypted response payload
		r, err = respAuthToken.EncryptPayload(strings.NewReader(testResponsePayload))
		Expect(err).ToNot(HaveOccurred())
		body, err = io.ReadAll(r)
		Expect(err).ToNot(HaveOccurred())

		// parse payload
		err = json.Unmarshal(body, &encryptedPayload)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(encryptedPayload.Payload)).To(BeNumerically(">", 0))

		encryptedRespToken, err := respAuthToken.GetEncryptedToken()
		Expect(err).ToNot(HaveOccurred())
		Expect(len(encryptedRespToken)).To(BeNumerically(">", 0))

		// validate response token and payload
		isValid, err := authToken.IsTokenResponseValid(encryptedRespToken)
		Expect(err).ToNot(HaveOccurred())
		Expect(isValid).To(BeTrue())

		r, err = authToken.DecryptPayload(bytes.NewReader(body))
		Expect(err).ToNot(HaveOccurred())
		payload, err = io.ReadAll(r)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(payload)).To(Equal(testResponsePayload))
	})

	It("Creates and auth token and saves it to the gin context", func() {

		var (
			err error
	
			mockAuthCrypt rest.AuthCrypt
		)

		mockAuthCrypt, err = test_mocks.NewMockAuthCrypt("some key", nil)
		Expect(err).ToNot(HaveOccurred())
		authToken, err := rest.NewAuthToken(mockAuthCrypt)
		Expect(err).ToNot(HaveOccurred())
		Expect(authToken).ToNot(BeNil())

		context := &gin.Context{}
		authToken.SetInContext(context)

		payload := struct {
			Val1 string
			Val2 string
		}{
			Val1: "abcd",
			Val2: "efgh",
		}
		render := rest.NewEncryptedRender(context, payload)

		writerMock := &httpResponseWriterMock{
			header: make(map[string][]string),
		}
		render.WriteContentType(writerMock)
		err = render.Render(writerMock)
		Expect(err).ToNot(HaveOccurred())

		Expect(writerMock.header["Content-Type"][0]).To(Equal("application/json; charset=utf-8"))
		Expect(len(writerMock.header["X-Auth-Token-Response"][0])).To(BeNumerically(">", 0))

		context.Request = &http.Request{
			Body: io.NopCloser(bytes.NewReader(writerMock.body)),
		}
		
		actualPayload := struct {
			Val1 string
			Val2 string
		}{}
		err = rest.DecryptPayloadFromContext(context, &actualPayload)
		Expect(err).ToNot(HaveOccurred())
		Expect(reflect.DeepEqual(payload, actualPayload)).To(BeTrue())
	})
})

type httpResponseWriterMock struct {
	header map[string][]string
	body   []byte
}

func (w *httpResponseWriterMock) Header() http.Header {
	return w.header
}

func (w *httpResponseWriterMock) Write(data []byte) (int, error) {
	w.body = data
	Expect(len(data)).To(BeNumerically(">", 0))

	match, err := regexp.MatchString("{\"payload\":\"[=+\\/0-9a-zA-Z]+\"}", string(data))
	Expect(err).ToNot(HaveOccurred())
	Expect(match).To(BeTrue())
	
	return len(data), nil
}

func (w *httpResponseWriterMock) WriteHeader(statusCode int) {
}

const testRequestPayload = `Hey, diddle, diddle, the cat and the fiddle
The cow jumped over the moon
The little dog laughed to see such fun
And the dish ran away with the spoon
Hey, diddle, diddle, the cat and the fiddle
The cow jumped over the moon
The little dog laughed to see such fun
And the dish ran away with the spoon
Hey, diddle, diddle, the cat and the fiddle
The cow jumped over the moon
The little dog laughed to see such fun
And the dish ran away with the spoon`

const testResponsePayload = `Hickory, dickory, dock.
The mouse ran up the clock.
The clock struck one,
The mouse ran down,
Hickory, dickory, dock.`