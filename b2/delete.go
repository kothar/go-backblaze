package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Delete is a command
type Delete struct {
	Hide bool `long:"hide" description:"Hide the file, leaving previous versions in place"`
	All  bool `short:"a" long:"all" description:"Remove all versions of a file"`
}

func init() {
	parser.AddCommand("delete", "Delete a file",
		"Specify just a filename to hide the file from listings. Specifiy a version id as fileName:versionId to permanently delete a file version.",
		&Delete{})
}

// Execute the delete command
func (o *Delete) Execute(args []string) error {
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

	for _, file := range args {
		// TODO handle wildcards

		if opts.Verbose {
			fmt.Println(file)
		}

		parts := strings.SplitN(file, ":", 2)
		if len(parts) > 1 {
			bucket.DeleteFileVersion(parts[0], parts[1])
		} else {
			if o.Hide {
				bucket.HideFile(file)
			} else {
				// Get most recent versions
				count := 1
				if o.All {
					count = 1000
				}

				versions, err := bucket.ListFileVersions(file, "", count)
				if err != nil {
					return err
				}

				count = 0
				for _, f := range versions.Files {
					if f.Name != file {
						break
					}

					if opts.Verbose && o.All {
						fmt.Printf("  %s %v\n", f.ID, time.Unix(f.UploadTimestamp/1000, f.UploadTimestamp%1000))
					}

					if _, err := bucket.DeleteFileVersion(file, f.ID); err != nil {
						return err
					}
					count++
				}
				if count == 0 {
					return errors.New("File not found: " + file)
				}
			}
		}
	}

	return nil
}
