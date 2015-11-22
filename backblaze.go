package backblaze

import (
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
	V2      = "/b2api/v2/"
)

type Credentials struct {
	AccountId      string
	ApplicationKey string
}

type Client struct {
	http.Client
	Credentials

	authorizationToken string
	apiUrl             string
	downloadUrl        string
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

func NewClient(creds Credentials) (*Client, error) {
	c := &Client{
		Credentials: creds,
	}

	// Authorize account
	req, err := http.NewRequest("GET", B2_HOST+V1+"b2_authorize_account", nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(creds.AccountId, creds.ApplicationKey)

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	authResponse := &AuthorizeAccountResponse{}
	err = parseResponse(resp, authResponse)
	if err != nil {
		return nil, err
	}

	// Store token
	c.authorizationToken = authResponse.AuthorizationToken
	c.downloadUrl = authResponse.DownloadUrl
	c.apiUrl = authResponse.ApiUrl

	return c, nil
}

// Create an authorized request using the client's credentials
func (c *Client) authRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	if c.authorizationToken != "" {
		req.Header.Add("Authorization", c.authorizationToken)
	}

	println("Request: " + req.URL.String())

	return req, nil
}

// Create an authorized GET request
func (c *Client) Get(path string) (*http.Response, error) {
	req, err := c.authRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

// create an authorized POST request
func (c *Client) Post(path string, body io.Reader) (*http.Response, error) {
	req, err := c.authRequest("POST", path, body)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

func parseError(resp *http.Response) error {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	println("Response: " + string(body))

	b2err := &B2Error{}
	json.Unmarshal(body, b2err)
	return b2err
}

func parseResponse(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()

	// Check response code
	switch resp.StatusCode {
	case 200: // Response is OK
	case 400: // BAD_REQUEST
		return parseError(resp)
	case 401:
		return errors.New("UNAUTHORIZED - The account ID is wrong, the account does not have B2 enabled, or the application key is not valid")
	default:
		if err := parseError(resp); err != nil {
			return err
		}
		return fmt.Errorf("Unrecognised status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	println("Response: " + string(body))

	return json.Unmarshal(body, result)
}
