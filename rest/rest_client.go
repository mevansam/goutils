package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/mevansam/goutils/logger"
	"github.com/sirupsen/logrus"
)

type RestApiClient struct {	
	ctx context.Context

	url        string
	httpClient *http.Client
}

type Request struct {
	Path    string
	Headers NV
	Params  NV
	Body    interface{}

	client *RestApiClient
}

type Response struct {
	StatusCode int
	Headers    NV

	Body  interface{}
	Error interface{}
}

type NV map[string]string

func NewRestApiClient(ctx context.Context, url string) *RestApiClient {

	return &RestApiClient{
		ctx: ctx,
		url: url,
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

func (c *RestApiClient) WithRequest(request *Request) *Request {
	request.client = c
	return request
}

func (r *Request) DoGet(response *Response) (err error) {
	return r.do("GET", response)
}

func (r *Request) DoPost(response *Response) (err error) {
	return r.do("POST", response)
}

func (r *Request) DoPut(response *Response) (err error) {
	return r.do("PUT", response)
}

func (r *Request) DoDelete(response *Response) (err error) {
	return r.do("DELETE", response)
}

func (r *Request) do(method string, response *Response) (err error) {

	var (
		url strings.Builder
		
		body   []byte
		reader io.Reader
		writer io.WriteCloser

		httpRequest  *http.Request
		httpResponse *http.Response
	)

	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	logger.DebugMessage("Request.DoPost: processing request: #% v", r)
	
	// concatonate client url with request 
	// path to create the complete url
	url.WriteString(r.client.url)
	if strings.HasSuffix(r.client.url, "/") {
		if strings.HasPrefix(r.Path, "/") {
			url.WriteString(r.Path[1:])
		} else {
			url.WriteString(r.Path)
		}
	} else {
		if strings.HasPrefix(r.Path, "/") {
			url.WriteString(r.Path)
		} else {
			url.Write([]byte{ '/' })
			url.WriteString(r.Path)
		}
	}

	if r.Body != nil {
		if logrus.IsLevelEnabled(logrus.TraceLevel) {
			if body, err = json.Marshal(&r.Body); err != nil {
				return err
			}
			reader = bytes.NewReader(body)
		} else {
			reader, writer = io.Pipe()
			go func() {
				defer writer.Close()
				if err = json.NewEncoder(writer).Encode(&r.Body); err != nil {
					panic(err)
				}
			}()	
		}	
	} else {
		reader = nil
	}
	if httpRequest, err = http.NewRequestWithContext(
		r.client.ctx, method, url.String(), reader,
	); err != nil {
		return err
	}

	// add headers
	httpRequest.Header.Set("Content-Type", "application/json; charset=utf-8")
	httpRequest.Header.Set("Accept", "application/json; charset=utf-8")
	for n, v := range r.Headers {
		httpRequest.Header.Set(n, v)
	}

	// add query params
	if len(r.Params) > 0 {
		query := httpRequest.URL.Query()
		for n, v := range r.Params {
			query.Add(n, v)
		}
		httpRequest.URL.RawQuery = query.Encode()	
	}
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		logger.TraceMessage(
			"Request.DoPost: sending post request to:\n  url=%s,\n  headers=%# v,\n  body=%s",
			httpRequest.URL.String(),
			httpRequest.Header,
			string(body),
		)
	}
	if httpResponse, err = r.client.httpClient.Do(httpRequest); err != nil {
		return err
	}
	defer httpResponse.Body.Close()

	response.StatusCode = httpResponse.StatusCode
	response.Headers = make(map[string]string)
	for n, v := range httpResponse.Header {
		if (len(v) > 0) {
			response.Headers[n] = v[0]
		} else {
			response.Headers[n] = ""
		}
	}

	decodeBody := func(r io.Reader, v interface{}) error {
		if logrus.IsLevelEnabled(logrus.TraceLevel) {
			// retrieve response body to output to trace log
			// before unmarshalling to the response body value
			if body, err = ioutil.ReadAll(r); err != nil {
				return err
			}
			return json.NewDecoder(bytes.NewReader(body)).Decode(v)
		} else {
			return json.NewDecoder(r).Decode(v)
		}
	}	
	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusBadRequest {		
		if err = decodeBody(httpResponse.Body, response.Error); err != nil {
			logger.DebugMessage("ERROR: Message body parse failed: %s", err.Error())
		}
		err = fmt.Errorf("api error: %d - %s", httpResponse.StatusCode, httpResponse.Status)
	}	else {
		err = decodeBody(httpResponse.Body, response.Body)
	}
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		logger.TraceMessage(
			"Request.DoPost: received post response:\n  url=%s,\n  status code=%d,\n  status=%s\n  headers=%# v,\n  body=%s",
			httpRequest.URL.String(),
			httpResponse.StatusCode,
			httpResponse.Status,
			httpResponse.Header,
			string(body),
		)
	}
	return err
}

