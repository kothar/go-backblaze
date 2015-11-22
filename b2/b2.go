// A minimal client for accessing B2
package main

import (
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/pH14/go-backblaze"
)

type Options struct {
	// Credentials
	AccountId      string `short:"a" long:"account" env:"B2_ACCOUNT_ID"`
	ApplicationKey string `short:"k" long:"appKey" env:"B2_APP_KEY"`

	// Bucket
	Bucket string `short:"b" long:"bucket" env:"B2_BUCKET"`

	// Commands =================

	Get struct {
	} `command:"get"`

	Delete struct {
	} `command:"delete"`

	ListBuckets struct {
	} `command:"listbuckets"`

	CreateBucket struct {
	} `command:"createbucket"`

	DeleteBucket struct {
	} `command:"deletebucket"`
}

var opts = &Options{}

var parser = flags.NewParser(opts, flags.Default)

func main() {
	_, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}
}

func Client() (*backblaze.Client, error) {
	return backblaze.NewClient(backblaze.Credentials{
		opts.AccountId,
		opts.ApplicationKey,
	})
}
