//go:generate ffjson $GOFILE

package backblaze

import (
	"encoding/json"
)

// B2Error encapsulates an error message returned by the B2 API.
//
// Failures to connect to the B2 servers, and networking problems in general can cause errors
type B2Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (e B2Error) Error() string {
	return e.Code + ": " + e.Message
}

// IsFatal returns true if this error represents
// an error which can't be recovered from by retrying
func (e *B2Error) IsFatal() bool {
	switch {
	case e.Status == 401: // Unauthorized
		switch e.Code {
		case "expired_auth_token":
			return false
		case "missing_auth_token", "bad_auth_token":
			return true
		default:
			return true
		}
	case e.Status == 408: // Timeout
		return false
	case e.Status >= 500 && e.Status < 600: // Server error
		return false
	default:
		return true
	}
}

type authorizeAccountResponse struct {
	AccountID          string `json:"accountId"`
	APIEndpoint        string `json:"apiUrl"`
	AuthorizationToken string `json:"authorizationToken"`
	DownloadURL        string `json:"downloadUrl"`
}

type accountRequest struct {
	ID string `json:"accountId"`
}

// BucketType defines the security setting for a bucket
type BucketType string

// Buckets can be either public, private, or snapshot
const (
	AllPublic  BucketType = "allPublic"
	AllPrivate BucketType = "allPrivate"
	Snapshot   BucketType = "snapshot"
)

// LifecycleRule instructs the B2 service to automatically hide and/or delete old files.
// You can set up rules to do things like delete old versions of files 30 days after a newer version was uploaded.
type LifecycleRule struct {
	DaysFromUploadingToHiding int    `json:"daysFromUploadingToHiding"`
	DaysFromHidingToDeleting  int    `json:"daysFromHidingToDeleting"`
	FileNamePrefix            string `json:"fileNamePrefix"`
}

// BucketInfo describes a bucket
type BucketInfo struct {
	// The account that the bucket is in.
	AccountID string `json:"accountId"`

	// The unique ID of the bucket.
	ID string `json:"bucketId"`

	// User-defined information to be stored with the bucket.
	Info json.RawMessage `json:"bucketInfo"`

	// The name to give the new bucket.
	// Bucket names must be a minimum of 6 and a maximum of 50 characters long, and must be globally unique;
	// two different B2 accounts cannot have buckets with the name name. Bucket names can consist of: letters,
	// digits, and "-". Bucket names cannot start with "b2-"; these are reserved for internal Backblaze use.
	Name string `json:"bucketName"`

	// Either "allPublic", meaning that files in this bucket can be downloaded by anybody, or "allPrivate",
	// meaning that you need a bucket authorization token to download the files.
	BucketType BucketType `json:"bucketType"`

	// The initial list of lifecycle rules for this bucket.
	LifecycleRules []LifecycleRule `json:"lifecycleRules"`

	// A counter that is updated every time the bucket is modified.
	Revision int `json:"revision"`
}

type bucketRequest struct {
	ID string `json:"bucketId"`
}

type createBucketRequest struct {
	AccountID  string `json:"accountId"`
	BucketName string `json:"bucketName"`
	BucketType `json:"bucketType"`
}

type deleteBucketRequest struct {
	AccountID string `json:"accountId"`
	BucketID  string `json:"bucketId"`
}

type updateBucketRequest struct {
	ID         string `json:"bucketId"`
	BucketType `json:"bucketType"`
}

type getUploadURLResponse struct {
	BucketID           string `json:"bucketId"`
	UploadURL          string `json:"uploadUrl"`
	AuthorizationToken string `json:"authorizationToken"`
}

type listBucketsResponse struct {
	Buckets []*BucketInfo `json:"buckets"`
}

type fileRequest struct {
	ID string `json:"fileId"`
}

type fileVersionRequest struct {
	Name string `json:"fileName"`
	ID   string `json:"fileId"`
}

// File descibes a file stored in a B2 bucket
type File struct {
	ID              string            `json:"fileId"`
	Name            string            `json:"fileName"`
	AccountID       string            `json:"accountId"`
	BucketID        string            `json:"bucketId"`
	ContentLength   int64             `json:"contentLength"`
	ContentSha1     string            `json:"contentSha1"`
	ContentType     string            `json:"contentType"`
	FileInfo        map[string]string `json:"fileInfo"`
	Action          FileAction        `json:"action"`
	Size            int               `json:"size"` // Deprecated - same as ContentSha1
	UploadTimestamp int64             `json:"uploadTimestamp"`
}

type listFilesRequest struct {
	BucketID      string `json:"bucketId"`
	StartFileName string `json:"startFileName"`
	MaxFileCount  int    `json:"maxFileCount"`
	Prefix        string `json:"prefix,omitempty"`
	Delimiter     string `json:"delimiter,omitempty"`
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
	MaxFileCount  int    `json:"maxFileCount,omitempty"`
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

// FileStatus is now identical to File in repsonses from ListFileNames and ListFileVersions
type FileStatus struct {
	File
}
