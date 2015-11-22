package backblaze

import (
	"errors"
	"net/url"
)

type BucketType string

const (
	AllPublic  BucketType = "allPublic"
	AllPrivate BucketType = "allPrivate"
)

type Bucket struct {
	Id         string `json:"bucketId"`
	AccountId  string `json:"accountId"`
	Name       string `json:"bucketName"`
	BucketType `json:"bucketType"`

	uploadUrl          *url.URL `json:"-"`
	authorizationToken string   `json:"-"`
	b2                 *B2      `json:"-"`
}

type BucketRequest struct {
	Id string `json:"bucketId"`
}

type CreateBucketRequest struct {
	AccountId  string `json:"accountId"`
	BucketName string `json:"bucketName"`
	BucketType `json:"bucketType"`
}

type DeleteBucketRequest struct {
	AccountId string `json:"accountId"`
	BucketId  string `json:"bucketId"`
}

type UpdateBucketRequest struct {
	Id         string `json:"bucketId"`
	BucketType `json:"bucketType"`
}

type GetUploadUrlResponse struct {
	BucketId           string `json:"bucketId"`
	UploadUrl          string `json:"uploadUrl"`
	AuthorizationToken string `json:"authorizationToken"`
}

type AccountRequest struct {
	Id string `json:"accountId"`
}

type ListBucketsResponse struct {
	Buckets []*Bucket `json:"buckets"`
}

// Creates a new bucket. A bucket belongs to the account used to create it.
//
// Buckets can be named. The name must be globally unique. No account can use
// a bucket with the same name. Buckets are assigned a unique bucketId which
// is used when uploading, downloading, or deleting files.
func (b *B2) CreateBucket(bucketName string, bucketType BucketType) (*Bucket, error) {
	request := &CreateBucketRequest{
		AccountId:  b.AccountId,
		BucketName: bucketName,
		BucketType: bucketType,
	}
	response := &Bucket{b2: b}

	if err := b.apiRequest("b2_create_bucket", request, response); err != nil {
		return nil, err
	}

	return response, nil
}

// Deletes the bucket specified. Only buckets that contain no version of any
// files can be deleted.
func (b *B2) DeleteBucket(bucketId string) (*Bucket, error) {
	request := &DeleteBucketRequest{
		AccountId: b.AccountId,
		BucketId:  bucketId,
	}
	response := &Bucket{b2: b}

	if err := b.apiRequest("b2_delete_bucket", request, response); err != nil {
		return nil, err
	}

	return response, nil
}

// Deletes the bucket specified. Only buckets that contain no version of any
// files can be deleted.
func (b *Bucket) Delete() error {
	_, error := b.b2.DeleteBucket(b.Id)
	return error
}

// Lists buckets associated with an account, in alphabetical order by bucket
// ID.
func (b *B2) ListBuckets() ([]*Bucket, error) {
	request := &AccountRequest{
		Id: b.AccountId,
	}
	response := &ListBucketsResponse{}

	if err := b.apiRequest("b2_list_buckets", request, response); err != nil {
		return nil, err
	}

	// Construct bucket list
	for _, bucket := range response.Buckets {
		bucket.b2 = b

		switch bucket.BucketType {
		case "allPublic":
		case "allPrivate":
		default:
			return nil, errors.New("Uncrecognised bucket type: " + string(bucket.BucketType))
		}
	}

	return response.Buckets, nil
}

// Update an existing bucket.
func (b *B2) UpdateBucket(bucketId string, bucketType BucketType) (*Bucket, error) {
	request := &UpdateBucketRequest{
		Id:         bucketId,
		BucketType: bucketType,
	}
	response := &Bucket{b2: b}

	if err := b.apiRequest("b2_update_bucket", request, response); err != nil {
		return nil, err
	}

	return response, nil
}

// Lookup a bucket for the currently authorized client
func (b *B2) Bucket(bucketName string) (*Bucket, error) {

	buckets, err := b.ListBuckets()
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

// Gets an URL to use for uploading files.
//
// When you upload a file to B2, you must call b2_get_upload_url first to get
// the URL for uploading directly to the place where the file will be stored.
func (b *Bucket) GetUploadUrl() (*url.URL, error) {
	if b.uploadUrl == nil {
		request := &BucketRequest{
			Id: b.Id,
		}

		response := &GetUploadUrlResponse{}
		if err := b.b2.apiRequest("b2_get_upload_url", request, response); err != nil {
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
