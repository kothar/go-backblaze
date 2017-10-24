// A minimal client for accessing B2
package main

import (
	"flag"
	"os"

	"github.com/jessevdk/go-flags"

	"gopkg.in/kothar/go-backblaze.v0"
)

// Options defines command line flags used by this application
type Options struct {
	// Credentials
	AccountID      string `long:"account" env:"B2_ACCOUNT_ID" description:"The account ID to use"`
	ApplicationKey string `long:"appKey" env:"B2_APP_KEY" description:"The application key to use"`

	// Bucket
	Bucket string `short:"b" long:"bucket" env:"B2_BUCKET" description:"The bucket to access"`

	Debug   bool `short:"d" long:"debug" description:"Debug API requests"`
	Verbose bool `short:"v" long:"verbose" description:"Display verbose output"`
}

var opts = &Options{}

var parser = flags.NewParser(opts, flags.Default)

func main() {
	flag.Parse()
	_, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}
}

// Client obtains an instance of the B2 client
func Client() (*backblaze.B2, error) {
	c, err := backblaze.NewB2(backblaze.Credentials{
		AccountID:      opts.AccountID,
		ApplicationKey: opts.ApplicationKey,
	})
	if err != nil {
		return nil, err
	}

	c.Debug = opts.Debug
	return c, nil
}
