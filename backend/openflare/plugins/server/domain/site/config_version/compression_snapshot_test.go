package config_version

import (
	"context"
	"strings"
	"testing"

	"Wavelet/openflare/plugins/server/kernel/model"
	"Wavelet/openflare/plugins/server/kernel/repository"
)

func TestCompressionSnapshotDefaultsAndSavedSettings(t *testing.T) {
	t.Cleanup(setupOriginErrorPageSnapshotDB(t))
	ctx := context.Background()
	snapshot := buildOpenRestyConfigSnapshot(ctx)
	if snapshot.ProxyRequestBuffering {
		t.Errorf("buildOpenRestyConfigSnapshot(default).ProxyRequestBuffering = true, want false")
	}
	if !strings.Contains(snapshot.MainConfigTemplate, "gzip_vary on;") {
		t.Errorf("buildOpenRestyConfigSnapshot(default).MainConfigTemplate = %q, want gzip_vary on", snapshot.MainConfigTemplate)
	}
	custom := strings.ReplaceAll(model.DefaultOpenRestyMainConfigTemplate, "gzip_vary on;", "gzip_vary off;")
	if err := repository.SaveOrUpdateSystemConfig(ctx, model.ConfigKeyOpenRestyMainConfigTemplate, custom); err != nil {
		t.Fatal(err)
	}
	if err := repository.SaveOrUpdateSystemConfig(ctx, model.ConfigKeyOpenRestyProxyRequestBufferingEnabled, "true"); err != nil {
		t.Fatal(err)
	}
	snapshot = buildOpenRestyConfigSnapshot(ctx)
	if snapshot.MainConfigTemplate != custom || !snapshot.ProxyRequestBuffering {
		t.Errorf("buildOpenRestyConfigSnapshot(saved) = %+v, want saved template and request buffering enabled", snapshot)
	}
}
