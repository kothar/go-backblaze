package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uiprogress/util/strutil"

	"gopkg.in/kothar/go-backblaze.v0"
)

// TODO support subdirectories
// TODO support destination path
// TODO support version id downloads

// Get is a command
type Get struct {
	Threads int `short:"j" long:"threads" default:"5" description:"Maximum simultaneous downloads to process"`
}

func init() {
	parser.AddCommand("get", "Download a file",
		"Downloads one or more files to the current directory. Specify the bucket with -b, and the filenames to download as extra arguments.",
		&Get{})
}

// Execute the get command
func (o *Get) Execute(args []string) error {
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

	uiprogress.Start()
	pool := make(chan bool, o.Threads)
	group := sync.WaitGroup{}
	var downloadError error

	for _, file := range args {
		// TODO handle wildcards

		fileInfo, reader, err := bucket.DownloadFileByName(file)
		if err != nil {
			downloadError = err
			break
		}

		// Get a ticket to process a download
		pool <- true

		if downloadError != nil {
			break
		}

		// Start next parallel download
		group.Add(1)
		go func(fileInfo *backblaze.File, reader io.ReadCloser, path string) {
			err := download(fileInfo, reader, path)
			if err != nil {
				fmt.Println(err)
				downloadError = err
			}

			// Allow next entry into pool
			group.Done()
			<-pool
		}(fileInfo, reader, file)
	}

	group.Wait()

	return downloadError
}

type progressWriter struct {
	bar *uiprogress.Bar
	w   io.Writer
}

func (p *progressWriter) Write(b []byte) (int, error) {
	written, err := p.w.Write(b)
	p.bar.Set(p.bar.Current() + written)
	return written, err
}

func download(fileInfo *backblaze.File, reader io.ReadCloser, path string) error {
	defer reader.Close()

	err := os.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		return err
	}

	writer, err := os.Create(path)
	if err != nil {
		return err
	}
	defer writer.Close()

	var w io.Writer = writer
	if opts.Verbose {
		bar := uiprogress.AddBar(int(fileInfo.ContentLength))
		start := time.Now()
		elapsed := time.Duration(1)
		bar.AppendFunc(func(b *uiprogress.Bar) string {
			// elapsed := b.TimeElapsed()
			if b.Current() < b.Total {
				elapsed = time.Now().Sub(start)
			}
			speed := uint64(float64(b.Current()) / elapsed.Seconds())
			return humanize.IBytes(speed) + "/sec"
		})
		bar.AppendCompleted()
		bar.PrependFunc(func(b *uiprogress.Bar) string { return fmt.Sprintf("%10s", humanize.IBytes(uint64(b.Total))) })
		bar.PrependFunc(func(b *uiprogress.Bar) string { return strutil.Resize(fileInfo.Name, 50) })
		bar.Width = 20

		w = &progressWriter{bar, writer}
	}

	sha := sha1.New()
	tee := io.MultiWriter(sha, w)

	_, err = io.Copy(tee, reader)
	if err != nil {
		return err
	}

	// Check SHA
	sha1Hash := hex.EncodeToString(sha.Sum(nil))
	if sha1Hash != fileInfo.ContentSha1 {
		return errors.New("Downloaded data does not match SHA1 hash")
	}

	return nil
}
