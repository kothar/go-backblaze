package backblaze

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/pquerna/ffjson/ffjson"
)

type response struct {
	code    int
	body    interface{}
	headers map[string]string
}

// Based on http://keighl.com/post/mocking-http-responses-in-golang/
func prepareResponses(responses []response) (*http.Client, *httptest.Server) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(responses) == 0 {
			w.WriteHeader(500)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, "No more responses")
			return
		}

		next := responses[0]
		responses = responses[1:]

		for k, v := range next.headers {
			w.Header().Add(k, v)
		}
		w.WriteHeader(next.code)
		w.Header().Set("Content-Type", "application/json")
		if body, ok := next.body.([]byte); ok {
			w.Write(body)
		} else {
			fmt.Fprint(w, toJSON(next.body))
		}
	}))

	// Make a transport that reroutes all traffic to the example server
	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL + req.URL.Path)
		},
	}

	// Make a http.Client with the transport
	client := &http.Client{Transport: transport}

	return client, server
}

func toJSON(o interface{}) string {
	bytes, err := ffjson.Marshal(o)
	if err != nil {
		log.Fatal(err)
	}
	return string(bytes)
}

func TestUnauthorizedError(T *testing.T) {
	err := &B2Error{Status: 401, Code: "expired_auth_token"}
	if err.IsFatal() {
		T.Fatal("401 expired_auth_token error should not be considered fatal")
	}
}

func TestAuth(T *testing.T) {
	accountID := "test"
	token := "testToken"
	apiURL := "http://api.url"
	downloadURL := "http://download.url"

	client, server := prepareResponses([]response{
		{code: 200, body: authorizeAccountResponse{
			AccountID:          accountID,
			APIEndpoint:        apiURL,
			AuthorizationToken: token,
			DownloadURL:        downloadURL,
		}},
	})
	defer server.Close()

	b2 := &B2{
		Credentials: Credentials{
			AccountID:      accountID,
			ApplicationKey: "test",
		},
		Debug:      testing.Verbose(),
		host:       server.URL,
		httpClient: *client,
	}

	if err := b2.AuthorizeAccount(); err != nil {
		T.Fatal(err)
	}

	if b2.auth.AuthorizationToken != token {
		T.Errorf("Auth token not set correctly: expecting %q, saw %q", token, b2.auth.AuthorizationToken)
	}

	if b2.auth.APIEndpoint != apiURL {
		T.Errorf("API Endpoint not set correctly: expecting %q, saw %q", apiURL, b2.auth.APIEndpoint)
	}

	if b2.auth.DownloadURL != downloadURL {
		T.Errorf("Download URL not set correctly: expecting %q, saw %q", downloadURL, b2.auth.DownloadURL)
	}
}

func TestReAuth(T *testing.T) {

	accountID := "test"
	bucketID := "bucketid"
	token2 := "testToken2"

	client, server := prepareResponses([]response{
		{code: 200, body: authorizeAccountResponse{
			AccountID:          accountID,
			APIEndpoint:        "http://api.url",
			AuthorizationToken: "testToken",
			DownloadURL:        "http://download.url",
		}},
		{code: 401, body: B2Error{
			Status:  401,
			Code:    "expired_auth_token",
			Message: "Authentication token expired",
		}},
		{code: 200, body: authorizeAccountResponse{
			AccountID:          accountID,
			APIEndpoint:        "http://api.url",
			AuthorizationToken: token2,
			DownloadURL:        "http://download.url",
		}},
		{code: 200, body: listBucketsResponse{
			Buckets: []*BucketInfo{
				&BucketInfo{
					ID:         bucketID,
					AccountID:  accountID,
					Name:       "testbucket",
					BucketType: AllPrivate,
				},
			},
		}},
	})
	defer server.Close()

	b2 := &B2{
		Credentials: Credentials{
			AccountID:      accountID,
			ApplicationKey: "test",
		},
		Debug:      testing.Verbose(),
		httpClient: *client,
		host:       server.URL,
	}

	_, err := b2.ListBuckets()
	if err != nil {
		T.Fatal(err)
	}

	if b2.auth.AuthorizationToken != token2 {
		T.Errorf("Expected auth token after re-auth to be %q, saw %q", token2, b2.auth.AuthorizationToken)
	}
}
