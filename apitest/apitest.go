package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/jessevdk/go-flags"

	"gopkg.in/kothar/go-backblaze.v0"
)

// Options defines command line flags used by this application
type Options struct {
	// Credentials
	AccountID      string `long:"account" env:"B2_ACCOUNT_ID" description:"The account ID to use"`
	ApplicationKey string `long:"appKey" env:"B2_APP_KEY" description:"The application key to use"`

	// Bucket
	Bucket string `short:"b" long:"bucket" description:"The bucket name to use for testing (a random bucket name will be chosen if not specified)"`
}

var opts = &Options{}

var parser = flags.NewParser(opts, flags.Default)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {

	_, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}

	// Create client
	b2, err := backblaze.NewB2(backblaze.Credentials{
		AccountID:      opts.AccountID,
		ApplicationKey: opts.ApplicationKey,
	})
	check(err)

	b := testBucketCreate(b2)

	f, data := testFileUpload(b)

	testFileDownload(b, f, data)

	testFileDelete(b, f)

	testListFiles(b)

	testBucketDelete(b)
}

func testBucketCreate(b2 *backblaze.B2) *backblaze.Bucket {
	// Get Test bucket
	if opts.Bucket == "" {
		opts.Bucket = "test-bucket-" + randSeq(10)
	}
	log.Printf("Testing with bucket %s", opts.Bucket)

	b, err := b2.Bucket(opts.Bucket)
	check(err)
	if b != nil {
		log.Fatal("Testing bucket already exists")
	}

	b, err = b2.CreateBucket(opts.Bucket, backblaze.AllPrivate)
	check(err)
	log.Print("Bucket created")

	return b
}

func testBucketDelete(b *backblaze.Bucket) {
	check(b.Delete())
	log.Print("Bucket deleted")
}

func testFileUpload(b *backblaze.Bucket) (*backblaze.File, []byte) {
	fileData := randBytes(1024 * 1024)

	f, err := b.UploadFile("test_file", nil, bytes.NewBuffer(fileData))
	check(err)

	log.Print("File uploaded")

	return f, fileData
}

func testFileDownload(b *backblaze.Bucket, f *backblaze.File, data []byte) {
	f, reader, err := b.DownloadFileByName(f.Name)
	check(err)

	body, err := ioutil.ReadAll(reader)
	check(err)

	if !bytes.Equal(body, data) {
		log.Fatal("Downloaded file content does not match upload")
	}

	log.Print("File downloaded")
}

func testFileDelete(b *backblaze.Bucket, f *backblaze.File) {
	_, err := b.DeleteFileVersion(f.Name, f.ID)
	check(err)

	log.Print("File deleted")
}

func testListFiles(b *backblaze.Bucket) {
	fileData := randBytes(1024)

	files := []*backblaze.File{}

	// Upload files
	count := 40
	for i := 1; i <= count; i++ {
		log.Printf("Uploading file %d/%d...", i, count)
		f, err := b.UploadFile("test_file_"+strconv.FormatInt(int64(i), 10), nil, bytes.NewBuffer(fileData))
		check(err)
		files = append(files, f)
	}
	log.Println("Done.")

	// List bucket content
	log.Println("Listing bucket contents")
	bulkResponse, err := b.ListFileNames("", 500)
	check(err)
	if len(bulkResponse.Files) != count {
		log.Fatalf("Expected listing to return %d files but found %d", count, len(bulkResponse.Files))
	}

	// Test paging
	log.Println("Paging bucket contents")
	pagedFiles := []backblaze.FileStatus{}
	cursor := ""
	for {
		r, err := b.ListFileNames(cursor, 10)
		check(err)

		pagedFiles = append(pagedFiles, r.Files...)

		if r.NextFileName == "" {
			break
		}

		cursor = r.NextFileName
	}

	if !reflect.DeepEqual(bulkResponse.Files, pagedFiles) {
		log.Fatalf("Result of paged directory listing does not match bulk listing")
	}

	// Delete files
	log.Printf("Deleting %d files...", count)
	for _, f := range files {
		_, err := b.DeleteFileVersion(f.Name, f.ID)
		check(err)
	}
	log.Println("Done.")
}

func check(err error) {
	if err == nil {
		return
	}

	log.Fatal(err)
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// see http://stackoverflow.com/a/22892986/37416
func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.Int())
	}
	return b
}
