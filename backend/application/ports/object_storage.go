package ports

import "context"

// ObjectStorage abstracts file storage (listing photos, for now) behind
// a small interface so usecases never talk to a specific object-storage
// SDK directly. The concrete implementation — Cloudflare R2 in
// production, SeaweedFS in local dev — is injected at startup via
// configuration only, per the deployment architecture section of the
// "Concepts & Architecture" document.
type ObjectStorage interface {
	// PresignUpload returns a URL the client can PUT the object to
	// directly, bypassing the API server for the file bytes themselves.
	// The URL only accepts a JPEG of exactly size bytes, so storage
	// refuses anything heavier than what the usecase allowed.
	PresignUpload(ctx context.Context, key string, size int64) (string, error)

	// PresignDownload returns a URL the client can GET the object from
	// directly (used to serve listing photos without proxying them).
	// The same key yields the same URL for hours, so browsers cache the
	// image instead of downloading it again on every page.
	PresignDownload(ctx context.Context, key string) (string, error)
}
