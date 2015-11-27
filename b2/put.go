package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uiprogress/util/strutil"

	"gopkg.in/kothar/go-backblaze.v0"
)

// TODO support directories
// TODO support replacing all previous versions
type Put struct {
}

func init() {
	parser.AddCommand("put", "Store a file",
		"Uploads one or more files. Specify the bucket with -b, and the filenames to upload as extra arguments.",
		&Put{})
}

func (o *Put) Execute(args []string) error {
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
		_, err := upload(bucket, file)
		if err != nil {
			return err
		}
	}

	return nil
}

type progressReader struct {
	bar *uiprogress.Bar
	r   io.ReadSeeker
}

func (p *progressReader) Read(b []byte) (int, error) {
	read, err := p.r.Read(b)
	p.bar.Set(p.bar.Current() + read)
	return read, err
}

func (p *progressReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		p.bar.Set(int(offset))
	case 1:
		p.bar.Set(p.bar.Current() + int(offset))
	case 2:
		p.bar.Set(p.bar.Total - int(offset))
	}
	return p.r.Seek(offset, whence)
}

func upload(bucket *backblaze.Bucket, file string) (*backblaze.File, error) {

	stat, err := os.Stat(file)
	if err != nil {
		return nil, err
	}

	reader, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	bar := uiprogress.AddBar(int(stat.Size()))
	bar.AppendFunc(func(b *uiprogress.Bar) string {
		speed := (float32(b.Current()) / 1.024) / float32(b.TimeElapsed())
		return fmt.Sprintf("%7.2f KB/s", speed)
	})
	bar.AppendCompleted()
	bar.PrependFunc(func(b *uiprogress.Bar) string { return fmt.Sprintf("%10d", b.Total) })
	bar.PrependFunc(func(b *uiprogress.Bar) string { return strutil.Resize(file, 50) })
	bar.Width = 20

	r := &progressReader{bar, reader}

	return bucket.UploadFile(filepath.Base(file), nil, r)
}
