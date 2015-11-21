package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pH14/go-backblaze"
)

type Put struct {
}

func init() {
	parser.AddCommand("put", "Store a file in B2", "", &Put{})
}

func (o *Put) Execute(args []string) error {
	client, err := Client()
	if err != nil {
		return err
	}
	fmt.Printf("\nClient: %+v\n", client)

	bucket, err := client.Bucket(opts.Bucket)
	if err != nil {
		return err
	}

	for _, file := range args {
		upload(bucket, file)
	}

	return nil
}

func upload(bucket *backblaze.Bucket, file string) (*backblaze.File, error) {
	reader, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return bucket.UploadFile(filepath.Base(file), reader)
}
