package backblaze

import (
	"os"
	"path/filepath"

	"gopkg.in/kothar/go-backblaze.v0"
)

func ExampleB2(accountId, applicationKey string) *backblaze.B2 {
	b2, _ := backblaze.NewB2(backblaze.Credentials{
		AccountId:      accountId,
		ApplicationKey: applicationKey,
	})
	return b2
}

func ExampleBucket(b2 *backblaze.B2) *backblaze.Bucket {
	bucket, _ := b2.CreateBucket("test_bucket", backblaze.AllPrivate)
	return bucket
}

func ExampleBucketUploadFile(bucket *backblaze.Bucket, path string) *backblaze.File {
	reader, _ := os.Open(path)
	name := filepath.Base(path)
	metadata := make(map[string]string)

	file, _ := bucket.UploadFile(name, metadata, reader)

	return file
}
