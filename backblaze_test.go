package backblaze

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type response struct {
	code int
	body interface{}
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

		w.WriteHeader(next.code)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, toJSON(next.body))
	}))

	// Make a transport that reroutes all traffic to the example server
	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			u, err := url.Parse(server.URL + req.URL.Path)
			fmt.Printf("Request URL: %s\n", req.URL)
			return u, err
		},
	}

	// Make a http.Client with the transport
	client := &http.Client{Transport: transport}

	return client, server
}

func toJSON(o interface{}) string {
	bytes, err := json.Marshal(o)
	if err != nil {
		log.Fatal(err)
	}
	return string(bytes)
}

func TestUnauthorizedError(T *testing.T) {
	err := &B2Error{Status: 401}
	if err.IsFatal() {
		T.Fatal("401 error should not be considered fatal")
	}
}

func TestAuth(T *testing.T) {
	accountID := "test"
	token := "testToken"
	apiURL := "http://api.url"
	downloadURL := "http://download.url"

	client, server := prepareResponses([]response{
		{200, authorizeAccountResponse{
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
		Debug:      true,
		host:       server.URL,
		httpClient: client,
	}

	if err := b2.AuthorizeAccount(); err != nil {
		T.Fatal(err)
	}

	if b2.authorizationToken != token {
		T.Errorf("Auth token not set correctly: expecting %q, saw %q", token, b2.authorizationToken)
	}

	if b2.apiEndpoint != apiURL {
		T.Errorf("API Endpoint not set correctly: expecting %q, saw %q", apiURL, b2.apiEndpoint)
	}

	if b2.downloadURL != downloadURL {
		T.Errorf("Download URL not set correctly: expecting %q, saw %q", downloadURL, b2.downloadURL)
	}
}

func TestListBuckets(T *testing.T) {

	accountID := "test"
	bucketID := "bucketid"

	client, server := prepareResponses([]response{
		{200, authorizeAccountResponse{
			AccountID:          accountID,
			APIEndpoint:        "http://api.url",
			AuthorizationToken: "testToken",
			DownloadURL:        "http://download.url",
		}},
		{200, listBucketsResponse{
			Buckets: []*Bucket{
				&Bucket{
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
		Debug:      true,
		httpClient: client,
		host:       server.URL,
	}

	buckets, err := b2.ListBuckets()
	if err != nil {
		T.Fatal(err)
	}

	if len(buckets) != 1 {
		T.Errorf("Expected 1 bucket, received %d", len(buckets))
	}
	if buckets[0].ID != bucketID {
		T.Errorf("Bucket ID does not match: expected %q, saw %q", bucketID, buckets[0].ID)
	}
}

func TestReAuth(T *testing.T) {

	accountID := "test"
	bucketID := "bucketid"
	token2 := "testToken2"

	client, server := prepareResponses([]response{
		{200, authorizeAccountResponse{
			AccountID:          accountID,
			APIEndpoint:        "http://api.url",
			AuthorizationToken: "testToken",
			DownloadURL:        "http://download.url",
		}},
		{401, B2Error{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "Authentication token expired",
		}},
		{200, authorizeAccountResponse{
			AccountID:          accountID,
			APIEndpoint:        "http://api.url",
			AuthorizationToken: token2,
			DownloadURL:        "http://download.url",
		}},
		{200, listBucketsResponse{
			Buckets: []*Bucket{
				&Bucket{
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
		Debug:      true,
		httpClient: client,
		host:       server.URL,
	}

	_, err := b2.ListBuckets()
	if err != nil {
		T.Fatal(err)
	}

	if b2.authorizationToken != token2 {
		T.Errorf("Expected auth token after re-auth to be %q, saw %q", token2, b2.authorizationToken)
	}
}
