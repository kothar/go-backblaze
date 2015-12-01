package backblaze

import (
	"os"
	"path/filepath"
)

func ExampleB2(accountID, applicationKey string) *B2 {
	b2, _ := NewB2(Credentials{
		AccountID:      accountID,
		ApplicationKey: applicationKey,
	})
	return b2
}

func ExampleBucket(b2 *B2) *Bucket {
	bucket, _ := b2.CreateBucket("test_bucket", AllPrivate)
	return bucket
}

func ExampleBucketUploadFile(bucket *Bucket, path string) *File {
	reader, _ := os.Open(path)
	name := filepath.Base(path)
	metadata := make(map[string]string)

	file, _ := bucket.UploadFile(name, metadata, reader)

	return file
}
