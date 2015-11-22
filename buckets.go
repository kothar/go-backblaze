package backblaze

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
)

type BucketType string

const (
	AllPublic  BucketType = "allPublic"
	AllPrivate            = "allPrivate"
)

type Bucket struct {
	Id         string `json:"bucketId"`
	AccountId  string `json:"accountId"`
	Name       string `json:"bucketName"`
	BucketType `json:"bucketType"`

	uploadUrl          *url.URL `json:"-"`
	authorizationToken string   `json:"-"`
	client             *Client  `json:"-"`
}

type ListFilesRequest struct {
	BucketId string `json:"bucketId"`
}

type GetUploadUrlRequest struct {
	BucketId string `json:"bucketId"`
}

type GetUploadUrlResponse struct {
	BucketId           string `json:"bucketId"`
	UploadUrl          string `json:"uploadUrl"`
	AuthorizationToken string `json:"authorizationToken"`
}

type ListBucketsRequest struct {
	AccountId string `json:"accountId"`
}

type ListBucketsResponse struct {
	Buckets []*Bucket `json:"buckets"`
}

func (c *Client) CreateBucket(bucketName string, bucketType BucketType) *Bucket {
	return nil
}

func (c *Client) DeleteBucket(bucketId string) error {
	return nil
}

func (c *Client) ListBuckets() ([]*Bucket, error) {
	request := &ListBucketsRequest{
		AccountId: c.AccountId,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	resp, err := c.Post(c.apiUrl+V1+"b2_list_buckets", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	response := &ListBucketsResponse{}
	err = parseResponse(resp, response)
	if err != nil {
		return nil, err
	}

	// Construct bucket list
	for _, v := range response.Buckets {
		v.client = c

		switch v.BucketType {
		case "allPublic":
		case "allPrivate":
		default:
			return nil, errors.New("Uncrecognised bucket type: " + string(v.BucketType))
		}
	}

	return response.Buckets, nil
}

func (c *Client) UpdateBucket(bucketId string, bucketType BucketType) *Bucket {
	return nil
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

func (b *Bucket) GetUploadUrl() (*url.URL, error) {
	if b.uploadUrl == nil {
		request := &GetUploadUrlRequest{
			BucketId: b.Id,
		}

		body, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}
		resp, err := b.client.Post(b.client.apiUrl+V1+"b2_get_upload_url", bytes.NewReader(body))
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
