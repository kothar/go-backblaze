package backblaze

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type FileRequest struct {
	Id string `json:"fileId"`
}

type FileVersionRequest struct {
	Name string `json:"fileName"`
	Id   string `json:"fileId"`
}

type File struct {
	Id            string            `json:"fileId"`
	Name          string            `json:"fileName"`
	AccountId     string            `json:"accountId"`
	BucketId      string            `json:"bucketId"`
	ContentLength int64             `json:"contentLength"`
	ContentSha1   string            `json:"contentSha1"`
	ContentType   string            `json:"contentType"`
	FileInfo      map[string]string `json:"fileInfo"`
}

type ListFilesRequest struct {
	BucketId      string `json:"bucketId"`
	StartFileName string `json:"startFileName"`
	MaxFileCount  int    `json:"maxFileCount"`
}

type ListFilesResponse struct {
	Files        []FileStatus `json:"files"`
	NextFileName string       `json:"nextFileName"`
}

type ListFileVersionsRequest struct {
	BucketId      string `json:"bucketId"`
	StartFileName string `json:"startFileName,omitempty"`
	StartFileId   string `json:"startFileId,omitempty"`
	MaxFileCount  int    `json:"maxFileCount"`
}

type ListFileVersionsResponse struct {
	Files        []FileStatus `json:"files"`
	NextFileName string       `json:"nextFileName"`
	NextFileId   string       `json:"nextFileId"`
}

type HideFileRequest struct {
	BucketId string `json:"bucketId"`
	FileName string `json:"fileName"`
}

type FileAction string

const (
	Upload FileAction = "upload"
	Hide   FileAction = "hide"
)

type FileStatus struct {
	FileAction      `json:"action"`
	Id              string `json:"fileId"`
	Name            string `json:"fileName"`
	Size            int    `json:"size"`
	UploadTimestamp int64  `json:"uploadTimestamp"`
}

// Downloads one file from B2.
func (b *B2) DownloadFileById(fileId string) (*File, io.ReadCloser, error) {
	request := &FileRequest{
		Id: fileId,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, nil, err
	}

	resp, err := b.post(b.apiUrl+V1+"b2_download_file_by_id", bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}

	return b.downloadFile(resp)
}

// Lists the names of all files in a bucket, starting at a given name.
func (b *Bucket) ListFileNames(startFileName string, maxFileCount int) (*ListFilesResponse, error) {
	request := &ListFilesRequest{
		BucketId:      b.Id,
		StartFileName: startFileName,
		MaxFileCount:  maxFileCount,
	}
	response := &ListFilesResponse{}

	if err := b.b2.apiRequest("b2_list_file_names", request, response); err != nil {
		return nil, err
	}

	return response, nil
}

// Uploads one file to B2, returning its unique file ID.
func (b *Bucket) UploadFile(name string, meta map[string]string, file io.Reader) (*File, error) {
	_, err := b.GetUploadUrl()
	if err != nil {
		return nil, err
	}

	if b.b2.Debug {
		println("Upload: " + b.Name + "/" + name)
	}

	// Hash the upload
	hash := sha1.New()
	buffer := &bytes.Buffer{}
	r := io.TeeReader(file, buffer)

	if _, err := io.Copy(hash, r); err != nil {
		return nil, err
	}
	sha1Hash := hex.EncodeToString(hash.Sum(nil))
	if b.b2.Debug {
		println("  SHA1: " + sha1Hash)
	}

	// Create authorized request
	req, err := http.NewRequest("POST", b.uploadUrl.String(), buffer)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", b.authorizationToken)

	// Set file metadata
	req.Header.Add("X-Bz-File-Name", name)
	req.Header.Add("Content-Type", "b2/x-auto")
	req.Header.Add("X-Bz-Content-Sha1", sha1Hash)

	if meta != nil {
		for k, v := range meta {
			req.Header.Add("X-Bz-Info-"+k, v)
		}
	}

	resp, err := b.b2.httpClient.Do(req)
	if err != nil {
		b.uploadUrl = nil
		b.authorizationToken = ""
		return nil, err
	}

	result := &File{}
	if err := b.b2.parseResponse(resp, result); err != nil {
		b.uploadUrl = nil
		b.authorizationToken = ""
		return nil, err
	}

	if sha1Hash != result.ContentSha1 {
		return nil, errors.New("SHA1 of uploaded file does not match local hash")
	}

	return result, nil
}

// Gets information about one file stored in B2.
func (b *Bucket) GetFileInfo(fileId string) (*File, error) {
	request := &FileRequest{
		Id: fileId,
	}
	response := &File{}

	if err := b.b2.apiRequest("b2_get_file_info", request, response); err != nil {
		return nil, err
	}

	return response, nil
}

// Downloads one file by providing the name of the bucket and the name of the
// file.
func (b *Bucket) DownloadFileByName(fileName string) (*File, io.ReadCloser, error) {

	url := b.b2.downloadUrl + "/file/" + b.Name + "/" + fileName

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
			return nil, nil, err
		}
		return nil, nil, fmt.Errorf("Unrecognised status code: %d", resp.StatusCode)
	}

	file := &File{
		Id:          resp.Header.Get("X-Bz-File-Id"),
		Name:        resp.Header.Get("X-Bz-File-Name"),
		ContentSha1: resp.Header.Get("X-Bz-Content-Sha1"),
		ContentType: resp.Header.Get("Content-Type"),
		FileInfo:    make(map[string]string),
	}

	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return nil, nil, err
	}
	file.ContentLength = size

	for k, v := range resp.Header {
		if strings.HasPrefix(k, "X-Bz-Info-") {
			file.FileInfo[k[len("X-Bz-Info-"):]] = v[0]
		}
	}

	return file, resp.Body, nil
}

// Lists all of the versions of all of the files contained in one bucket, in
// alphabetical order by file name, and by reverse of date/time uploaded for
// versions of files with the same name.
func (b *Bucket) ListFileVersions(startFileName, startFileId string, maxFileCount int) (*ListFileVersionsResponse, error) {
	request := &ListFileVersionsRequest{
		BucketId:      b.Id,
		StartFileName: startFileName,
		StartFileId:   startFileId,
		MaxFileCount:  maxFileCount,
	}
	response := &ListFileVersionsResponse{}

	if err := b.b2.apiRequest("b2_list_file_versions", request, response); err != nil {
		return nil, err
	}

	return response, nil
}

// Deletes one version of a file from B2.
//
// If the version you delete is the latest version, and there are older
// versions, then the most recent older version will become the current
// version, and be the one that you'll get when downloading by name. See the
// File Versions page for more details.
func (b *Bucket) DeleteFileVersion(fileName, fileId string) (*FileStatus, error) {
	request := &FileVersionRequest{
		Name: fileName,
		Id:   fileId,
	}
	response := &FileStatus{}

	if err := b.b2.apiRequest("b2_delete_file_version", request, response); err != nil {
		return nil, err
	}

	return response, nil
}

// Hides a file so that downloading by name will not find the file, but
// previous versions of the file are still stored. See File Versions about
// what it means to hide a file.
func (b *Bucket) HideFile(fileName string) (*FileStatus, error) {
	request := &HideFileRequest{
		BucketId: b.Id,
		FileName: fileName,
	}
	response := &FileStatus{}

	if err := b.b2.apiRequest("b2_hide_file", request, response); err != nil {
		return nil, err
	}

	return response, nil
}
