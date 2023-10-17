package backblaze

import (
	"testing"
	"time"
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

func TestCreateApplicationKey(T *testing.T) {

	accountID := "test"

	keyName := "my-new-key"
	applicationKeyId := "fffff000123"
	applicationKey := "abcdfxyz"

	timestampNow := time.Now().Unix()
	keyValidDurationInSeconds := 3600
	expirationTimeStamp := timestampNow + int64(keyValidDurationInSeconds)

	client, server := prepareResponses([]response{
		{code: 200, body: authorizeAccountResponse{
			AccountID:          accountID,
			APIEndpoint:        "http://api.url",
			AuthorizationToken: "testToken",
			DownloadURL:        "http://download.url",
		}},
		{code: 200, body: ApplicationKeyResponse{
			KeyName:             keyName,
			ApplicationKeyId:    applicationKeyId,
			ApplicationKey:      applicationKey,
			ExpirationTimestamp: expirationTimeStamp,
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

	appkey, err := b2.CreateApplicationKey(&CreateKeyRequest{
		Capabilities:           []string{"listAllBucketNames", "listKeys"},
		KeyName:                keyName,
		ValidDurationInSeconds: keyValidDurationInSeconds,
	})

	if err != nil {
		T.Fatal(err)
	}

	if appkey.KeyName != keyName {
		T.Errorf("Expected to get keyName '%s', received '%s'", keyName, appkey.KeyName)
	}

	if appkey.ApplicationKeyId != applicationKeyId {
		T.Errorf("Expected to get applicationKeyId '%s', received '%s'", applicationKeyId, appkey.ApplicationKeyId)
	}

	if appkey.ApplicationKey != applicationKey {
		T.Errorf("Expected to get applicationKey '%s', received '%s'", applicationKey, appkey.ApplicationKey)
	}

	if appkey.ExpirationTimestamp != expirationTimeStamp {
		T.Errorf("Expected to get ExpirationTimestamp '%d', received '%d'", expirationTimeStamp, appkey.ExpirationTimestamp)
	}

}
