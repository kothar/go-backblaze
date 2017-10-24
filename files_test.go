package backblaze

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestDownloadFile(T *testing.T) {

	accountID := "test"
	testFile := []byte("File contents")

	client, server := prepareResponses([]response{
		{code: 200, body: authorizeAccountResponse{
			AccountID:          accountID,
			APIEndpoint:        "http://api.url",
			AuthorizationToken: "testToken",
			DownloadURL:        "http://download.url",
		}},
		{code: 200, body: testFile},
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

	_, reader, err := b2.DownloadFileRangeByID("fileId", &FileRange{0, 3})
	if err != nil {
		T.Fatal(err)
	} else {
		defer reader.Close()
		body, err := ioutil.ReadAll(reader)
		if err != nil {
			T.Fatal(err)
		}
		if !bytes.Equal(body, testFile) {
			T.Errorf("Expected file contents to be [%s], saw [%s]", testFile[0:4], body)
		}
	}
}

func TestDownloadFileRange(T *testing.T) {

	accountID := "test"
	testFile := []byte("File contents")

	client, server := prepareResponses([]response{
		{code: 200, body: authorizeAccountResponse{
			AccountID:          accountID,
			APIEndpoint:        "http://api.url",
			AuthorizationToken: "testToken",
			DownloadURL:        "http://download.url",
		}},
		{code: 200, body: testFile[0:4], headers: map[string]string{
			"Content-Range": "bytes 0-3/13",
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

	_, reader, err := b2.DownloadFileRangeByID("fileId", &FileRange{0, 3})
	if err != nil {
		T.Fatal(err)
	} else {
		defer reader.Close()
		body, err := ioutil.ReadAll(reader)
		if err != nil {
			T.Fatal(err)
		}
		if !bytes.Equal(body, testFile[0:4]) {
			T.Errorf("Expected file contents to be [%s], saw [%s]", testFile[0:4], body)
		}
	}
}

func TestDownloadReAuth(T *testing.T) {

	accountID := "test"
	token2 := "testToken2"
	testFile := "File contents"

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
		{code: 200, body: testFile},
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

	_, reader, err := b2.DownloadFileByID("fileId")
	if err != nil {
		T.Fatal(err)
	} else {
		defer reader.Close()
		body, err := ioutil.ReadAll(reader)
		if err != nil {
			T.Fatal(err)
		}
		if string(body) != toJSON(testFile) {
			T.Errorf("Expected file contents to be [%s], saw [%s]", toJSON(testFile), body)
		}
	}

	if b2.auth.AuthorizationToken != token2 {
		T.Errorf("Expected auth token after re-auth to be %q, saw %q", token2, b2.auth.AuthorizationToken)
	}
}
