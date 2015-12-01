package main

import (
	"fmt"

	"gopkg.in/kothar/go-backblaze.v0"
)

// CreateBucket is a command
type CreateBucket struct {
	Public bool `short:"p" long:"public" description:"Make bucket contents public"`
}

func init() {
	parser.AddCommand("createbucket", "Create a new bucket", "", &CreateBucket{})
}

// Execute the createbucket command
func (o *CreateBucket) Execute(args []string) error {
	client, err := Client()
	if err != nil {
		return err
	}

	bucketType := backblaze.AllPrivate
	if o.Public {
		bucketType = backblaze.AllPublic
	}

	bucket, err := client.CreateBucket(opts.Bucket, bucketType)
	if err != nil {
		return err
	}

	fmt.Println("Created bucket:", bucket.Name)

	return nil
}
