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
	PresignUpload(ctx context.Context, key string) (string, error)

	// PresignDownload returns a URL the client can GET the object from
	// directly (used to serve listing photos without proxying them).
	PresignDownload(ctx context.Context, key string) (string, error)
}
