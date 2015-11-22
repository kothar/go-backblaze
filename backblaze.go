package backblaze

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

const (
	B2_HOST = "https://api.backblaze.com"
	V1      = "/b2api/v1/"
)

type Credentials struct {
	AccountId      string
	ApplicationKey string
}

type B2 struct {
	Credentials

	// If true, display debugging information about API calls
	Debug bool

	// State
	authorizationToken string
	apiUrl             string
	downloadUrl        string
	httpClient         http.Client
}

type B2Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (e *B2Error) Error() string {
	return e.Code + ": " + e.Message
}

type AuthorizeAccountResponse struct {
	AccountId          string `json:"accountId"`
	ApiUrl             string `json:"apiUrl"`
	AuthorizationToken string `json:"authorizationToken"`
	DownloadUrl        string `json:"downloadUrl"`
}

// Creates a new Client for accessing the B2 API.
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

// Used to log in to the B2 API. Returns an authorization token that can be
// used for account-level operations, and a URL that should be used as the
// base URL for subsequent API calls.
func (c *B2) AuthorizeAccount() error {
	req, err := http.NewRequest("GET", B2_HOST+V1+"b2_authorize_account", nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.AccountId, c.ApplicationKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	authResponse := &AuthorizeAccountResponse{}
	err = c.parseResponse(resp, authResponse)
	if err != nil {
		return err
	}

	// Store token
	c.authorizationToken = authResponse.AuthorizationToken
	c.downloadUrl = authResponse.DownloadUrl
	c.apiUrl = authResponse.ApiUrl

	return nil
}

// Create an authorized request using the client's credentials
func (c *B2) authRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	if c.authorizationToken != "" {
		req.Header.Add("Authorization", c.authorizationToken)
	}

	if c.Debug {
		println("Request: " + req.URL.String())
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
func (c *B2) parseError(resp *http.Response) error {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if c.Debug {
		println("Response: " + string(body))
	}

	b2err := &B2Error{}
	json.Unmarshal(body, b2err)
	return b2err
}

// Attempts to parse a response body into the provided result struct
func (c *B2) parseResponse(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()

	// Check response code
	switch resp.StatusCode {
	case 200: // Response is OK
	case 400: // BAD_REQUEST
		return c.parseError(resp)
	case 401:
		return errors.New("UNAUTHORIZED - The account ID is wrong, the account does not have B2 enabled, or the application key is not valid")
	default:
		if err := c.parseError(resp); err != nil {
			return err
		}
		return fmt.Errorf("Unrecognised status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if c.Debug {
		println("Response: " + string(body))
	}

	return json.Unmarshal(body, result)
}

// Perform a B2 API request with the provided request and response objects
func (c *B2) apiRequest(apiMethod string, request interface{}, response interface{}) error {
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	resp, err := c.post(c.apiUrl+V1+apiMethod, bytes.NewReader(body))
	if err != nil {
		return err
	}

	err = c.parseResponse(resp, response)
	if err != nil {
		return err
	}

	return nil
}
