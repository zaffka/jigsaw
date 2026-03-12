package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// Config holds S3 client configuration.
type Config struct {
	Endpoint   string // Must include scheme (http:// or https://)
	AccessKey  string
	SecretKey  string
	BucketName string
	Region     string
}

// PutObjectOptions holds options for uploading an object to S3.
type PutObjectOptions struct {
	ContentType string
}

// UploadInfo contains information about an uploaded object.
type UploadInfo struct {
	Key  string
	ETag string
}

// GetObjectResult contains information about a downloaded object.
type GetObjectResult struct {
	Body        io.ReadCloser
	ContentType string
	Size        int64
}

// BucketCli wraps AWS SDK S3 client for specific bucket.
type BucketCli struct {
	client     *s3.Client
	presignCli *s3.PresignClient
	bucketName string
}

// NewBucketCli creates a new S3 client with bucket validation/creation.
// It initializes the S3 client internally using the provided configuration,
// checks if the bucket exists, and creates it if necessary.
func NewBucketCli(ctx context.Context, cfg Config) (*BucketCli, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	// Create S3 client configuration
	s3Config := aws.Config{
		Region: cfg.Region,
		Credentials: credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		),
	}

	// Create S3 client with custom endpoint resolver for SeaweedFS and other S3-compatible services
	client := s3.NewFromConfig(s3Config, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true // Required for SeaweedFS and some S3-compatible services
	})

	if err := ensureBucket(ctx, client, cfg.BucketName); err != nil {
		return nil, err
	}

	return &BucketCli{
		client:     client,
		presignCli: s3.NewPresignClient(client),
		bucketName: cfg.BucketName,
	}, nil
}

// PutObject uploads an object to S3.
func (c *BucketCli) PutObject(ctx context.Context, objectName string, reader io.Reader, objectSize int64, opts PutObjectOptions) (UploadInfo, error) {
	input := &s3.PutObjectInput{
		Bucket:        aws.String(c.bucketName),
		Key:           aws.String(objectName),
		Body:          reader,
		ContentLength: aws.Int64(objectSize),
	}

	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}

	result, err := c.client.PutObject(ctx, input)
	if err != nil {
		return UploadInfo{}, fmt.Errorf("failed to put object to bucket: %w", err)
	}

	etag := ""
	if result.ETag != nil {
		etag = *result.ETag
	}

	return UploadInfo{
		Key:  objectName,
		ETag: etag,
	}, nil
}

// GetObject downloads an object from S3.
func (c *BucketCli) GetObject(ctx context.Context, objectName string) (*GetObjectResult, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(objectName),
	}

	result, err := c.client.GetObject(ctx, input)
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, fmt.Errorf("object not found: %s", objectName)
		}
		return nil, fmt.Errorf("failed to get object from bucket: %w", err)
	}

	contentType := ""
	if result.ContentType != nil {
		contentType = *result.ContentType
	}

	size := int64(0)
	if result.ContentLength != nil {
		size = *result.ContentLength
	}

	return &GetObjectResult{
		Body:        result.Body,
		ContentType: contentType,
		Size:        size,
	}, nil
}

// DeleteObject deletes an object from S3.
func (c *BucketCli) DeleteObject(ctx context.Context, objectName string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(objectName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from bucket: %w", err)
	}

	return nil
}

// ObjectExists checks if an object exists in the bucket.
func (c *BucketCli) ObjectExists(ctx context.Context, objectName string) (bool, error) {
	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(objectName),
	})
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// GetPresignedURL generates a presigned URL for downloading an object.
// The URL is valid for the specified duration.
func (c *BucketCli) GetPresignedURL(ctx context.Context, objectName string, expiresIn time.Duration) (string, error) {
	presignResult, err := c.presignCli.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(objectName),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiresIn
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignResult.URL, nil
}

func validateConfig(cfg Config) error {
	if cfg.Endpoint == "" {
		return fmt.Errorf("S3 endpoint is required")
	}
	if cfg.AccessKey == "" {
		return fmt.Errorf("S3 access key is required")
	}
	if cfg.SecretKey == "" {
		return fmt.Errorf("S3 secret key is required")
	}
	if cfg.BucketName == "" {
		return fmt.Errorf("S3 bucket name is required")
	}
	if cfg.Region == "" {
		return fmt.Errorf("S3 region is required")
	}

	parsedURL, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid S3 endpoint URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("S3 endpoint must include scheme (http:// or https://), got: %s", cfg.Endpoint)
	}
	if parsedURL.Host == "" {
		return fmt.Errorf("S3 endpoint must include host, got: %s", cfg.Endpoint)
	}

	return nil
}

func ensureBucket(ctx context.Context, client *s3.Client, bucket string) error {
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err == nil {
		return nil
	}

	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return fmt.Errorf("bucket check failed: %w", err)
	}

	if apiErr.ErrorCode() != "NotFound" && apiErr.ErrorCode() != "NoSuchBucket" {
		return fmt.Errorf("bucket check failed: %w", err)
	}

	_, createErr := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if createErr != nil {
		return fmt.Errorf("failed to create bucket: %w", createErr)
	}

	return nil
}
