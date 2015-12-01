package main

import (
	"fmt"
)

// ListBuckets is a command
type ListBuckets struct {
}

func init() {
	parser.AddCommand("listbuckets", "List buckets in an account", "", &ListBuckets{})
}

// Execute the listbuckets command
func (o *ListBuckets) Execute(args []string) error {
	client, err := Client()
	if err != nil {
		return err
	}

	response, err := client.ListBuckets()
	if err != nil {
		return err
	}

	if opts.Verbose {
		fmt.Printf("%-30s%-35s%-15s\n", "Name", "Id", "Type")
		for _, bucket := range response {
			fmt.Printf("%-30s%-35s%-15s\n", bucket.Name, bucket.ID, bucket.BucketType)
		}
	} else {
		for _, bucket := range response {
			fmt.Println(bucket.Name)
		}
	}

	return nil
}
