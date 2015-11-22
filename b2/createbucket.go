package main

import (
	"fmt"

	"github.com/pH14/go-backblaze"
)

type CreateBucket struct {
	Public bool `short:"p" long:"public" description:"Make bucket contents public"`
}

func init() {
	parser.AddCommand("createbucket", "Create a new bucket", "", &CreateBucket{})
}

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
