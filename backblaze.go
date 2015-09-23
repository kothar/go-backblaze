package backblaze

import (
	"net/url"
)

type Credentials struct {
	AccountId      string
	ApplicationKey string
}

type Client struct {
	Credentials
	authorizationToken string
	apiUrl             url.URL
	downloadUrl        url.URL
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
	Files FileStatus[]
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
	Hide FileAction = iota
)

type FileStatus struct {
	FileId   string
	FileName string
	FileAction
	Size     int
}

type UploadUrl struct {
	BucketId  string
	UploadUrl url.URL
}

func NewClient(creds Credentials) *Client {
	// Authorize account

	return &Client{
		Credentials: creds,
	}
}

func (c *Client) CreateBucket(bucketName string, bucketType BucketType) Bucket {

}

func (c *Client) DeleteBucket(bucketId string) Bucket {

}

func (c *Client) ListBuckets() []Bucket {

}

func (c *Client) UpdateBucket(bucketId string, bucketType BucketType) Bucket {

}

func (c *Client) DownloadFileById(fileId string) File {

}

func (b *Bucket) ListFileNames(startFileName string, maxFileCount int) ListFilesResponse {

}

func (b *Bucket) UploadFile(file File) File {
	uploadUrl := b.GetUploadUrl()
}

func (b *Bucket) GetFileInfo(fileId string) File {

}

func (b *Bucket) DownloadFileByName(fileName string) File {

}

func (b *Bucket) ListAllFileVersions() ListFileVersionsResponse {

}

func (b *Bucket) ListFileVersions(startFileName string, startFileId string, maxFileCount int) ListFileVersionsResponse {

}

func (b *Bucket) GetUploadUrl() UploadUrl {

}

func (b *Bucket) HideFile(fileName string) FileStatus {

}
