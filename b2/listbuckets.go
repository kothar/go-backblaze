package main

import (
	"fmt"
)

type ListBuckets struct {
}

func init() {
	parser.AddCommand("listbuckets", "List buckets in an account", "", &ListBuckets{})
}

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
			fmt.Printf("%-30s%-35s%-15s\n", bucket.Name, bucket.Id, bucket.BucketType)
		}
	} else {
		for _, bucket := range response {
			fmt.Println(bucket.Name)
		}
	}

	return nil
}
