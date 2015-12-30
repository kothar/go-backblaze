// Package backblaze B2 API for Golang
package backblaze // import "gopkg.in/kothar/go-backblaze.v0"

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	b2Host = "https://api.backblaze.com"
	v1     = "/b2api/v1/"
)

// Credentials are the identification required by the Backblaze B2 API
//
// The account ID is a 12-digit hex number that you can get from
// your account page on backblaze.com.
//
// The application key is a 40-digit hex number that you can get from
// your account page on backblaze.com.
type Credentials struct {
	AccountID      string
	ApplicationKey string
}

// B2 implements a B2 API client
type B2 struct {
	Credentials

	// If true, don't retry requests if authorization has expired
	NoRetry bool

	// If true, display debugging information about API calls
	Debug bool

	// State
	host               string
	apiEndpoint        string
	downloadURL        string
	authorizationToken string
	httpClient         http.Client
}

// B2Error encapsulates an error message returned by the B2 API.
//
// Failures to connect to the B2 servers, and networking problems in general can cause errors
type B2Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (e B2Error) Error() string {
	return e.Code + ": " + e.Message
}

// IsFatal returns true if this error represents
// an error which can't be recovered from by retrying
func (e *B2Error) IsFatal() bool {
	switch e.Status {
	case 401:
		return false
	default:
		return true
	}
}

type authorizeAccountResponse struct {
	AccountID          string `json:"accountId"`
	APIEndpoint        string `json:"apiUrl"`
	AuthorizationToken string `json:"authorizationToken"`
	DownloadURL        string `json:"downloadUrl"`
}

// NewB2 creates a new Client for accessing the B2 API.
// The AuthorizeAccount method will be called immediately.
func NewB2(creds Credentials) (*B2, error) {
	c := &B2{
		Credentials: creds,
	}

	// Authorize account
	if err := c.AuthorizeAccount(); err != nil {
		return nil, err
	}

	return c, nil
}

// AuthorizeAccount is used to log in to the B2 API.
func (c *B2) AuthorizeAccount() error {
	if c.host == "" {
		c.host = b2Host
	}

	req, err := http.NewRequest("GET", c.host+v1+"b2_authorize_account", nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.AccountID, c.ApplicationKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	authResponse := &authorizeAccountResponse{}
	if err = c.parseResponse(resp, authResponse); err != nil {
		return err
	}

	// Store token
	c.authorizationToken = authResponse.AuthorizationToken
	c.downloadURL = authResponse.DownloadURL
	c.apiEndpoint = authResponse.APIEndpoint

	return nil
}

// DownloadURL returns the URL prefix needed to construct download links.
// Bucket.FileURL will costruct a full URL for given file names.
func (c *B2) DownloadURL() (string, error) {
	if c.downloadURL == "" {
		if err := c.AuthorizeAccount(); err != nil {
			return "", err
		}
	}
	return c.downloadURL, nil
}

// Create an authorized request using the client's credentials
func (c *B2) authRequest(method, path string, body io.Reader) (*http.Request, error) {

	if c.authorizationToken == "" {
		if c.Debug {
			log.Println("No valid authorization token, re-authorizing client")
		}
		if err := c.AuthorizeAccount(); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", c.authorizationToken)

	if c.Debug {
		log.Printf("authRequest: %s %s\n", method, req.URL)
	}

	return req, nil
}

// Create an authorized GET request
func (c *B2) get(path string) (*http.Response, error) {
	req, err := c.authRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Do(req)
}

// Create an authorized POST request
func (c *B2) post(path string, body io.Reader) (*http.Response, error) {
	req, err := c.authRequest("POST", path, body)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Do(req)
}

// Looks for an error message in the response body and parses it into a
// B2Error object
func (c *B2) parseError(body []byte) error {
	b2err := &B2Error{}
	if json.Unmarshal(body, b2err) != nil {
		return nil
	}
	return b2err
}

// Attempts to parse a response body into the provided result struct
func (c *B2) parseResponse(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if c.Debug {
		log.Printf("Response: %s", body)
	}

	// Check response code
	switch resp.StatusCode {
	case 200: // Response is OK
	case 401:
		c.authorizationToken = ""
		return &B2Error{
			Code:    "UNAUTHORIZED",
			Message: "The account ID is wrong, the account does not have B2 enabled, or the application key is not valid",
			Status:  resp.StatusCode,
		}
	default:
		if err := c.parseError(body); err != nil {
			return err
		}
		return &B2Error{
			Code:    "UNKNOWN",
			Message: "Unrecognised status code",
			Status:  resp.StatusCode,
		}
	}

	return json.Unmarshal(body, result)
}

// Perform a B2 API request with the provided request and response objects
func (c *B2) apiRequest(apiMethod string, request interface{}, response interface{}) error {
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	if c.Debug {
		log.Println("----")
		log.Printf("apiRequest: %s %s", apiMethod, body)
	}

	// Check if we have a valid API endpoint
	if c.apiEndpoint == "" {
		if c.Debug {
			log.Println("No valid apiEndpoint, re-authorizing client")
		}
		if err := c.AuthorizeAccount(); err != nil {
			return err
		}
	}

	// Post the API request
	resp, err := c.post(c.apiEndpoint+v1+apiMethod, bytes.NewReader(body))
	if err != nil {
		if c.Debug {
			log.Println("B2.post returned an error: ", err)
		}
		return err
	}

	err = c.parseResponse(resp, response)

	// Retry after non-fatal errors
	if b2err, ok := err.(*B2Error); ok {
		if !b2err.IsFatal() && !c.NoRetry {
			resp, err = c.post(c.apiEndpoint+v1+apiMethod, bytes.NewReader(body))
			if err == nil {
				err = c.parseResponse(resp, response)
			}
		}
	}
	return err
}
