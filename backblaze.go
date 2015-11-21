package backblaze

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	V1 = "https://api.backblaze.com/b2api/v1/"
)

type Credentials struct {
	AccountId      string
	ApplicationKey string
}

type Client struct {
	Credentials
	authorizationToken string
	apiUrl             string
	downloadUrl        string
	httpClient         *http.Client
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

	client *Client
}

type File struct {
	Id            string
	Name          string
	AccountId     string
	BucketId      string
	ContentLength int
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

type UploadUrl struct {
	BucketId  string
	UploadUrl *url.URL
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
	httpClient := &http.Client{}

	// Authorize account
	req, err := http.NewRequest("GET", V1+"b2_authorize_account", nil)
	req.SetBasicAuth(creds.AccountId, creds.ApplicationKey)

	authResponse := &AuthorizeAccountResponse{}
	err = getJson(httpClient, req, authResponse)
	if err != nil {
		return nil, err
	}

	return &Client{
		Credentials:        creds,
		authorizationToken: authResponse.AuthorizationToken,
		apiUrl:             V1,
		downloadUrl:        authResponse.DownloadUrl,
		httpClient:         httpClient,
	}, nil
}

func getJson(httpClient *http.Client, req *http.Request, result interface{}) error {
	println("Request: " + req.URL.String())
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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

func (c *Client) ListBuckets() []*Bucket {
	return nil
}

func (c *Client) UpdateBucket(bucketId string, bucketType BucketType) *Bucket {
	return nil
}

func (c *Client) DownloadFileById(fileId string) (*File, io.Reader) {
	return nil, nil
}

func (c *Client) Bucket(bucketName string) (*Bucket, error) {
	return &Bucket{
		AccountId: c.Credentials.AccountId,
		Name:      bucketName,
		client:    c,
	}, nil
}

func (b *Bucket) ListFileNames(startFileName string, maxFileCount int) *ListFilesResponse {
	return nil
}

func (b *Bucket) UploadFile(name string, file io.Reader) (*File, error) {
	uploadUrl, err := b.GetUploadUrl()
	if err != nil {
		return nil, err
	}

	print("Upload: " + b.Name + "/" + name + " (" + uploadUrl.UploadUrl.String() + ")\n")
	return nil, nil
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

func (b *Bucket) GetUploadUrl() (*UploadUrl, error) {
	url, err := url.Parse("https://pod-000-1005-03.backblaze.com/b2api/v1/b2_upload_file?cvt=c001_v0001005_t0027&bucket=" + b.Id)
	if err != nil {
		return nil, err
	}

	return &UploadUrl{
		BucketId:  b.Id,
		UploadUrl: url,
	}, nil
}

func (b *Bucket) HideFile(fileName string) *FileStatus {
	return nil
}
