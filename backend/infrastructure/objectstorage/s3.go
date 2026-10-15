// Package objectstorage implements application/ports.ObjectStorage
// against any S3-compatible endpoint via aws-sdk-go-v2 — Cloudflare R2
// in production, SeaweedFS in local dev, same code either way (see the
// deployment architecture section of the "Concepts & Architecture"
// document).
package objectstorage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/pauluno777/betailo/backend/application/ports"
)

// Config holds what's needed to talk to any S3-compatible endpoint. The
// same struct, only its values changing per environment, points at
// SeaweedFS in local dev and Cloudflare R2 in production — no branching
// in application code.
type Config struct {
	Endpoint  string // e.g. "http://seaweedfs:9000" (dev) or the R2 account endpoint (prod)
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string        // R2 ignores the value but the SDK requires one; SeaweedFS does too
	URLExpiry time.Duration // how long a presigned upload URL stays valid; defaults to 15 minutes

	// PublicEndpoint is what a presigned URL is actually addressed to —
	// the browser that receives one from an API response resolves it
	// directly, so it must be reachable from outside the Docker network
	// Endpoint lives on (e.g. "http://localhost:9000" against the
	// "seaweedfs" service, which docker-compose already publishes on
	// that host port). Defaults to Endpoint, which is correct whenever
	// the two coincide (single-endpoint R2 in production).
	PublicEndpoint string
}

// S3Provider implements ports.ObjectStorage against any S3-compatible
// object store.
type S3Provider struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
	expiry  time.Duration
}

var _ ports.ObjectStorage = (*S3Provider)(nil)

// NewS3Provider builds a Provider from Config. It performs no network
// call itself — the SDK client only touches the network when a URL is
// actually presigned — so a misconfigured bucket/credentials pair only
// surfaces on first use, same as the rest of this package's callers.
func NewS3Provider(ctx context.Context, cfg Config) (*S3Provider, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("objectstorage: Bucket must not be empty")
	}
	if cfg.URLExpiry <= 0 {
		cfg.URLExpiry = 15 * time.Minute
	}

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("objectstorage: load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		// SeaweedFS and R2 both expect path-style addressing rather than
		// AWS's virtual-hosted-style buckets.
		o.UsePathStyle = true
	})

	publicEndpoint := cfg.PublicEndpoint
	if publicEndpoint == "" {
		publicEndpoint = cfg.Endpoint
	}
	presignClient := client
	if publicEndpoint != cfg.Endpoint {
		presignClient = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(publicEndpoint)
			o.UsePathStyle = true
		})
	}

	return &S3Provider{
		client:  client,
		presign: s3.NewPresignClient(presignClient),
		bucket:  cfg.Bucket,
		expiry:  cfg.URLExpiry,
	}, nil
}

// PresignUpload implements ports.ObjectStorage.
// PresignUpload signs Content-Length and Content-Type into the URL, so
// storage refuses a body that isn't exactly size bytes of JPEG.
func (p *S3Provider) PresignUpload(ctx context.Context, key string, size int64) (string, error) {
	req, err := p.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(p.bucket),
		Key:           aws.String(key),
		ContentLength: aws.Int64(size),
		ContentType:   aws.String("image/jpeg"),
	}, s3.WithPresignExpires(p.expiry))
	if err != nil {
		return "", fmt.Errorf("objectstorage: presign upload: %w", err)
	}
	return req.URL, nil
}

// Download URLs are signed as of the start of the current
// downloadWindow and stay valid for two of them: every request within a
// window gets the same URL for a key, still good for at least one more
// window, so the browser's cache (told the object never changes, which
// holds since every key is minted once) serves it instead of storage.
const downloadWindow = 6 * time.Hour

// PresignDownload implements ports.ObjectStorage. It returns the same
// URL for a key throughout a downloadWindow; see downloadWindow.
func (p *S3Provider) PresignDownload(ctx context.Context, key string) (string, error) {
	req, err := p.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:               aws.String(p.bucket),
		Key:                  aws.String(key),
		ResponseCacheControl: aws.String(fmt.Sprintf("private, max-age=%d, immutable", int((2 * downloadWindow).Seconds()))),
	}, s3.WithPresignExpires(2*downloadWindow), func(o *s3.PresignOptions) {
		o.Presigner = windowedPresigner{v4.NewSigner()}
	})
	if err != nil {
		return "", fmt.Errorf("objectstorage: presign download: %w", err)
	}
	return req.URL, nil
}

// EnsureBucket creates the configured bucket if it doesn't already
// exist. Idempotent — safe to call on every startup, mirroring
// mongodb.Bootstrap's approach for MongoDB collections (see
// internal/app). Needed for local dev (SeaweedFS/MinIO start with no
// buckets); a production R2 bucket is provisioned out of band and this
// becomes a no-op against it.
func (p *S3Provider) EnsureBucket(ctx context.Context) error {
	_, err := p.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(p.bucket)})
	if err == nil {
		return nil
	}

	var alreadyOwned *types.BucketAlreadyOwnedByYou
	var alreadyExists *types.BucketAlreadyExists
	if errors.As(err, &alreadyOwned) || errors.As(err, &alreadyExists) {
		return nil
	}

	// R2 denies CreateBucket outright (AccessDenied, not "already
	// owned") for any token scoped below Admin — including the
	// least-privilege Object Read & Write token production should use,
	// against a bucket that token already reads and writes fine. A
	// production bucket is provisioned out of band (package doc), so
	// this is exactly the no-op that call is supposed to be; only a
	// different failure here is worth surfacing.
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && apiErr.ErrorCode() == "AccessDenied" {
		return nil
	}
	return fmt.Errorf("objectstorage: create bucket: %w", err)
}

// windowedPresigner signs as of the start of the current downloadWindow
// instead of now, which is what keeps a download URL stable.
type windowedPresigner struct{ signer *v4.Signer }

func (w windowedPresigner) PresignHTTP(ctx context.Context, creds aws.Credentials, r *http.Request, payloadHash, service, region string, signingTime time.Time, optFns ...func(*v4.SignerOptions)) (string, http.Header, error) {
	return w.signer.PresignHTTP(ctx, creds, r, payloadHash, service, region, signingTime.UTC().Truncate(downloadWindow), optFns...)
}
