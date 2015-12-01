package backblaze

import (
	"os"
	"path/filepath"
)

func ExampleB2() {
	NewB2(Credentials{
		AccountID:      "", // Obtained from your B2 account page.
		ApplicationKey: "", // Obtained from your B2 account page.
	})
}

func ExampleBucket() {
	var b2 B2
	// ...

	b2.CreateBucket("test_bucket", AllPrivate)
}

func ExampleBucket_UploadFile() {
	var bucket Bucket
	// ...

	path := "/path/to/file"
	reader, _ := os.Open(path)
	name := filepath.Base(path)
	metadata := make(map[string]string)

	bucket.UploadFile(name, metadata, reader)
}
