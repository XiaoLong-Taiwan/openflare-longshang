package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestReleaseRateLimitFallsBackAndDefersAPI(t *testing.T) {
	apiCalls := 0
	webCalls := 0
	s := testService(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("User-Agent") == "" {
			t.Error("release request User-Agent is empty, want application identity")
		}
		if req.URL.Host == "api.github.com" {
			apiCalls++
			return &http.Response{StatusCode: 403, Header: http.Header{"Retry-After": {"120"}}, Body: io.NopCloser(strings.NewReader("secret-response"))}, nil
		}
		webCalls++
		finalURL, _ := url.Parse("https://github.com/owner/repo/releases/tag/v1.2.3")
		return &http.Response{StatusCode: 200, Header: make(http.Header), Request: &http.Request{URL: finalURL}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})})
	for range 2 {
		release, err := s.getRelease(context.Background(), "owner/repo", UpdateOptions{})
		if err != nil || release == nil || release.TagName != "v1.2.3" {
			t.Fatalf("getRelease(rate limited) = %#v, %v, want v1.2.3 fallback", release, err)
		}
		asset := s.assetNameForGOOSGOARCH(runtime.GOOS, runtime.GOARCH)
		if len(release.Assets) != 2 || release.Assets[0].BrowserDownloadURL != "https://github.com/owner/repo/releases/download/v1.2.3/"+asset {
			t.Fatalf("fallback assets = %#v, want binary and checksum for selected tag", release.Assets)
		}
	}
	if apiCalls != 1 || webCalls != 2 {
		t.Errorf("release calls = API %d, web %d, want 1, 2", apiCalls, webCalls)
	}
}

func TestPreviewRateLimitNeverFallsBackToStable(t *testing.T) {
	calls := 0
	s := testService(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Host != "api.github.com" {
			t.Errorf("preview host = %q, want api.github.com", req.URL.Host)
		}
		return &http.Response{StatusCode: 429, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
	})})
	for range 2 {
		if release, err := s.getRelease(context.Background(), "owner/repo", UpdateOptions{Channel: "preview", Force: true}); err == nil || release != nil {
			t.Fatalf("getRelease(preview) = %#v, %v, want rate-limit error", release, err)
		}
	}
	if calls != 1 {
		t.Errorf("preview API calls = %d, want 1 despite Force", calls)
	}
}

func TestChecksumErrorDoesNotExposeSignedURL(t *testing.T) {
	s := testService(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("transport secret-token")
	})})
	_, err := s.downloadChecksum(context.Background(), "https://example.test/file?token=secret-token", "binary")
	if err == nil || strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("downloadChecksum error safe = %t, want non-nil sanitized error", err != nil && !strings.Contains(err.Error(), "secret-token"))
	}
}

func TestRetryDeadline(t *testing.T) {
	now := time.Unix(1800000000, 0)
	cases := []struct {
		name   string
		header http.Header
		want   time.Duration
	}{
		{name: "default", header: make(http.Header), want: time.Minute},
		{name: "seconds", header: http.Header{"Retry-After": {"120"}}, want: 2 * time.Minute},
		{name: "date", header: http.Header{"Retry-After": {now.Add(3 * time.Minute).UTC().Format(http.TimeFormat)}}, want: 3 * time.Minute},
		{name: "reset", header: http.Header{"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {"1800000300"}}, want: 5 * time.Minute},
		{name: "overflow", header: http.Header{"Retry-After": {"9999999999999999999999"}}, want: time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := retryDeadline(tc.header, now).Sub(now); got != tc.want {
				t.Errorf("retryDeadline(%v) = %v, want %v", tc.header, got, tc.want)
			}
		})
	}
}

func TestFallbackRejectsForeignRepository(t *testing.T) {
	s := testService(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		finalURL, _ := url.Parse("https://github.com/other/repo/releases/tag/v1.2.3")
		return &http.Response{StatusCode: 200, Header: make(http.Header), Request: &http.Request{URL: finalURL}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})})
	if release, err := s.fallbackRelease(context.Background(), "owner/repo", ""); err == nil || release != nil {
		t.Fatalf("fallbackRelease(foreign repository) = %#v, %v, want rejection", release, err)
	}
}

func TestReleaseNotFoundDoesNotFallback(t *testing.T) {
	calls := 0
	s := testService(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 404, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
	})})
	release, err := s.getRelease(context.Background(), "owner/repo", UpdateOptions{TagName: "missing"})
	if release != nil || err != nil || calls != 1 {
		t.Errorf("getRelease(missing) = %#v, %v, calls %d, want nil, nil, 1", release, err, calls)
	}
}

func TestFallbackDownloadsVerifiedBinary(t *testing.T) {
	payload := "verified-update"
	sum := sha256.Sum256([]byte(payload))
	checksum := hex.EncodeToString(sum[:])
	s := testService(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := payload
		if strings.Contains(req.URL.Path, "/releases/tag/") {
			body = ""
		} else if strings.HasSuffix(req.URL.Path, ".sha256") {
			body = checksum
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Request: req, Body: io.NopCloser(strings.NewReader(body))}, nil
	})})
	release, err := s.fallbackRelease(context.Background(), "owner/repo", "release/v1.2.3")
	if err != nil {
		t.Fatalf("fallbackRelease(exact tag) = %v, want success", err)
	}
	asset := s.assetNameForGOOSGOARCH(runtime.GOOS, runtime.GOARCH)
	downloadURL, digest, err := s.resolveReleaseAsset(context.Background(), release, asset)
	if err != nil || digest != checksum {
		t.Fatalf("resolveReleaseAsset(fallback) digest = %q, error %v, want %q", digest, err, checksum)
	}
	original := replaceAndRestartFunc
	replaced := false
	replaceAndRestartFunc = func(_, tmpPath string) error {
		data, readErr := os.ReadFile(tmpPath)
		if readErr != nil || string(data) != payload {
			t.Errorf("downloaded payload = %q, error %v, want %q", data, readErr, payload)
		}
		replaced = true
		return nil
	}
	t.Cleanup(func() { replaceAndRestartFunc = original })
	if err = s.downloadAndRestart(context.Background(), downloadURL, digest, filepath.Join(t.TempDir(), "binary")); err != nil || !replaced {
		t.Errorf("downloadAndRestart(fallback) = %v, replaced %t, want nil, true", err, replaced)
	}
}

func TestDownloadRateLimitDefersNextRequest(t *testing.T) {
	calls := 0
	s := testService(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 403, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
	})})
	for range 2 {
		if _, err := s.downloadChecksum(context.Background(), "https://github.com/owner/repo/releases/download/v1/binary.sha256", "binary"); err == nil {
			t.Fatal("downloadChecksum(rate limit) = nil error, want backoff error")
		}
	}
	if calls != 1 {
		t.Errorf("downloadChecksum network calls = %d, want 1", calls)
	}
}
