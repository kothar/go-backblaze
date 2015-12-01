package main

import (
	"errors"
	"fmt"
	"time"
)

// List is a command
type List struct {
	ListVersions bool `short:"a" long:"allVersions" description:"List all versions of files"`
}

func init() {
	parser.AddCommand("list", "List files in a bucket", "", &List{})
}

// Execute the list command
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

	if o.ListVersions {
		response, err := bucket.ListFileVersions("", "", 100)
		if err != nil {
			return err
		}

		if opts.Verbose {
			fmt.Printf("Contents of %s/\n", opts.Bucket)
			for _, file := range response.Files {
				fmt.Printf("%s\n%10d %s %-20s\n\n", file.ID, file.Size, time.Unix(file.UploadTimestamp/1000, file.UploadTimestamp%1000), file.Name)
			}
		} else {
			for _, file := range response.Files {
				fmt.Println(file.Name + ":" + file.ID)
			}
		}
	} else {
		response, err := bucket.ListFileNames("", 100)
		if err != nil {
			return err
		}

		if opts.Verbose {
			fmt.Printf("Contents of %s/\n", opts.Bucket)
			for _, file := range response.Files {
				fmt.Printf("%10d %s %-20s\n", file.Size, time.Unix(file.UploadTimestamp/1000, file.UploadTimestamp%1000), file.Name)
			}
		} else {
			for _, file := range response.Files {
				fmt.Println(file.Name)
			}
		}
	}

	return nil
}
