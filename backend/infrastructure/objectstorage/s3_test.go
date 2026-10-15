package objectstorage

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

func newTestProvider(t *testing.T) *S3Provider {
	t.Helper()
	p, err := NewS3Provider(context.Background(), Config{
		Endpoint: "https://account.r2.cloudflarestorage.com", Bucket: "betailo",
		AccessKey: "key", SecretKey: "secret", Region: "auto",
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPresignUpload_LocksSizeAndType(t *testing.T) {
	raw, err := newTestProvider(t).PresignUpload(context.Background(), "listings/a/b.jpg", 1234)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(raw)
	if got := u.Query().Get("X-Amz-SignedHeaders"); got != "content-length;content-type;host" {
		t.Errorf("signed headers = %q, want size and type locked", got)
	}
}

func TestPresignDownload_StableAndCacheable(t *testing.T) {
	p := newTestProvider(t)
	first, err := p.PresignDownload(context.Background(), "listings/a/b.jpg")
	if err != nil {
		t.Fatal(err)
	}
	second, _ := p.PresignDownload(context.Background(), "listings/a/b.jpg")
	if first != second {
		t.Errorf("two presigns within a window differ:\n%s\n%s", first, second)
	}
	if !strings.Contains(first, "response-cache-control=private") {
		t.Errorf("URL %s doesn't ask storage for a cache header", first)
	}
}
