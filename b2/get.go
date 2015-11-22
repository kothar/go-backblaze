package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uiprogress/util/strutil"
	"github.com/pH14/go-backblaze"
)

type Get struct {
}

func init() {
	parser.AddCommand("get", "Download a file from B2", "", &Get{})
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

type barWriter struct {
	bar *uiprogress.Bar
}

func (bw *barWriter) Write(b []byte) (int, error) {
	size := len(b)
	bw.bar.Set(bw.bar.Current() + size)
	return size, nil
}

func download(bucket *backblaze.Bucket, file string) error {
	fileInfo, reader, err := bucket.DownloadFileByName(file)
	if err != nil {
		return err
	}
	defer reader.Close()

	bar := uiprogress.AddBar(int(fileInfo.ContentLength))
	bar.AppendCompleted()
	bar.PrependFunc(func(b *uiprogress.Bar) string { return strutil.Resize(fileInfo.Name, 50) })
	bar.Width = 30

	writer, err := os.Create(file)
	if err != nil {
		return err
	}
	defer writer.Close()

	sha := sha1.New()
	tee := io.MultiWriter(writer, sha, &barWriter{bar})

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
