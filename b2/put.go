package main

import (
	"bytes"
	"errors"
	"io/ioutil"
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

	bucket, err := client.Bucket(opts.Bucket)
	if err != nil {
		return err
	}
	if bucket == nil {
		return errors.New("Bucket not found: " + opts.Bucket)
	}

	for _, file := range args {
		_, err := upload(bucket, file)
		if err != nil {
			return err
		}

	}

	return nil
}

func upload(bucket *backblaze.Bucket, file string) (*backblaze.File, error) {
	reader, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	all, err := ioutil.ReadAll(reader)
	defer reader.Close()

	return bucket.UploadFile(filepath.Base(file), bytes.NewReader(all))
}
