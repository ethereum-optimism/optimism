package s3

import (
	"encoding/json"
	"errors"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"

	// "context"
	// "fmt"
	// "io"
	// "net/http"
	// "net/url"
	// "os"
	// "path/filepath"
	// "strings"
	// "sync"
	// "time"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3StorageImpl struct {
	uploader *s3manager.Uploader
	bucket   string
}

func (s *S3StorageImpl) SaveBlobs(hash common.Hash, blobs []*eth.BlobAndMetadata) error {
	blobsData, err := json.Marshal(blobs)
	if err != nil {
		return err
	}
	// Create a new AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-west-2"),                         // Replace with your desired region
		Credentials: credentials.NewSharedCredentials("", "default"), // Replace with your credentials profile
	})
	if err != nil {
		return err
	}

	uploader := s3manager.NewUploader(sess)

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s.bucket),

		// Can also use the `filepath` standard library package to modify the
		// filename as need for an S3 object key. Such as turning absolute path
		// to a relative path.
		Key: aws.String(hash),

		// The file to be uploaded. io.ReadSeeker is preferred as the Uploader
		// will be able to optimize memory when uploading large content. io.Reader
		// is supported, but will require buffering of the reader's bytes for
		// each part.
		Body: blobsData,
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *S3StorageImpl) GetLatestSavedBlockHash() (string, error) {
	// Create a new AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-west-2"),                         // Replace with your desired region
		Credentials: credentials.NewSharedCredentials("", "default"), // Replace with your credentials profile
	})
	if err != nil {
		return "", err
	}

	// Create a new S3 service client
	svc := s3.New(sess)

	// Create the input parameters for the S3 ListObjectsV2 operation
	params := &s3.ListObjectsV2Input{Bucket: aws.String(s.bucket)}

	// List the objects in the S3 bucket
	resp, err := svc.ListObjectsV2(params)
	if err != nil {
		return "", err
	}

	// Sort the objects by LastModified in descending order
	sort.Slice(resp.Contents, func(i, j int) bool {
		return resp.Contents[i].LastModified.After(*resp.Contents[j].LastModified)
	})

	// Retrieve the key of the most recently saved object
	if len(resp.Contents) > 0 {
		return *resp.Contents[0].Key, nil
	}

	return "", errors.New("no objects found in bucket")
}
