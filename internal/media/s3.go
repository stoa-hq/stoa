package media

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// S3Config holds S3 storage configuration.
type S3Config struct {
	Bucket          string
	Region          string
	Endpoint        string // optional: custom endpoint for MinIO, Cloudflare R2, etc.
	AccessKeyID     string
	SecretAccessKey string
}

// S3Storage stores files in S3-compatible object storage.
type S3Storage struct {
	client *s3.Client
	config S3Config
}

// NewS3Storage creates a new S3Storage. Static credentials are used when
// AccessKeyID and SecretAccessKey are non-empty; otherwise the default AWS
// credential chain (env vars, ~/.aws/credentials, instance metadata) applies.
func NewS3Storage(ctx context.Context, cfg S3Config) (*S3Storage, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	clientOpts := []func(*s3.Options){}
	if cfg.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true // required for MinIO and most S3-compatible stores
		})
	}

	client := s3.NewFromConfig(awsCfg, clientOpts...)
	return &S3Storage{client: client, config: cfg}, nil
}

func (s *S3Storage) Store(ctx context.Context, filename string, reader io.Reader, size int64) (*StoredFile, error) {
	now := time.Now().UTC()
	key := fmt.Sprintf("%d/%02d/%s-%s", now.Year(), now.Month(), uuid.New().String()[:8], filename)

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
		Body:   reader,
	}
	if size > 0 {
		input.ContentLength = aws.Int64(size)
	}

	if _, err := s.client.PutObject(ctx, input); err != nil {
		return nil, fmt.Errorf("s3: uploading %q: %w", key, err)
	}

	return &StoredFile{
		Path: key,
		URL:  s.URL(key),
		Size: size,
	}, nil
}

func (s *S3Storage) Delete(ctx context.Context, path string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return fmt.Errorf("s3: deleting %q: %w", path, err)
	}
	return nil
}

func (s *S3Storage) URL(path string) string {
	if s.config.Endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", s.config.Endpoint, s.config.Bucket, path)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.config.Bucket, s.config.Region, path)
}
