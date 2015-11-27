package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uiprogress/util/strutil"

	"gopkg.in/kothar/go-backblaze.v0"
)

// TODO support subdirectories
// TODO support destination path
// TODO support version id downloads
type Get struct {
}

func init() {
	parser.AddCommand("get", "Download a file",
		"Downloads one or more files to the current directory. Specify the bucket with -b, and the filenames to download as extra arguments.",
		&Get{})
}

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
	for _, file := range args {
		// TODO handle wildcards
		err := download(bucket, file)
		if err != nil {
			return err
		}

	}

	return nil
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

func download(bucket *backblaze.Bucket, file string) error {
	fileInfo, reader, err := bucket.DownloadFileByName(file)
	if err != nil {
		return err
	}
	defer reader.Close()

	bar := uiprogress.AddBar(int(fileInfo.ContentLength))
	bar.AppendCompleted()
	bar.PrependFunc(func(b *uiprogress.Bar) string { return fmt.Sprintf("%10d", b.Total) })
	bar.PrependFunc(func(b *uiprogress.Bar) string { return strutil.Resize(fileInfo.Name, 50) })
	bar.Width = 30

	writer, err := os.Create(file)
	if err != nil {
		return err
	}
	defer writer.Close()

	sha := sha1.New()
	tee := io.MultiWriter(sha, &progressWriter{bar, writer})

	_, err = io.Copy(tee, reader)
	if err != nil {
		return err
	}

	// Check sha
	sha1Hash := hex.EncodeToString(sha.Sum(nil))
	if sha1Hash != fileInfo.ContentSha1 {
		return errors.New("Downloaded data does not match SHA1 hash")
	}

	return nil
}
