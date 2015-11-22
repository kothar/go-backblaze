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

type ListFilesResponse struct {
	Files        []*FileStatus `json:"files"`
	NextFileName string        `json:"nextFileName"`
}

type ListFileVersionsResponse struct {
	Files          []FileStatus
	NextFileString string
	NextFileId     string
}

type FileAction string

const (
	Upload FileAction = "upload"
	Hide              = "hide"
)

type FileStatus struct {
	FileAction      `json:"action"`
	Id              string `json:"fileId"`
	Name            string `json:"fileName"`
	Size            int    `json:"size"`
	UploadTimestamp int64  `json:"uploadTimestamp"`
}

func (c *Client) DownloadFileById(fileId string) (*File, io.Reader) {
	return nil, nil
}

func (b *Bucket) ListFileNames(startFileName string, maxFileCount int) (*ListFilesResponse, error) {
	request := &ListFilesRequest{
		BucketId: b.Id,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	resp, err := b.client.Post(b.client.apiUrl+V1+"b2_list_file_names", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	response := &ListFilesResponse{}
	err = parseResponse(resp, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (b *Bucket) UploadFile(name string, file io.ReadSeeker) (*File, error) {
	_, err := b.GetUploadUrl()
	if err != nil {
		return nil, err
	}

	println("Upload: " + b.Name + "/" + name)

	// Hash the upload
	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}
	sha1Hash := hex.EncodeToString(hash.Sum(nil))
	println("  SHA1: " + sha1Hash)

	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	// Create authorized request
	req, err := http.NewRequest("POST", b.uploadUrl.String(), file)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", b.authorizationToken)

	// Set file metadata
	req.Header.Add("X-Bz-File-Name", name)
	req.Header.Add("Content-Type", "b2/x-auto")
	req.Header.Add("X-Bz-Content-Sha1", sha1Hash)

	resp, err := b.client.Do(req)
	if err != nil {
		b.uploadUrl = nil
		b.authorizationToken = ""
		return nil, err
	}

	result := &File{}
	if err := parseResponse(resp, result); err != nil {
		b.uploadUrl = nil
		b.authorizationToken = ""
		return nil, err
	}

	if sha1Hash != result.ContentSha1 {
		return nil, errors.New("SHA1 of uploaded file does not match local hash")
	}

	return result, nil
}

func (b *Bucket) GetFileInfo(fileId string) *File {
	return nil
}

func (b *Bucket) DownloadFileByName(fileName string) (*File, io.ReadCloser, error) {

	url := b.client.downloadUrl + "/file/" + b.Name + "/" + fileName

	resp, err := b.client.Get(url)
	if err != nil {
		return nil, nil, err
	}

	switch resp.StatusCode {
	case 200:
	default:
		if err := parseError(resp); err != nil {
			return nil, nil, err
		}
		return nil, nil, fmt.Errorf("Unrecognised status code: %d", resp.StatusCode)
	}

	file := &File{
		AccountId:   b.AccountId,
		BucketId:    b.Id,
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

func (b *Bucket) ListAllFileVersions() *ListFileVersionsResponse {
	return nil
}

func (b *Bucket) ListFileVersions(startFileName string, startFileId string, maxFileCount int) *ListFileVersionsResponse {
	return nil
}

func (b *Bucket) HideFile(fileName string) *FileStatus {
	return nil
}
