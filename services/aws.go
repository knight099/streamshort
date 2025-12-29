package services

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"streamshort/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// AWSService handles S3 and CloudFront operations
type AWSService struct {
	s3Client             *s3.Client
	s3PresignClient      *s3.PresignClient
	bucket               string
	cloudFrontDomain     string
	cloudFrontKeyPairID  string
	cloudFrontPrivateKey *rsa.PrivateKey
	region               string
}

// NewAWSService creates a new AWS service instance
func NewAWSService(cfg *config.Config) (*AWSService, error) {
	// Skip initialization if AWS credentials are not configured
	if cfg.AWSAccessKeyID == "" || cfg.AWSSecretAccessKey == "" || cfg.AWSS3Bucket == "" {
		return nil, nil // Return nil service, handler will use mock responses
	}

	// Create AWS config with credentials
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(cfg.AWSRegion),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AWSAccessKeyID,
			cfg.AWSSecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsCfg)
	presignClient := s3.NewPresignClient(s3Client)

	service := &AWSService{
		s3Client:            s3Client,
		s3PresignClient:     presignClient,
		bucket:              cfg.AWSS3Bucket,
		cloudFrontDomain:    cfg.AWSCloudFrontDomain,
		cloudFrontKeyPairID: cfg.AWSCloudFrontKeyPairID,
		region:              cfg.AWSRegion,
	}

	// Load CloudFront private key if configured
	if cfg.AWSCloudFrontKeyPairID != "" {
		var keyData []byte
		var err error

		if cfg.AWSCloudFrontPrivateKey != "" {
			keyData = []byte(cfg.AWSCloudFrontPrivateKey)
		} else if cfg.AWSCloudFrontPrivateKeyPath != "" {
			keyData, err = os.ReadFile(cfg.AWSCloudFrontPrivateKeyPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read private key file: %w", err)
			}
		}

		if len(keyData) > 0 {
			privateKey, err := parsePrivateKey(keyData)
			if err != nil {
				return nil, fmt.Errorf("failed to parse CloudFront private key: %w", err)
			}
			service.cloudFrontPrivateKey = privateKey
		}
	}

	return service, nil
}

// parsePrivateKey parses an RSA private key from PEM data
func parsePrivateKey(keyData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Try parsing as PKCS1 first, then PKCS8
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
	}

	return privateKey, nil
}

// GenerateUploadPresignedURL generates a pre-signed URL for uploading to S3
// fileType can be "video", "thumbnail", or "caption" to determine the S3 prefix
func (s *AWSService) GenerateUploadPresignedURL(ctx context.Context, uploadID, filename, contentType string, expiresIn time.Duration) (string, error) {
	return s.GenerateUploadPresignedURLWithType(ctx, uploadID, filename, contentType, "video", expiresIn)
}

// GenerateUploadPresignedURLWithType generates a pre-signed URL with specific file type
func (s *AWSService) GenerateUploadPresignedURLWithType(ctx context.Context, uploadID, filename, contentType, fileType string, expiresIn time.Duration) (string, error) {
	var prefix string
	switch fileType {
	case "thumbnail":
		prefix = "thumbnails"
	case "caption":
		prefix = "captions"
	default:
		prefix = "uploads"
	}

	// Extract file extension from original filename
	ext := ".mp4" // default extension
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		ext = filename[idx:]
	}

	// Use sanitized filename: uploadID + extension
	// This avoids issues with special characters (Hindi, emojis, etc.) in S3/CloudFront URLs
	sanitizedFilename := uploadID + ext
	key := fmt.Sprintf("%s/%s/%s", prefix, uploadID, sanitizedFilename)

	presignedReq, err := s.s3PresignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expiresIn))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedReq.URL, nil
}

// GetS3Path returns the S3 path for an upload
func (s *AWSService) GetS3Path(uploadID, filename string) string {
	return s.GetS3PathWithType(uploadID, filename, "video")
}

// GetS3PathWithType returns the S3 path for an upload with specific file type
func (s *AWSService) GetS3PathWithType(uploadID, filename, fileType string) string {
	var prefix string
	switch fileType {
	case "thumbnail":
		prefix = "thumbnails"
	case "caption":
		prefix = "captions"
	default:
		prefix = "uploads"
	}

	// Extract file extension from original filename
	ext := ".mp4" // default extension
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		ext = filename[idx:]
	}

	// Use sanitized filename: uploadID + extension
	sanitizedFilename := uploadID + ext
	return fmt.Sprintf("s3://%s/%s/%s/%s", s.bucket, prefix, uploadID, sanitizedFilename)
}

// GenerateCloudFrontSignedURL generates a signed URL for CloudFront streaming
func (s *AWSService) GenerateCloudFrontSignedURL(resourcePath string, expiresIn time.Duration) (string, time.Time, error) {
	if s.cloudFrontPrivateKey == nil || s.cloudFrontKeyPairID == "" || s.cloudFrontDomain == "" {
		return "", time.Time{}, fmt.Errorf("CloudFront signing not configured")
	}

	expiresAt := time.Now().Add(expiresIn)

	// URL encode each path segment to handle special characters (Hindi, emojis, etc.)
	pathParts := strings.Split(resourcePath, "/")
	for i, part := range pathParts {
		pathParts[i] = url.PathEscape(part)
	}
	encodedPath := strings.Join(pathParts, "/")

	resourceURL := fmt.Sprintf("https://%s/%s", s.cloudFrontDomain, encodedPath)

	// Create CloudFront signer using AWS SDK v2
	signer := sign.NewURLSigner(s.cloudFrontKeyPairID, s.cloudFrontPrivateKey)

	// Sign the URL with expiration (canned policy)
	signedURL, err := signer.Sign(resourceURL, expiresAt)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign CloudFront URL: %w", err)
	}

	return signedURL, expiresAt, nil
}

// GenerateManifestURL generates a signed URL for HLS manifest access
func (s *AWSService) GenerateManifestURL(episodeID string, expiresIn time.Duration) (string, time.Time, error) {
	// Path to the HLS manifest in the transcoded folder
	resourcePath := fmt.Sprintf("transcoded/%s/index.m3u8", episodeID)
	return s.GenerateCloudFrontSignedURL(resourcePath, expiresIn)
}

// IsConfigured returns true if AWS service is properly configured
func (s *AWSService) IsConfigured() bool {
	return s != nil && s.s3Client != nil && s.bucket != ""
}

// IsCloudFrontConfigured returns true if CloudFront signing is configured
func (s *AWSService) IsCloudFrontConfigured() bool {
	return s != nil && s.cloudFrontPrivateKey != nil && s.cloudFrontKeyPairID != "" && s.cloudFrontDomain != ""
}

// GetBucket returns the S3 bucket name
func (s *AWSService) GetBucket() string {
	if s == nil {
		return ""
	}
	return s.bucket
}

// GetCloudFrontDomain returns the CloudFront domain
func (s *AWSService) GetCloudFrontDomain() string {
	if s == nil {
		return ""
	}
	return s.cloudFrontDomain
}

// ExtractS3Key extracts the S3 key from an s3:// URL or returns the path as-is
func (s *AWSService) ExtractS3Key(s3Path string) string {
	if strings.HasPrefix(s3Path, "s3://") {
		// Remove s3:// scheme
		withoutScheme := strings.TrimPrefix(s3Path, "s3://")
		// Find the first slash to separate bucket from key
		slashIndex := strings.Index(withoutScheme, "/")
		if slashIndex != -1 && slashIndex+1 < len(withoutScheme) {
			return withoutScheme[slashIndex+1:]
		}
		return ""
	}
	// If it's already a key path, return as-is
	return s3Path
}

// DeleteS3Object deletes a single object from S3
func (s *AWSService) DeleteS3Object(ctx context.Context, s3Path string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("AWS service not configured")
	}

	key := s.ExtractS3Key(s3Path)
	if key == "" {
		return fmt.Errorf("invalid S3 path: %s", s3Path)
	}

	_, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete S3 object %s: %w", s3Path, err)
	}

	return nil
}

// DeleteS3ObjectsByPrefix deletes all objects with the given prefix from S3
func (s *AWSService) DeleteS3ObjectsByPrefix(ctx context.Context, prefix string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("AWS service not configured")
	}

	// Ensure prefix ends with / for folder-like deletion
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	// List all objects with the prefix
	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	}

	var objectKeys []types.ObjectIdentifier

	// Paginate through all objects
	paginator := s3.NewListObjectsV2Paginator(s.s3Client, listInput)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list S3 objects with prefix %s: %w", prefix, err)
		}

		for _, obj := range output.Contents {
			objectKeys = append(objectKeys, types.ObjectIdentifier{
				Key: obj.Key,
			})
		}
	}

	// If no objects found, return early
	if len(objectKeys) == 0 {
		return nil
	}

	// Delete objects in batches (S3 allows up to 1000 objects per delete request)
	const maxBatchSize = 1000
	for i := 0; i < len(objectKeys); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(objectKeys) {
			end = len(objectKeys)
		}

		batch := objectKeys[i:end]
		_, err := s.s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &types.Delete{
				Objects: batch,
				Quiet:   aws.Bool(true),
			},
		})

		if err != nil {
			return fmt.Errorf("failed to delete S3 objects with prefix %s: %w", prefix, err)
		}
	}

	return nil
}
