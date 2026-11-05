// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package updater provides capabilities to check for, download, and apply updates.
package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"Wavelet/openflare/share/ofutil"
)

const (
	maxChecksumAssetSize = 64 * 1024
	goosWindows          = "windows"
	updateTmpFilePerm    = 0o600
	updateBinaryFilePerm = 0o755
)

var replaceAndRestartFunc = replaceAndRestart

// Config defines the configuration for the update service.
type Config struct {
	LocalVersion string
	AssetPrefix  string
	LogLabel     string
}

// Service handles checking and applying application binary updates.
type Service struct {
	updateMu     sync.Mutex
	requestMu    sync.Mutex
	retryAfter   map[string]time.Time
	httpClient   *http.Client
	lastCheckKey string
	localVersion string
	assetPrefix  string
	logLabel     string
}

// New creates a new updater Service with the provided configuration.
func New(cfg Config) *Service {
	return &Service{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		localVersion: cfg.LocalVersion,
		assetPrefix:  cfg.AssetPrefix,
		logLabel:     cfg.LogLabel,
	}
}

// UpdateOptions specifies parameters for checking and applying updates.
type UpdateOptions struct {
	Channel string
	TagName string
	Force   bool
}

type githubRelease struct {
	TagName    string        `json:"tag_name"`
	Prerelease bool          `json:"prerelease"`
	Draft      bool          `json:"draft"`
	Assets     []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

// CheckAndUpdate checks for a newer release on GitHub and performs an update if available.
func (s *Service) CheckAndUpdate(ctx context.Context, repo string, options UpdateOptions) error {
	s.updateMu.Lock()
	defer s.updateMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	release, err := s.getRelease(ctx, repo, options)
	if err != nil {
		return fmt.Errorf("check latest release: %w", err)
	}
	if release == nil || release.TagName == "" {
		return nil
	}

	remoteVersion := normalizeVersion(release.TagName)
	localVersion := normalizeVersion(s.localVersion)
	checkKey := buildReleaseCheckKey(options, remoteVersion)

	if remoteVersion == localVersion {
		return nil
	}
	if !options.Force && checkKey != "" && checkKey == s.lastCheckKey {
		return nil
	}
	if !isNewer(localVersion, remoteVersion) {
		s.lastCheckKey = checkKey
		return nil
	}

	slog.Info(s.logLabel+" update available", "from", localVersion, "to", remoteVersion)
	assetName := s.assetNameForGOOSGOARCH(runtime.GOOS, runtime.GOARCH)

	downloadURL, expectedChecksum, err := s.resolveReleaseAsset(ctx, release, assetName)
	if err != nil {
		return err
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	if err = s.downloadAndRestart(ctx, downloadURL, expectedChecksum, execPath); err != nil {
		return fmt.Errorf("download and restart: %w", err)
	}
	s.lastCheckKey = checkKey
	return nil
}

func (s *Service) getRelease(ctx context.Context, repo string, options UpdateOptions) (*githubRelease, error) {
	repo = strings.TrimSpace(repo)
	if !validRepository(repo) {
		return nil, errors.New("invalid update repository")
	}
	tagName := strings.TrimSpace(options.TagName)
	var release *githubRelease
	var err error
	if tagName != "" {
		release, err = s.getReleaseByTag(ctx, repo, tagName)
	} else if strings.EqualFold(strings.TrimSpace(options.Channel), "preview") {
		return s.getLatestPreviewRelease(ctx, repo)
	} else {
		release, err = s.getLatestStableRelease(ctx, repo)
	}
	var failure *githubHTTPError
	if errors.As(err, &failure) && (failure.status == http.StatusForbidden || failure.status == http.StatusTooManyRequests) {
		return s.fallbackRelease(ctx, repo, tagName)
	}
	return release, err
}

func (s *Service) getLatestStableRelease(ctx context.Context, repo string) (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	return s.fetchReleaseFromURL(ctx, url)
}

func (s *Service) getLatestPreviewRelease(ctx context.Context, repo string) (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=20", repo)
	resp, err := s.request(ctx, url, true)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, &githubHTTPError{status: resp.StatusCode}
	}

	var releases []githubRelease
	if err = json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&releases); err != nil {
		return nil, errors.New("invalid github release list")
	}
	for _, release := range releases {
		if release.Draft || !release.Prerelease {
			continue
		}
		releaseCopy := release
		return &releaseCopy, nil
	}
	return nil, nil
}

func (s *Service) getReleaseByTag(ctx context.Context, repo string, tag string) (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, url.PathEscape(strings.TrimSpace(tag)))
	return s.fetchReleaseFromURL(ctx, url)
}

func (s *Service) fetchReleaseFromURL(ctx context.Context, url string) (*githubRelease, error) {
	resp, err := s.request(ctx, url, true)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &githubHTTPError{status: resp.StatusCode}
	}

	return decodeRelease(resp.Body)
}

func decodeRelease(reader io.Reader) (*githubRelease, error) {
	var release githubRelease
	if err := json.NewDecoder(io.LimitReader(reader, 4<<20)).Decode(&release); err != nil {
		return nil, errors.New("invalid github release metadata")
	}
	return &release, nil
}

func (s *Service) resolveReleaseAsset(ctx context.Context, release *githubRelease, assetName string) (downloadURL string, expectedChecksum string, err error) {
	checksumAssetName := assetName + ".sha256"
	var checksumURL string

	for _, asset := range release.Assets {
		switch asset.Name {
		case assetName:
			downloadURL = asset.BrowserDownloadURL
			expectedChecksum = normalizeGitHubDigest(asset.Digest)
		case checksumAssetName:
			checksumURL = asset.BrowserDownloadURL
		}
	}
	if downloadURL == "" {
		return "", "", fmt.Errorf("no matching asset %q in release %s", assetName, release.TagName)
	}
	if expectedChecksum != "" {
		return downloadURL, expectedChecksum, nil
	}
	if checksumURL == "" {
		return downloadURL, "", fmt.Errorf("no sha256 digest or checksum asset %q in release %s", checksumAssetName, release.TagName)
	}

	expectedChecksum, err = s.downloadChecksum(ctx, checksumURL, assetName)
	if err != nil {
		return downloadURL, "", fmt.Errorf("download checksum: %w", err)
	}
	return downloadURL, expectedChecksum, nil
}

func normalizeGitHubDigest(digest string) string {
	digest = strings.TrimSpace(digest)
	if digest == "" {
		return ""
	}
	const prefix = "sha256:"
	if strings.HasPrefix(strings.ToLower(digest), prefix) {
		digest = digest[len(prefix):]
	}
	digest = strings.ToLower(digest)
	if isSHA256Hex(digest) {
		return digest
	}
	return ""
}

func (s *Service) downloadChecksum(ctx context.Context, url string, assetName string) (string, error) {
	resp, err := s.request(ctx, url, false)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", &githubHTTPError{status: resp.StatusCode}
	}

	content, err := io.ReadAll(io.LimitReader(resp.Body, maxChecksumAssetSize+1))
	if err != nil {
		return "", errors.New("read checksum response failed")
	}
	if len(content) > maxChecksumAssetSize {
		return "", fmt.Errorf("checksum asset exceeds %d bytes", maxChecksumAssetSize)
	}
	checksum, err := parseSHA256Checksum(string(content), assetName)
	if err != nil {
		return "", err
	}
	return checksum, nil
}

func parseSHA256Checksum(content string, assetName string) (string, error) {
	assetName = strings.TrimSpace(assetName)
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if checksum, ok := parseSHA256Line(line, assetName); ok {
			return checksum, nil
		}
	}
	if assetName == "" {
		return "", errors.New("checksum asset does not contain a valid sha256 digest")
	}
	return "", fmt.Errorf("checksum asset does not contain a sha256 digest for %q", assetName)
}

func parseSHA256Line(line string, assetName string) (string, bool) {
	fields := strings.Fields(line)
	if len(fields) == 1 && isSHA256Hex(fields[0]) {
		return strings.ToLower(fields[0]), true
	}
	if len(fields) >= 2 && isSHA256Hex(fields[0]) {
		fileName := strings.TrimPrefix(strings.TrimSpace(fields[1]), "*")
		if assetName == "" || fileName == assetName {
			return strings.ToLower(fields[0]), true
		}
	}

	prefix := "SHA256("
	if strings.HasPrefix(line, prefix) {
		closing := strings.Index(line, ")")
		if closing > len(prefix) && closing+1 < len(line) {
			fileName := strings.TrimSpace(line[len(prefix):closing])
			rest := strings.TrimSpace(line[closing+1:])
			rest = strings.TrimPrefix(rest, "=")
			rest = strings.TrimSpace(rest)
			if isSHA256Hex(rest) && (assetName == "" || fileName == assetName) {
				return strings.ToLower(rest), true
			}
		}
	}
	return "", false
}

func isSHA256Hex(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func (s *Service) downloadAndRestart(ctx context.Context, url string, expectedChecksum string, targetPath string) error {
	expectedChecksum = strings.ToLower(strings.TrimSpace(expectedChecksum))
	if !isSHA256Hex(expectedChecksum) {
		return errors.New("invalid expected sha256 checksum")
	}
	resp, err := s.request(ctx, url, false)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return &githubHTTPError{status: resp.StatusCode}
	}

	tmpPath := targetPath + ".update"
	if runtime.GOOS == goosWindows && !strings.HasSuffix(strings.ToLower(tmpPath), ".exe") {
		tmpPath += ".exe"
	}
	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, updateTmpFilePerm) //nolint:gosec // tmpPath is derived from the configured updater binary location
	if err != nil {
		return err
	}
	hasher := sha256.New()
	if _, err = io.Copy(io.MultiWriter(tmpFile, hasher), resp.Body); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return errors.New("read binary response or write update file failed")
	}
	if err = tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	if actualChecksum != expectedChecksum {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("sha256 checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}
	if err = os.Chmod(tmpPath, updateBinaryFilePerm); err != nil && runtime.GOOS != goosWindows { //nolint:gosec // downloaded edge binary must remain executable
		_ = os.Remove(tmpPath)
		return fmt.Errorf("set executable permission: %w", err)
	}

	slog.Info(s.logLabel + " binary updated, restarting")
	return replaceAndRestartFunc(targetPath, tmpPath)
}

func (s *Service) assetNameForGOOSGOARCH(goos string, goarch string) string {
	name := fmt.Sprintf("%s-%s-%s", s.assetPrefix, goos, goarch)
	if goos == goosWindows {
		return name + ".exe"
	}
	return name
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	return v
}

func isNewer(local, remote string) bool {
	return compareVersions(local, remote) < 0
}

func buildReleaseCheckKey(options UpdateOptions, remoteVersion string) string {
	channel := strings.TrimSpace(options.Channel)
	if channel == "" {
		channel = "stable"
	}
	if tagName := strings.TrimSpace(options.TagName); tagName != "" {
		return channel + ":" + tagName
	}
	return channel + ":" + remoteVersion
}

func compareVersions(local string, remote string) int {
	return ofutil.CompareVersions(local, remote)
}
