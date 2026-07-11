package mediawiki

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoRequestRetriesNonJSONResponse(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("error code: 502"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query":{"pages":{"1":{"revisions":[{"slots":{"main":{"*":"ok"}}}]}}}}`))
	}))
	t.Cleanup(server.Close)

	backoffs := rateLimitBackoffs
	rateLimitBackoffs = []time.Duration{time.Millisecond}
	t.Cleanup(func() { rateLimitBackoffs = backoffs })

	requestDelay = 0
	t.Cleanup(func() { requestDelay = 100 * time.Millisecond })

	client := &MediaWikiClient{
		apiURL:     server.URL,
		httpClient: server.Client(),
		userAgent:  "test",
		tokens:     make(map[string]string),
	}

	content, err := client.getPageContent("Template:World/wrld_test/name")
	if err != nil {
		t.Fatalf("getPageContent: %v", err)
	}
	if content != "ok" {
		t.Fatalf("content = %q, want ok", content)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
}

func TestDoRequestRetriesServerError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("bad gateway"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query":{"pages":{"1":{"revisions":[{"slots":{"main":{"*":"ok"}}}]}}}}`))
	}))
	t.Cleanup(server.Close)

	backoffs := rateLimitBackoffs
	rateLimitBackoffs = []time.Duration{time.Millisecond}
	t.Cleanup(func() { rateLimitBackoffs = backoffs })

	requestDelay = 0
	t.Cleanup(func() { requestDelay = 100 * time.Millisecond })

	client := &MediaWikiClient{
		apiURL:     server.URL,
		httpClient: server.Client(),
		userAgent:  "test",
		tokens:     make(map[string]string),
	}

	content, err := client.getPageContent("Template:World/wrld_test/name")
	if err != nil {
		t.Fatalf("getPageContent: %v", err)
	}
	if content != "ok" {
		t.Fatalf("content = %q, want ok", content)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
}

func TestResponseBodyPreview(t *testing.T) {
	t.Parallel()

	long := make([]byte, responseBodyPreviewLen+50)
	for i := range long {
		long[i] = 'a'
	}
	got := responseBodyPreview(long)
	if len(got) != responseBodyPreviewLen+3 {
		t.Fatalf("preview len = %d, want %d", len(got), responseBodyPreviewLen+3)
	}
	if got[len(got)-3:] != "..." {
		t.Fatalf("preview suffix = %q, want ...", got[len(got)-3:])
	}
}
