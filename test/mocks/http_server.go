package mocks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"sync"

	"github.com/mevansam/goutils/logger"
)

type MockHttpServer struct {
	server     *http.Server
	serverExit sync.WaitGroup

	expectCommonHeaders []nv
	expectRequests      []*request
}

type request struct {
	callbackTest func(r *http.Request, body string) *string

	expectPath        string
	expectMethod      string
	expectHeaders     []nv
	expectQueryArgs   []nv
	expectJSONRequest interface{}
	expectRequestBody *string
	responseBody      *string

	httpError *string
	httpCode  int
}

type nv struct {
	name, value string
}

func NewMockHttpServer(port int) *MockHttpServer {

	ms := MockHttpServer{}
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", ms.mockResponseReflector)

	ms.server = &http.Server{ 
		Addr: fmt.Sprintf(":%d", port),
		Handler: serveMux,
	}

	return &ms
}

func (ms *MockHttpServer) Start() {
	go func() {
		defer ms.serverExit.Done() // let caller know we are done cleaning up
		ms.serverExit.Add(1)

		// always returns error. ErrServerClosed on graceful close
		if err := ms.server.ListenAndServe(); err != http.ErrServerClosed {
				// unexpected error. port in use?
				log.Fatalf("MockServer.Start(): %v", err)
		}
	}()	
}

func (ms *MockHttpServer) Stop() {
	if err := ms.server.Shutdown(context.Background()); err != nil {
		log.Fatalf("MockServer.Stop(): %v", err)
	}
	ms.serverExit.Wait()
}

func (ms *MockHttpServer) ExpectCommonHeader(name, value string) {
	ms.expectCommonHeaders = append(ms.expectCommonHeaders, nv{name, value})
}

func (ms *MockHttpServer) PushRequest() *request {
	request := &request{}
	ms.expectRequests = append(ms.expectRequests, request)
	return request
}

func (ms *MockHttpServer) mockResponseReflector(w http.ResponseWriter, r *http.Request) {

	var (
		err    error
		exists bool	
		value  []string

		buffer       bytes.Buffer
		size         int64
		requestBody  string
		responseBody *string

		requestURL *url.URL

		hasError bool
	)

	logger.TraceMessage("MockServer: request URI: %s", r.RequestURI)
	logger.TraceMessage("MockServer: request Headers: %s", r.Header)

	if size, err = buffer.ReadFrom(r.Body); err != nil {
		http.Error(w, 
			fmt.Sprintf("Error reading request body: %s", err.Error()), 
			http.StatusBadRequest,
		)
		return
	}
	requestBody = buffer.String()
	logger.TraceMessage("MockServer: request Body (%d): %s", size, requestBody)

	// expected request
	if len(ms.expectRequests) == 0 {
		http.Error(w, 
			"Error expected request stack is empty", 
			http.StatusBadRequest,
		)
		return
	}
	expectedRequest := ms.expectRequests[0]
	ms.expectRequests = ms.expectRequests[1:]

	// check path	
	if len(expectedRequest.expectPath) > 0 && expectedRequest.expectPath != r.URL.Path {
		http.Error(w, 
			fmt.Sprintf(
				"Expecting path '%s' but got %s", 
				expectedRequest.expectPath, requestURL.Path, 
			), 
			http.StatusBadRequest,
		)
	}

	// check method
	if len(expectedRequest.expectMethod) > 0 && expectedRequest.expectMethod != r.Method {
		http.Error(w, 
			fmt.Sprintf(
				"Expecting method '%s' but got %s", 
				expectedRequest.expectMethod, r.Method,
			), 
			http.StatusBadRequest,
		)
	}

	// check expected headers
	checkHeaders := func(expectedHeaders []nv) bool {

		for _, header := range expectedHeaders {
			if value, exists = r.Header[header.name]; !exists {
				http.Error(w, 
					fmt.Sprintf("Error expected header is missing: %s", header.name), 
					http.StatusBadRequest,
				)
				return true
			}
			if len(value) == 0 {
				http.Error(w, 
					fmt.Sprintf("Error expected header value was empty: %s", header.name), 
					http.StatusBadRequest,
				)
				return true
			}
			if value[0] != header.value {
				http.Error(w, 
					fmt.Sprintf(
						"Error expected header '%s' value does not match: expected '%s', got '%s'", 
						header.name, header.value, value[0],
					), 
					http.StatusBadRequest,
				)
				return true
			}
		}
		return false
	}

	// common headers
	if hasError = checkHeaders(ms.expectCommonHeaders); hasError {
		return
	}
	// request headers
	if hasError = checkHeaders(expectedRequest.expectHeaders); hasError {
		return
	}

	// request args
	queryArgs := r.URL.Query()
	for _, arg := range expectedRequest.expectQueryArgs {
		if value, exists = queryArgs[arg.name]; !exists || len(value) == 0{
			http.Error(w, 
				fmt.Sprintf(
					"Error expected query arg '%s' is missing", 
					arg.name,
				), 
				http.StatusBadRequest,
			)
			return
		}
		if value[0] != arg.value {
			http.Error(w, 
				fmt.Sprintf(
					"Error expected header '%s' value does not match: expected '%s', got '%s'", 
					arg.name, arg.value, value[0],
				), 
				http.StatusBadRequest,
			)
			return
		}

	}

	// check expected request body
	if expectedRequest.expectJSONRequest != nil {

		var actual interface{}
		if err := json.Unmarshal([]byte(requestBody), &actual); err != nil {
			http.Error(w, 
				fmt.Sprintf(
					"Error parsing JSON request body '%s': %s", 
					requestBody, err.Error(), 
				), 
				http.StatusBadRequest,
			)
			return
		}

		if !reflect.DeepEqual(expectedRequest.expectJSONRequest, actual) {

			http.Error(w, 
				fmt.Sprintf(
					"Error request body: expected '%v', got '%v'", 
					expectedRequest.expectJSONRequest, actual,
				), 
				http.StatusBadRequest,
			)
			return	
		}

	} else if expectedRequest.expectRequestBody != nil && 
		*expectedRequest.expectRequestBody != requestBody {

		http.Error(w, 
			fmt.Sprintf(
				"Error request body: expected '%s', got '%s'", 
				*expectedRequest.expectRequestBody, requestBody,
			), 
			http.StatusBadRequest,
		)
		return
	}

	// callback test
	responseBody = expectedRequest.responseBody
	if (expectedRequest.callbackTest != nil) {
		if respBody := expectedRequest.callbackTest(r, requestBody); respBody != nil {
			responseBody = respBody
		}
	}
	// return response	
	if responseBody != nil {
		if _, err = w.Write([]byte(*responseBody)); err != nil {
			http.Error(w, 
				fmt.Sprintf("Error unable to return response: %s", err.Error()),
				http.StatusBadRequest,
			)
			return
		}	
	}
	// return error
	if expectedRequest.httpError != nil {
		http.Error(w, *expectedRequest.httpError, expectedRequest.httpCode)
	}
}

func (r *request) ExpectPath(path string) *request {
	r.expectPath = path
	return r
}

func (r *request) ExpectMethod(method string) *request {
	r.expectMethod = method
	return r
}

func (r *request) ExpectHeader(name, value string) *request {
	r.expectHeaders = append(r.expectHeaders, nv{name, value})
	return r
}

func (r *request) ExpectQueryArg(name, value string) *request {
	r.expectQueryArgs = append(r.expectQueryArgs, nv{name, value})
	return r
}

func (r *request) ExpectJSONRequest(body string) *request {

	var expected interface{}
	if err := json.Unmarshal([]byte(body), &expected); err != nil {
		log.Fatalf("Error parsing JSON request '%s': %s", body, err.Error())
	}

	r.expectJSONRequest = expected
	return r
}

func (r *request) ExpectRequest(body string) *request {
	r.expectRequestBody = &body
	return r
}

func (r *request) WithCallbackTest(cb func(r *http.Request, body string) *string) *request {
	r.callbackTest = cb
	return r
}

func (r *request) RespondWith(body string) *request {
	r.responseBody = &body
	return r
}

func (r *request) RespondWithError(httpError string, code int) *request {
	r.httpError = &httpError
	r.httpCode = code
	return r
}