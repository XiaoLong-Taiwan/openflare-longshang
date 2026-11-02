package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"Wavelet/openflare/plugins/server/kernel/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCleanupRetainedConfigVersionsKeepsRecentAndNodeAppliedVersions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(:memory:) error = %v, want nil", err)
	}
	if err := db.AutoMigrate(&model.ConfigVersion{}, &model.OpenFlareApplyLog{}); err != nil {
		t.Fatalf("AutoMigrate(config versions, node apply logs) error = %v, want nil", err)
	}
	SetDBForTest(db)
	t.Cleanup(func() {
		SetDBForTest(nil)
		sqlDB, err := db.DB()
		if err != nil {
			t.Errorf("db.DB() error = %v, want nil", err)
			return
		}
		if err := sqlDB.Close(); err != nil {
			t.Errorf("sqlDB.Close() error = %v, want nil", err)
		}
	})

	const appliedVersion = "2026-01-01-00001"
	versions := make([]model.ConfigVersion, 8)
	for i := range versions {
		versions[i] = model.ConfigVersion{
			Version:        fmt.Sprintf("2026-01-01-%05d", i+1),
			SnapshotJSON:   "{}",
			RenderedConfig: "",
			Checksum:       fmt.Sprintf("checksum-%d", i+1),
			CreatedBy:      "test",
			CreatedAt:      time.Date(2026, 1, i+1, 0, 0, 0, 0, time.UTC),
		}
	}
	if err := db.Create(&versions).Error; err != nil {
		t.Fatalf("create config versions error = %v, want nil", err)
	}
	if err := db.Create(&model.OpenFlareApplyLog{NodeID: "node-1", Version: appliedVersion, Result: "success"}).Error; err != nil {
		t.Fatalf("create node config apply reference error = %v, want nil", err)
	}

	deleted, err := CleanupRetainedConfigVersions(context.Background())
	if err != nil {
		t.Fatalf("CleanupRetainedConfigVersions() error = %v, want nil", err)
	}
	if deleted != 2 {
		t.Fatalf("CleanupRetainedConfigVersions() deleted = %d, want 2", deleted)
	}
	var got []model.ConfigVersion
	if err := db.Order("version").Find(&got).Error; err != nil {
		t.Fatalf("query retained config versions error = %v, want nil", err)
	}
	if len(got) != 6 {
		t.Fatalf("retained config version count = %d, want 6", len(got))
	}
	foundApplied := false
	foundRecent := false
	foundRetainedBoundary := false
	for _, version := range got {
		if version.Version == appliedVersion {
			foundApplied = true
		}
		if version.Version == versions[len(versions)-1].Version {
			foundRecent = true
		}
		if version.Version == versions[3].Version {
			foundRetainedBoundary = true
		}
	}
	if !foundApplied {
		t.Errorf("CleanupRetainedConfigVersions() removed node-applied version %q", appliedVersion)
	}
	if !foundRecent {
		t.Errorf("CleanupRetainedConfigVersions() removed newest version %q", versions[len(versions)-1].Version)
	}
	if !foundRetainedBoundary {
		t.Errorf("CleanupRetainedConfigVersions() removed retained boundary version %q", versions[3].Version)
	}
	for _, version := range got {
		if version.Version == versions[2].Version {
			t.Errorf("CleanupRetainedConfigVersions() retained expired unreferenced version %q", version.Version)
		}
	}
}
