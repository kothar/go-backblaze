// A minimal client for accessing B2
package main

import (
	"os"

	"github.com/jessevdk/go-flags"

	"gopkg.in/kothar/go-backblaze.v0"
)

type Options struct {
	// Credentials
	AccountId      string `long:"account" env:"B2_ACCOUNT_ID" description:"The account ID to use"`
	ApplicationKey string `long:"appKey" env:"B2_APP_KEY" description:"The application key to use"`

	// Bucket
	Bucket string `short:"b" long:"bucket" env:"B2_BUCKET" description:"The bucket to access"`

	Debug   bool `short:"d" long:"debug" description:"Debug API requests"`
	Verbose bool `short:"v" long:"verbose" description:"Display verbose output"`
}

var opts = &Options{}

var parser = flags.NewParser(opts, flags.Default)

func main() {
	_, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}
}

func Client() (*backblaze.B2, error) {
	c, err := backblaze.NewB2(backblaze.Credentials{
		AccountId:      opts.AccountId,
		ApplicationKey: opts.ApplicationKey,
	})
	if err != nil {
		return nil, err
	}

	c.Debug = opts.Debug
	return c, nil
}
