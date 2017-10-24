package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"sync"
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
	Debug  bool   `short:"d" long:"debug" description:"Show debug information during test"`
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
	b2.Debug = opts.Debug
	check(err)

	b := testBucketCreate(b2)

	// Test basic file operations
	f, data := testFileUpload(b)
	testFileDownload(b, f, data)
	testFileRangeDownload(b, f, data)
	testFileDelete(b, f)

	// Test file listing calls
	files := uploadFiles(b)
	testListFiles(b, files)
	testListDirectories(b, files)
	deleteFiles(b, files)

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

func testFileRangeDownload(b *backblaze.Bucket, f *backblaze.File, data []byte) {
	f, reader, err := b.DownloadFileRangeByName(f.Name, &backblaze.FileRange{Start: 100, End: 2000})
	check(err)

	body, err := ioutil.ReadAll(reader)
	check(err)

	if !bytes.Equal(body, data[100:2000+1]) {
		log.Fatal("Downloaded file range does not match upload")
	}

	log.Print("File range downloaded")
}

func testFileDelete(b *backblaze.Bucket, f *backblaze.File) {
	_, err := b.DeleteFileVersion(f.Name, f.ID)
	check(err)

	log.Print("File deleted")
}

func uploadFiles(b *backblaze.Bucket) []*backblaze.File {
	fileData := randBytes(1024)

	files := []*backblaze.File{}

	queue := make(chan int64)
	var m sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			for n := range queue {
				f, err := b.UploadFile("test/file_"+strconv.FormatInt(n, 10), nil, bytes.NewBuffer(fileData))
				check(err)

				m.Lock()
				files = append(files, f)
				m.Unlock()
			}

			wg.Done()
		}()
	}

	// Upload files
	count := 40
	for i := 1; i <= count; i++ {
		log.Printf("Uploading file %d/%d...", i, count)
		queue <- int64(i)
	}

	close(queue)
	wg.Wait()
	log.Println("Done.")

	return files
}

func testListFiles(b *backblaze.Bucket, files []*backblaze.File) {

	// List bucket content
	log.Println("Listing bucket contents")
	bulkResponse, err := b.ListFileNames("", 500)
	check(err)
	if len(bulkResponse.Files) != len(files) {
		log.Fatalf("Expected listing to return %d files but found %d", len(files), len(bulkResponse.Files))
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
}

func testListDirectories(b *backblaze.Bucket, files []*backblaze.File) {
	// List root directory
	log.Println("Listing root directory contents")
	bulkResponse, err := b.ListFileNamesWithPrefix("", 500, "", "/")
	check(err)
	if len(bulkResponse.Files) != 1 {
		log.Fatalf("Expected listing to return 1 directory but found %d", len(bulkResponse.Files))
	}

	// List subdirectory
	log.Println("Listing subdirectory contents")
	bulkResponse, err = b.ListFileNamesWithPrefix("", 500, "test/", "/")
	check(err)
	if len(bulkResponse.Files) != len(files) {
		log.Fatalf("Expected listing to return %d files but found %d", len(files), len(bulkResponse.Files))
	}
}

func deleteFiles(b *backblaze.Bucket, files []*backblaze.File) {
	// Delete files
	log.Printf("Deleting %d files...", len(files))
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
