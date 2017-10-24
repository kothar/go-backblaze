package backblaze

import (
	"testing"
)

func TestListBuckets(T *testing.T) {

	accountID := "test"
	bucketID := "bucketid"

	client, server := prepareResponses([]response{
		{code: 200, body: authorizeAccountResponse{
			AccountID:          accountID,
			APIEndpoint:        "http://api.url",
			AuthorizationToken: "testToken",
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
