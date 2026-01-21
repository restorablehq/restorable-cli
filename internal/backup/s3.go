package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"restorable.io/restorable-cli/internal/config"
)

// S3Source implements BackupSource for S3-compatible storage.
type S3Source struct {
	client   *s3.Client
	bucket   string
	prefix   string
	endpoint string
	// resolvedKey stores the actual key used after prefix resolution
	resolvedKey string
}

// NewS3Source creates a new S3Source from configuration.
func NewS3Source(cfg *config.S3) (*S3Source, error) {
	accessKey := os.Getenv(cfg.AccessKeyEnv)
	if accessKey == "" {
		return nil, fmt.Errorf("S3 access key environment variable %s is not set", cfg.AccessKeyEnv)
	}

	secretKey := os.Getenv(cfg.SecretKeyEnv)
	if secretKey == "" {
		return nil, fmt.Errorf("S3 secret key environment variable %s is not set", cfg.SecretKeyEnv)
	}

	// Build S3 client options
	opts := []func(*s3.Options){
		func(o *s3.Options) {
			o.Region = cfg.Region
			o.Credentials = credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
		},
	}

	// Custom endpoint for S3-compatible services (MinIO, etc.)
	if cfg.Endpoint != "" {
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true // Required for most S3-compatible services
		})
	}

	client := s3.New(s3.Options{}, opts...)

	return &S3Source{
		client:   client,
		bucket:   cfg.Bucket,
		prefix:   cfg.Prefix,
		endpoint: cfg.Endpoint,
	}, nil
}

// Acquire retrieves the backup from S3.
// If a prefix is configured, it lists objects and fetches the most recent one.
func (s *S3Source) Acquire(ctx context.Context) (io.ReadCloser, error) {
	key := s.prefix

	// If prefix ends with /, list and find the most recent object
	if len(s.prefix) > 0 && s.prefix[len(s.prefix)-1] == '/' {
		var err error
		key, err = s.findLatestObject(ctx)
		if err != nil {
			return nil, err
		}
	}

	s.resolvedKey = key

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object s3://%s/%s: %w", s.bucket, key, err)
	}

	return result.Body, nil
}

// findLatestObject lists objects under the prefix and returns the key of the most recently modified one.
func (s *S3Source) findLatestObject(ctx context.Context) (string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(s.prefix),
	}

	result, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to list objects in s3://%s/%s: %w", s.bucket, s.prefix, err)
	}

	if len(result.Contents) == 0 {
		return "", fmt.Errorf("no objects found in s3://%s/%s", s.bucket, s.prefix)
	}

	// Sort by LastModified descending
	sort.Slice(result.Contents, func(i, j int) bool {
		return result.Contents[i].LastModified.After(*result.Contents[j].LastModified)
	})

	return *result.Contents[0].Key, nil
}

// Identifier returns the S3 URI for traceability.
func (s *S3Source) Identifier() string {
	key := s.resolvedKey
	if key == "" {
		key = s.prefix
	}
	if s.endpoint != "" {
		return fmt.Sprintf("s3://%s/%s (endpoint: %s)", s.bucket, key, s.endpoint)
	}
	return fmt.Sprintf("s3://%s/%s", s.bucket, key)
}
