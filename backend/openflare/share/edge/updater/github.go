package updater

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9._-]+$`)

type githubHTTPError struct {
	status  int
	retryAt time.Time
}

func (e *githubHTTPError) Error() string {
	if !e.retryAt.IsZero() {
		return fmt.Sprintf("github request status=%d; retry after %s", e.status, e.retryAt.UTC().Format(time.RFC3339))
	}
	return fmt.Sprintf("github request status=%d", e.status)
}

func validRepository(repo string) bool {
	return repositoryPattern.MatchString(repo) && !strings.HasSuffix(repo, "/.") && !strings.HasSuffix(repo, "/..")
}

func (s *Service) request(ctx context.Context, target string, metadata bool) (*http.Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil || req.URL.Scheme != "https" || req.URL.User != nil {
		return nil, errors.New("invalid github request URL")
	}
	host := req.URL.Host
	s.requestMu.Lock()
	deadline := s.retryAfter[host]
	s.requestMu.Unlock()
	if time.Now().Before(deadline) {
		return nil, &githubHTTPError{status: http.StatusTooManyRequests, retryAt: deadline}
	}
	req.Header.Set("User-Agent", "OpenFlare-Updater/1.0")
	if metadata {
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	} else {
		req.Header.Set("Accept", "application/octet-stream")
	}
	client := *s.httpClient
	if !metadata {
		client.Timeout = 10 * time.Minute
	}
	previousRedirect := client.CheckRedirect
	client.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if len(via) >= 10 || next.URL.Scheme != "https" || next.URL.User != nil {
			return errors.New("invalid github redirect")
		}
		for _, header := range []string{"Authorization", "Cookie", "Proxy-Authorization", "Referer"} {
			next.Header.Del(header)
		}
		if previousRedirect != nil {
			return previousRedirect(next, via)
		}
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("github request failed")
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		deadline = retryDeadline(resp.Header, time.Now())
		s.requestMu.Lock()
		if s.retryAfter == nil {
			s.retryAfter = make(map[string]time.Time)
		}
		s.retryAfter[host] = deadline
		s.requestMu.Unlock()
		_ = resp.Body.Close()
		return nil, &githubHTTPError{status: resp.StatusCode, retryAt: deadline}
	}
	return resp, nil
}

func retryDeadline(header http.Header, now time.Time) time.Time {
	deadline := now.Add(time.Minute)
	if seconds, err := strconv.ParseInt(header.Get("Retry-After"), 10, 64); err == nil && seconds >= 0 && seconds <= 86400 {
		deadline = now.Add(time.Duration(seconds) * time.Second)
	} else if date, err := http.ParseTime(header.Get("Retry-After")); err == nil && date.After(now) {
		deadline = date
	}
	if reset, err := strconv.ParseInt(header.Get("X-RateLimit-Reset"), 10, 64); err == nil && header.Get("X-RateLimit-Remaining") == "0" {
		if resetTime := time.Unix(reset, 0); resetTime.After(deadline) {
			deadline = resetTime
		}
	}
	if deadline.Before(now.Add(time.Minute)) {
		deadline = now.Add(time.Minute)
	}
	if deadline.After(now.Add(24 * time.Hour)) {
		deadline = now.Add(24 * time.Hour)
	}
	return deadline
}

func (s *Service) fallbackRelease(ctx context.Context, repo, tag string) (*githubRelease, error) {
	page := "https://github.com/" + repo + "/releases/latest"
	if tag != "" {
		page = "https://github.com/" + repo + "/releases/tag/" + url.PathEscape(tag)
	}
	resp, err := s.request(ctx, page, false)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, &githubHTTPError{status: resp.StatusCode}
	}
	if resp.Request == nil || resp.Request.URL == nil {
		return nil, errors.New("github release fallback has no final URL")
	}
	final := resp.Request.URL
	prefix := "/" + repo + "/releases/tag/"
	if final.Scheme != "https" || final.Host != "github.com" || final.User != nil || !strings.HasPrefix(strings.ToLower(final.EscapedPath()), strings.ToLower(prefix)) {
		return nil, errors.New("invalid github release fallback redirect")
	}
	resolvedTag, err := url.PathUnescape(final.EscapedPath()[len(prefix):])
	if err != nil || resolvedTag == "" || resolvedTag == "." || resolvedTag == ".." || (tag != "" && resolvedTag != tag) {
		return nil, errors.New("github release fallback tag mismatch")
	}
	asset := s.assetNameForGOOSGOARCH(runtime.GOOS, runtime.GOARCH)
	base := "https://github.com/" + repo + "/releases/download/" + url.PathEscape(resolvedTag) + "/"
	return &githubRelease{TagName: resolvedTag, Assets: []githubAsset{
		{Name: asset, BrowserDownloadURL: base + url.PathEscape(asset)},
		{Name: asset + ".sha256", BrowserDownloadURL: base + url.PathEscape(asset+".sha256")},
	}}, nil
}
