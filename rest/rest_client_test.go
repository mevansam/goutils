package rest_test

import (
	"context"

	"github.com/mevansam/goutils/rest"

	test_server "github.com/mevansam/goutils/test/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Rest Client", func() {

	var (
		err error

		testServer *test_server.MockHttpServer
	)

	type responseBody struct {
		Resparg1 string `json:"resparg1,omitempty"`
		Resparg2 string `json:"resparg2,omitempty"`
	}
	type responseError struct {
		Message *string `json:"message,omitempty"`
	}

	requestBody := struct {
		Arg1 string `json:"arg1,omitempty"`
		Arg2 string `json:"arg2,omitempty"`
		Arg3 string `json:"arg3,omitempty"`
	}{
		Arg1: "value1",
		Arg2: "value2",
		Arg3: "value3",
	}

	BeforeEach(func() {

		// start test server
		testServer = test_server.NewMockHttpServer(9096)
		testServer.ExpectCommonHeader("Content-Type", "application/json; charset=utf-8")
		testServer.ExpectCommonHeader("Accept", "application/json; charset=utf-8")
		testServer.Start()
	})

	It("sends a rest post request and receives a good response", func() {

		testServer.PushRequest().
			ExpectPath("/api/a").
			ExpectMethod("POST").
			ExpectHeader("Api-Key", "12345").
			ExpectJSONRequest(restRequest).
			RespondWith(restResponse)

		responseBody := responseBody{}
		responseError := responseError{}
		response := &rest.Response{
			Body: &responseBody,
			Error: &responseError,
		}		

		restApiClient := rest.NewRestApiClient(context.Background(), "http://localhost:9096/api")
		err = restApiClient.WithRequest(
			&rest.Request{
				Path: "/a",
				Headers: rest.NV{
					"Api-Key": "12345",
				},
				Body: &requestBody,
			},
		).DoPost(response)
		Expect(err).ToNot((HaveOccurred()))

		Expect(response.StatusCode).To(Equal(200))
		Expect(responseBody.Resparg1).To(Equal("respvalue1"))
		Expect(responseBody.Resparg2).To(Equal("respvalue2"))
		Expect(responseError.Message).To(BeNil())
	})

	AfterEach(func() {		
		testServer.Stop()
	})
})

const restRequest = `{"arg1":"value1","arg2":"value2","arg3":"value3"}`
const restResponse = `{"resparg1":"respvalue1","resparg2":"respvalue2"}`
