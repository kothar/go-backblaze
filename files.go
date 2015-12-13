package backblaze

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type fileRequest struct {
	ID string `json:"fileId"`
}

type fileVersionRequest struct {
	Name string `json:"fileName"`
	ID   string `json:"fileId"`
}

// File descibes a file stored in a B2 bucket
type File struct {
	ID            string            `json:"fileId"`
	Name          string            `json:"fileName"`
	AccountID     string            `json:"accountId"`
	BucketID      string            `json:"bucketId"`
	ContentLength int64             `json:"contentLength"`
	ContentSha1   string            `json:"contentSha1"`
	ContentType   string            `json:"contentType"`
	FileInfo      map[string]string `json:"fileInfo"`
}

type listFilesRequest struct {
	BucketID      string `json:"bucketId"`
	StartFileName string `json:"startFileName"`
	MaxFileCount  int    `json:"maxFileCount"`
}

// ListFilesResponse lists a page of files stored in a B2 bucket
type ListFilesResponse struct {
	Files        []FileStatus `json:"files"`
	NextFileName string       `json:"nextFileName"`
}

type listFileVersionsRequest struct {
	BucketID      string `json:"bucketId"`
	StartFileName string `json:"startFileName,omitempty"`
	StartFileID   string `json:"startFileId,omitempty"`
	MaxFileCount  int    `json:"maxFileCount"`
}

// ListFileVersionsResponse lists a page of file versions stored in a B2 bucket
type ListFileVersionsResponse struct {
	Files        []FileStatus `json:"files"`
	NextFileName string       `json:"nextFileName"`
	NextFileID   string       `json:"nextFileId"`
}

type hideFileRequest struct {
	BucketID string `json:"bucketId"`
	FileName string `json:"fileName"`
}

// FileAction indicates the current status of a file in a B2 bucket
type FileAction string

// Files can be either uploads (visible) or hidden.
//
// Hiding a file makes it look like the file has been deleted, without
// removing any of the history. It adds a new version of the file that is a
// marker saying the file is no longer there.
const (
	Upload FileAction = "upload"
	Hide   FileAction = "hide"
)

// FileStatus describes minimal metadata about a file in a B2 bucket.
// It is returned by the ListFileNames and ListFileVersions methods
type FileStatus struct {
	FileAction      `json:"action"`
	ID              string `json:"fileId"`
	Name            string `json:"fileName"`
	Size            int    `json:"size"`
	UploadTimestamp int64  `json:"uploadTimestamp"`
}

// ListFileNames lists the names of all files in a bucket, starting at a given name.
func (b *Bucket) ListFileNames(startFileName string, maxFileCount int) (*ListFilesResponse, error) {
	request := &listFilesRequest{
		BucketID:      b.ID,
		StartFileName: startFileName,
		MaxFileCount:  maxFileCount,
	}
	response := &ListFilesResponse{}

	if err := b.b2.apiRequest("b2_list_file_names", request, response); err != nil {
		return nil, err
	}

	return response, nil
}

// UploadFile uploads a file to B2, returning its unique file ID.
// This method computes the hash of the file before passing it to UploadHashedFile
func (b *Bucket) UploadFile(name string, meta map[string]string, file io.Reader) (*File, error) {

	// Hash the upload
	hash := sha1.New()

	var reader io.Reader
	var contentLength int64
	if r, ok := file.(io.ReadSeeker); ok {
		// If the input is seekable, just hash then seek back to the beginning
		written, err := io.Copy(hash, r)
		if err != nil {
			return nil, err
		}
		r.Seek(0, 0)
		reader = r
		contentLength = written
	} else {
		// If the input is not seekable, buffer it while hashing, and use the buffer as input
		buffer := &bytes.Buffer{}
		r := io.TeeReader(file, buffer)

		written, err := io.Copy(hash, r)
		if err != nil {
			return nil, err
		}
		reader = buffer
		contentLength = written
	}

	sha1Hash := hex.EncodeToString(hash.Sum(nil))
	return b.UploadHashedFile(name, meta, reader, sha1Hash, contentLength)
}

// UploadHashedFile Uploads a file to B2, returning its unique file ID.
func (b *Bucket) UploadHashedFile(name string, meta map[string]string, file io.Reader, sha1Hash string, contentLength int64) (*File, error) {

	_, err := b.getUploadURL()
	if err != nil {
		return nil, err
	}

	if b.b2.Debug {
		fmt.Printf("         Upload: %s/%s\n", b.Name, name)
		fmt.Printf("           SHA1: %s\n", sha1Hash)
		fmt.Printf("  ContentLength: %d\n", contentLength)
	}

	// Create authorized request
	req, err := http.NewRequest("POST", b.uploadURL.String(), file)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", b.authorizationToken)

	// Set file metadata
	req.ContentLength = contentLength
	req.Header.Add("Content-Type", "b2/x-auto")
	req.Header.Add("X-Bz-File-Name", url.QueryEscape(name))
	req.Header.Add("X-Bz-Content-Sha1", sha1Hash)

	if meta != nil {
		for k, v := range meta {
			req.Header.Add("X-Bz-Info-"+url.QueryEscape(k), url.QueryEscape(v))
		}
	}

	resp, err := b.b2.httpClient.Do(req)
	if err != nil {
		b.uploadURL = nil
		b.authorizationToken = ""
		return nil, err
	}

	result := &File{}
	if err := b.b2.parseResponse(resp, result); err != nil {
		b.uploadURL = nil
		b.authorizationToken = ""
		return nil, err
	}

	if sha1Hash != result.ContentSha1 {
		return nil, errors.New("SHA1 of uploaded file does not match local hash")
	}

	return result, nil
}

// GetFileInfo retrieves information about one file stored in B2.
func (b *Bucket) GetFileInfo(fileID string) (*File, error) {
	request := &fileRequest{
		ID: fileID,
	}
	response := &File{}

	if err := b.b2.apiRequest("b2_get_file_info", request, response); err != nil {
		return nil, err
	}

	return response, nil
}

// DownloadFileByID downloads a file from B2 using its unique ID
func (b *B2) DownloadFileByID(fileID string) (*File, io.ReadCloser, error) {
	request := &fileRequest{
		ID: fileID,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, nil, err
	}

	resp, err := b.post(b.apiEndpoint+v1+"b2_download_file_by_id", bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}

	return b.downloadFile(resp)
}

// FileURL returns a URL which may be used to dowload the laterst version of a file.
// This will only work for public URLs unless the correct authorization header is provided.
func (b *Bucket) FileURL(fileName string) string {
	return b.b2.downloadURL + "/file/" + b.Name + "/" + fileName
}

// DownloadFileByName Downloads one file by providing the name of the bucket and the name of the
// file.
func (b *Bucket) DownloadFileByName(fileName string) (*File, io.ReadCloser, error) {

	url := b.FileURL(fileName)

	resp, err := b.b2.get(url)
	if err != nil {
		return nil, nil, err
	}

	return b.b2.downloadFile(resp)
}

func (b *B2) downloadFile(resp *http.Response) (*File, io.ReadCloser, error) {
	switch resp.StatusCode {
	case 200:
	default:
		if err := b.parseError(resp); err != nil {
			resp.Body.Close()
			return nil, nil, err
		}
		resp.Body.Close()
		return nil, nil, fmt.Errorf("Unrecognised status code: %d", resp.StatusCode)
	}

	name, err := url.QueryUnescape(resp.Header.Get("X-Bz-File-Name"))
	if err != nil {
		resp.Body.Close()
		return nil, nil, err
	}

	file := &File{
		ID:          resp.Header.Get("X-Bz-File-Id"),
		Name:        name,
		ContentSha1: resp.Header.Get("X-Bz-Content-Sha1"),
		ContentType: resp.Header.Get("Content-Type"),
		FileInfo:    make(map[string]string),
	}

	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		resp.Body.Close()
		return nil, nil, err
	}
	file.ContentLength = size

	for k, v := range resp.Header {
		if strings.HasPrefix(k, "X-Bz-Info-") {
			key, err := url.QueryUnescape(k[len("X-Bz-Info-"):])
			if err != nil {
				key = k[len("X-Bz-Info-"):]
				log.Printf("Unable to decode key: %q", key)
			}

			value, err := url.QueryUnescape(v[0])
			if err != nil {
				value = v[0]
				log.Printf("Unable to decode value: %q", value)
			}
			file.FileInfo[key] = value
		}
	}

	return file, resp.Body, nil
}

// ListFileVersions lists all of the versions of all of the files contained in
// one bucket, in alphabetical order by file name, and by reverse of date/time
// uploaded for versions of files with the same name.
func (b *Bucket) ListFileVersions(startFileName, startFileID string, maxFileCount int) (*ListFileVersionsResponse, error) {
	request := &listFileVersionsRequest{
		BucketID:      b.ID,
		StartFileName: startFileName,
		StartFileID:   startFileID,
		MaxFileCount:  maxFileCount,
	}
	response := &ListFileVersionsResponse{}

	if err := b.b2.apiRequest("b2_list_file_versions", request, response); err != nil {
		return nil, err
	}

	return response, nil
}

// DeleteFileVersion deletes one version of a file from B2.
//
// If the version you delete is the latest version, and there are older
// versions, then the most recent older version will become the current
// version, and be the one that you'll get when downloading by name. See the
// File Versions page for more details.
func (b *Bucket) DeleteFileVersion(fileName, fileID string) (*FileStatus, error) {
	request := &fileVersionRequest{
		Name: fileName,
		ID:   fileID,
	}
	response := &FileStatus{}

	if err := b.b2.apiRequest("b2_delete_file_version", request, response); err != nil {
		return nil, err
	}

	return response, nil
}

// HideFile hides a file so that downloading by name will not find the file,
// but previous versions of the file are still stored. See File Versions about
// what it means to hide a file.
func (b *Bucket) HideFile(fileName string) (*FileStatus, error) {
	request := &hideFileRequest{
		BucketID: b.ID,
		FileName: fileName,
	}
	response := &FileStatus{}

	if err := b.b2.apiRequest("b2_hide_file", request, response); err != nil {
		return nil, err
	}

	return response, nil
}
