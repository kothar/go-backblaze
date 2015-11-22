package main

import (
	"fmt"
)

type DeleteBucket struct {
}

func init() {
	parser.AddCommand("deletebucket", "Delete a bucket", "", &DeleteBucket{})
}

func (o *DeleteBucket) Execute(args []string) error {
	client, err := Client()
	if err != nil {
		return err
	}

	bucket, err := client.Bucket(opts.Bucket)
	if err != nil {
		return err
	}

	if _, err = client.DeleteBucket(bucket.Id); err != nil {
		return err
	}

	fmt.Println("Deleted bucket:", bucket.Name)

	return nil
}
