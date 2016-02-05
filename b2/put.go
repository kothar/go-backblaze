package main

import (
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

// TODO support directories
// TODO support replacing all previous versions

// Put is a command
type Put struct {
	Threads int               `short:"j" long:"threads" default:"5" description:"Maximum simultaneous uploads to process"`
	Meta    map[string]string `long:"meta" description:"Assign metadata to uploaded files"`
}

func init() {
	parser.AddCommand("put", "Store a file",
		"Uploads one or more files. Specify the bucket with -b, and the filenames to upload as extra arguments.",
		&Put{})
}

// Execute the put command
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
	tasks := make(chan string, o.Threads)
	group := sync.WaitGroup{}

	// Create workers
	for i := 0; i < o.Threads; i++ {
		group.Add(1)
		go func() {
			for file := range tasks {
				_, err := upload(bucket, file, o.Meta)
				if err != nil {
					fmt.Println(err)
				}

				// TODO handle termination on error
			}
			group.Done()
		}()
	}

	for _, file := range args {
		tasks <- file
	}
	close(tasks)

	group.Wait()

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

func upload(bucket *backblaze.Bucket, file string, meta map[string]string) (*backblaze.File, error) {

	stat, err := os.Stat(file)
	if err != nil {
		return nil, err
	}

	reader, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var r io.Reader = reader
	if opts.Verbose {
		bar := uiprogress.AddBar(int(stat.Size()))
		// TODO Stop bar refresh when complete

		if stat.Size() > 1024*100 {
			start := time.Now()
			elapsed := time.Duration(1)
			count := 0
			bar.AppendFunc(func(b *uiprogress.Bar) string {
				count++
				if count < 2 {
					return ""
				}

				// elapsed := b.TimeElapsed()
				if b.Current() < b.Total {
					elapsed = time.Now().Sub(start)
				}
				speed := uint64(float64(b.Current()) / elapsed.Seconds())
				return humanize.IBytes(speed) + "/sec"
			})
		}
		bar.AppendCompleted()
		bar.PrependFunc(func(b *uiprogress.Bar) string { return fmt.Sprintf("%10s", humanize.IBytes(uint64(b.Total))) })
		bar.PrependFunc(func(b *uiprogress.Bar) string { return strutil.Resize(file, 50) })
		bar.Width = 20

		r = &progressReader{bar, reader}
	}

	return bucket.UploadFile(filepath.Base(file), meta, r)
}
