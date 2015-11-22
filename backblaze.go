package backblaze

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	B2_HOST = "https://api.backblaze.com"
	V1      = "/b2api/v1/"
	V2      = "/b2api/v2/"
)

type Credentials struct {
	AccountId      string
	ApplicationKey string
}

type Client struct {
	http.Client
	Credentials

	authorizationToken string
	apiUrl             string
	downloadUrl        string
}

type BucketType int

const (
	AllPublic BucketType = iota
	AllPrivate
)

type Bucket struct {
	AccountId string
	Id        string
	Name      string
	BucketType

	uploadUrl          *url.URL
	authorizationToken string

	client *Client
}

type File struct {
	Id            string
	Name          string
	AccountId     string
	BucketId      string
	ContentLength int64
	ContentType   string
	FileInfo      map[string]string
}

type ListFilesResponse struct {
	Files        []FileStatus
	NextFileName string
}

type ListFileVersionsResponse struct {
	Files          []FileStatus
	NextFileString string
	NextFileId     string
}

type FileAction int

const (
	Upload FileAction = iota
	Hide
)

type FileStatus struct {
	FileId   string
	FileName string
	FileAction
	Size int
}

// {
// 	"code": "codeValue",
//  "message": "messageValue",
//  "status": http_ret_status_int
// }
type B2Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (e *B2Error) Error() string {
	return e.Code + ": " + e.Message
}

// {
//   "accountId": "YOUR_ACCOUNT_ID",
//   "apiUrl": "https://api900.backblaze.com",
//   "authorizationToken": "2_20150807002553_443e98bf57f978fa58c284f8_24d25d99772e3ba927778b39c9b0198f412d2163_acct",
//   "downloadUrl": "https://f900.backblaze.com"
// }
type AuthorizeAccountResponse struct {
	AccountId          string `json:"accountId"`
	ApiUrl             string `json:"apiUrl"`
	AuthorizationToken string `json:"authorizationToken"`
	DownloadUrl        string `json:"downloadUrl"`
}

func NewClient(creds Credentials) (*Client, error) {
	c := &Client{
		Credentials: creds,
	}

	// Authorize account
	req, err := http.NewRequest("GET", B2_HOST+V1+"b2_authorize_account", nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(creds.AccountId, creds.ApplicationKey)

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	authResponse := &AuthorizeAccountResponse{}
	err = parseResponse(resp, authResponse)
	if err != nil {
		return nil, err
	}

	// Store token
	c.authorizationToken = authResponse.AuthorizationToken
	c.downloadUrl = authResponse.DownloadUrl
	c.apiUrl = authResponse.ApiUrl

	return c, nil
}

// Create an authorized request using the client's credentials
func (c *Client) authRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, c.apiUrl+V1+path, body)
	if err != nil {
		return nil, err
	}

	if c.authorizationToken != "" {
		req.Header.Add("Authorization", c.authorizationToken)
	}

	println("Request: " + req.URL.String())

	return req, nil
}

// Create an authorized GET request
func (c *Client) Get(path string) (*http.Response, error) {
	req, err := c.authRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

// create an authorized POST request
func (c *Client) Post(path string, body io.Reader) (*http.Response, error) {
	req, err := c.authRequest("POST", path, body)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

func parseError(resp *http.Response) error {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	println("Response: " + string(body))

	b2err := &B2Error{}
	json.Unmarshal(body, b2err)
	return b2err
}

func parseResponse(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()

	// Check response code
	switch resp.StatusCode {
	case 200: // Response is OK
	case 400: // BAD_REQUEST
		return parseError(resp)
	case 401:
		return errors.New("UNAUTHORIZED - The account ID is wrong, the account does not have B2 enabled, or the application key is not valid")
	default:
		if err := parseError(resp); err != nil {
			return err
		}
		return fmt.Errorf("Unrecognised status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	println("Response: " + string(body))

	return json.Unmarshal(body, result)
}

func (c *Client) CreateBucket(bucketName string, bucketType BucketType) *Bucket {
	return nil
}

func (c *Client) DeleteBucket(bucketId string) error {
	return nil
}

// {"accountId": "ACCOUNT_ID"}
type ListBucketsRequest struct {
	AccountId string `json:"accountId"`
}

// {
//     "buckets": [
//     {
//         "bucketId": "4a48fe8875c6214145260818",
//         "accountId": "30f20426f0b1",
//         "bucketName" : "Kitten Videos",
//         "bucketType": "allPrivate"
//     },
//     {
//         "bucketId" : "5b232e8875c6214145260818",
//         "accountId": "30f20426f0b1",
//         "bucketName": "Puppy Videos",
//         "bucketType": "allPublic"
//     },
//     {
//         "bucketId": "87ba238875c6214145260818",
//         "accountId": "30f20426f0b1",
//         "bucketName": "Vacation Pictures",
//         "bucketType" : "allPrivate"
//     } ]
// }
type ListBucketsResponse struct {
	Buckets []struct {
		BucketId   string `json:"bucketId"`
		AccountId  string `json:"accountId"`
		BucketName string `json:"bucketName"`
		BucketType string `json:"bucketType"`
	} `json:"buckets"`
}

func (c *Client) ListBuckets() ([]*Bucket, error) {
	request := &ListBucketsRequest{
		AccountId: c.AccountId,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	resp, err := c.Post("b2_list_buckets", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	response := &ListBucketsResponse{}
	err = parseResponse(resp, response)
	if err != nil {
		return nil, err
	}

	// Construct bucket list
	buckets := make([]*Bucket, len(response.Buckets))
	for i, v := range response.Buckets {
		buckets[i] = &Bucket{
			AccountId: v.AccountId,
			Id:        v.BucketId,
			Name:      v.BucketName,
			client:    c,
		}

		switch v.BucketType {
		case "allPublic":
			buckets[i].BucketType = AllPublic
		case "allPrivate":
			buckets[i].BucketType = AllPrivate
		default:
			return nil, errors.New("Uncrecognised bucket type: " + v.BucketType)
		}
	}

	return buckets, nil
}

func (c *Client) UpdateBucket(bucketId string, bucketType BucketType) *Bucket {
	return nil
}

func (c *Client) DownloadFileById(fileId string) (*File, io.Reader) {
	return nil, nil
}

func (c *Client) Bucket(bucketName string) (*Bucket, error) {

	// Lookup a bucket for the currently authorized client
	buckets, err := c.ListBuckets()
	if err != nil {
		return nil, err
	}

	for _, bucket := range buckets {
		if bucket.Name == bucketName {
			return bucket, nil
		}
	}

	return nil, nil
}

func (b *Bucket) ListFileNames(startFileName string, maxFileCount int) *ListFilesResponse {
	return nil
}

// {
//     "fileId" : "4_h4a48fe8875c6214145260818_f000000000000472a_d20140104_m032022_c001_v0000123_t0104",
//     "fileName" : "typing_test.txt",
//     "accountId" : "d522aa47a10f",
//     "bucketId" : "4a48fe8875c6214145260818",
//     "contentLength" : 46,
//     "contentSha1" : "bae5ed658ab3546aee12f23f36392f35dba1ebdd",
//     "contentType" : "text/plain",
//     "fileInfo" : {
//        "author" : "unknown"
//     }
// }
type UploadFileResponse struct {
	FileId        string            `json:"fileId"`
	FileName      string            `json:"fileName"`
	AccountId     string            `json:"accountId"`
	BucketId      string            `json:"bucketId"`
	ContentLength int64             `json:"contentLength"`
	ContentSha1   string            `json:"contentSha1"`
	ContentType   string            `json:"contentType"`
	FileInfo      map[string]string `json:"fileInfo"`
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

	file.Seek(0, 0)
	sha1Hash := hex.EncodeToString(hash.Sum(nil))

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

	result := &UploadFileResponse{}
	if err := parseResponse(resp, result); err != nil {
		b.uploadUrl = nil
		b.authorizationToken = ""
		return nil, err
	}

	if sha1Hash != result.ContentSha1 {
		return nil, errors.New("SHA1 of uploaded file does not match local hash")
	}

	return &File{
		Id:            result.FileId,
		Name:          result.FileName,
		AccountId:     result.AccountId,
		BucketId:      result.BucketId,
		ContentLength: result.ContentLength,
		ContentType:   result.ContentType,
		FileInfo:      result.FileInfo,
	}, nil
}

func (b *Bucket) GetFileInfo(fileId string) *File {
	return nil
}

func (b *Bucket) DownloadFileByName(fileName string) (*File, io.Reader) {
	return nil, nil
}

func (b *Bucket) ListAllFileVersions() *ListFileVersionsResponse {
	return nil
}

func (b *Bucket) ListFileVersions(startFileName string, startFileId string, maxFileCount int) *ListFileVersionsResponse {
	return nil
}

// {"bucketId": "BUCKET_ID"}
type GetUploadUrlRequest struct {
	BucketId string `json:"bucketId"`
}

// {
//     "bucketId" : "4a48fe8875c6214145260818",
//     "uploadUrl" : "https://pod-000-1005-03.backblaze.com/b2api/v1/b2_upload_file?cvt=c001_v0001005_t0027&bucket=4a48fe8875c6214145260818",
//     "authorizationToken" : "2_20151009170037_f504a0f39a0f4e657337e624_9754dde94359bd7b8f1445c8f4cc1a231a33f714_upld"
// }
type GetUploadUrlResponse struct {
	BucketId           string `json:"bucketId"`
	UploadUrl          string `json:"uploadUrl"`
	AuthorizationToken string `json:"authorizationToken"`
}

func (b *Bucket) GetUploadUrl() (*url.URL, error) {
	if b.uploadUrl == nil {
		request := &GetUploadUrlRequest{
			BucketId: b.Id,
		}

		body, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}
		resp, err := b.client.Post("b2_get_upload_url", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}

		response := &GetUploadUrlResponse{}
		err = parseResponse(resp, response)
		if err != nil {
			return nil, err
		}

		// Set bucket upload URL
		url, err := url.Parse(response.UploadUrl)
		if err != nil {
			return nil, err
		}
		b.uploadUrl = url
		b.authorizationToken = response.AuthorizationToken
	}
	return b.uploadUrl, nil
}

func (b *Bucket) HideFile(fileName string) *FileStatus {
	return nil
}
