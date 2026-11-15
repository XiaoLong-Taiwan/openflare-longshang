package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"Wavelet/core/contracts"
	"Wavelet/plugins/domain/auth/model/entity"
	"golang.org/x/oauth2"
)

type providerTransport func(*http.Request) (*http.Response, error)

func (f providerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestProviderIdentityPrecisionAndMissingID(t *testing.T) {
	for _, body := range []string{`{"id":123456789012345678,"login":"test"}`, `{"login":"test"}`, `{"id":null}`} {
		client := &http.Client{Transport: providerTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
		})}
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
		var info contracts.OAuthUserInfoDTO
		err := (&OAuthService{}).fetchProviderUserInfo(ctx, &entity.AuthSource{Type: "github"}, &oauth2.Token{AccessToken: "test"}, &info)
		if strings.Contains(body, "123456789012345678") {
			if err != nil || info.Sub != "123456789012345678" {
				t.Errorf("fetchProviderUserInfo(%s) = %q, %v, want exact ID", body, info.Sub, err)
			}
		} else if err == nil {
			t.Errorf("fetchProviderUserInfo(%s) error = nil, want missing ID rejection", body)
		}
	}
}
