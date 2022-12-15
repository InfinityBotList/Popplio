// Consumers of other projects who wish to directly use the IBL API should use this package.
//
// client_api is a package that provides a client for the IBL API similar to discordgo etc.
// and can be used by normal users using Go to interact with the API.
package client_api

import (
	"bytes"
	"net/http"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	apiUrl = "https://spider.infinitybots.gg"
)

// Makes a request to the API
func request(method string, path string, jsonP any, headers map[string]string) (*ClientResponse, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	var body []byte
	var err error
	if jsonP != nil {
		body, err = json.Marshal(jsonP)

		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, apiUrl+path, bytes.NewReader(body))

	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	return &ClientResponse{
		Request:  req,
		Response: resp,
	}, nil
}

type ClientResponse struct {
	Request  *http.Request
	Response *http.Response
}

// Returns true if the response is a maintenance response (502, 503, 408)
func (c ClientResponse) IsMaint() bool {
	return c.Response.StatusCode == 502 || c.Response.StatusCode == 503 || c.Response.StatusCode == 408
}

// Unmarshals the response body into the given struct
func (c ClientResponse) Json(v any) error {
	return json.NewDecoder(c.Response.Body).Decode(v)
}

// Returns the retry after header. Is a string
func (c ClientResponse) RetryAfter() string {
	return c.Response.Header.Get("Retry-After")
}

type ClientRequest struct {
	method  string
	path    string
	body    any
	headers map[string]string
}

func NewReq() ClientRequest {
	return ClientRequest{
		headers: make(map[string]string),
	}
}

func (r ClientRequest) Method(method string) ClientRequest {
	r.method = method
	return r
}

func (r ClientRequest) Path(path string) ClientRequest {
	r.path = path
	return r
}

func (r ClientRequest) Json(json any) ClientRequest {
	r.body = json
	return r
}

func (r ClientRequest) Auth(token string) ClientRequest {
	r.headers["Authorization"] = token
	return r
}

func (r ClientRequest) Header(key string, value string) ClientRequest {
	r.headers[key] = value
	return r
}

func (r ClientRequest) Do() (*ClientResponse, error) {
	return request(r.method, r.path, r.body, r.headers)
}
