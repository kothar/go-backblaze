package main

import (
	"fmt"
)

// DeleteBucket is a command
type DeleteBucket struct {
}

func init() {
	parser.AddCommand("deletebucket", "Delete a bucket", "", &DeleteBucket{})
}

// Execute the deletebucket command
func (o *DeleteBucket) Execute(args []string) error {
	if opts.Bucket == "" {
		return fmt.Errorf("No bucket specified")
	}

	client, err := Client()
	if err != nil {
		return err
	}

	bucket, err := client.Bucket(opts.Bucket)
	if err != nil {
		return err
	}
	if bucket == nil {
		return fmt.Errorf("Bucket not found: %s", opts.Bucket)
	}

	if err = bucket.Delete(); err != nil {
		return err
	}

	fmt.Println("Deleted bucket:", bucket.Name)

	return nil
}
