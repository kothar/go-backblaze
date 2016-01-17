package backblaze

import (
	"io/ioutil"
	"testing"
)

func TestDownloadReAuth(T *testing.T) {

	accountID := "test"
	token2 := "testToken2"
	testFile := "File contents"

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
		{200, testFile},
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
