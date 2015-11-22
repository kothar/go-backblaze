package main

import (
	"errors"
	"fmt"
	"time"
)

type List struct {
}

func init() {
	parser.AddCommand("list", "List files in a bucket", "", &List{})
}

func (o *List) Execute(args []string) error {
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

	response, err := bucket.ListFileNames("", 100)
	if err != nil {
		return err
	}

	if opts.Verbose {
		fmt.Printf("Contents of %s/\n", opts.Bucket)
		for _, file := range response.Files {
			fmt.Printf("%10d %40s     %s\n", file.Size, time.Unix(file.UploadTimestamp/1000, file.UploadTimestamp%1000), file.Name)
		}
	} else {
		for _, file := range response.Files {
			fmt.Println(file.Name)
		}
	}

	return nil
}
