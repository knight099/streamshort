package services

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"streamshort/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// AWSService handles S3 and CloudFront operations
type AWSService struct {
	s3Client              *s3.Client
	s3PresignClient       *s3.PresignClient
	bucket                string
	cloudFrontDomain      string
	cloudFrontKeyPairID   string
	cloudFrontPrivateKey  *rsa.PrivateKey
	region                string
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
func (s *AWSService) GenerateUploadPresignedURL(ctx context.Context, uploadID, filename, contentType string, expiresIn time.Duration) (string, error) {
	key := fmt.Sprintf("uploads/%s/%s", uploadID, filename)

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
	return fmt.Sprintf("s3://%s/uploads/%s/%s", s.bucket, uploadID, filename)
}

// GenerateCloudFrontSignedURL generates a signed URL for CloudFront streaming
func (s *AWSService) GenerateCloudFrontSignedURL(resourcePath string, expiresIn time.Duration) (string, time.Time, error) {
	if s.cloudFrontPrivateKey == nil || s.cloudFrontKeyPairID == "" || s.cloudFrontDomain == "" {
		return "", time.Time{}, fmt.Errorf("CloudFront signing not configured")
	}

	expiresAt := time.Now().Add(expiresIn)
	url := fmt.Sprintf("https://%s/%s", s.cloudFrontDomain, resourcePath)

	// Create CloudFront signer using AWS SDK v2
	signer := sign.NewURLSigner(s.cloudFrontKeyPairID, s.cloudFrontPrivateKey)

	// Sign the URL with expiration (canned policy)
	signedURL, err := signer.Sign(url, expiresAt)
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

