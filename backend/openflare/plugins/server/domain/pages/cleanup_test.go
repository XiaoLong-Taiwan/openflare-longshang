package pages

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"Wavelet/openflare/plugins/server/kernel/model"
	"Wavelet/openflare/plugins/server/kernel/ofupload"
	"Wavelet/openflare/plugins/server/kernel/repository"
)

func TestCleanupRetainedVersions(t *testing.T) {
	t.Cleanup(setupPagesTestDB(t))
	ctx := context.Background()
	db := repository.DB(ctx)
	for _, slug := range []string{"active", "inactive"} {
		project := model.PagesProject{Name: slug, Slug: slug}
		if err := db.Create(&project).Error; err != nil {
			t.Fatalf("create project %s: %v", slug, err)
		}
		deployments := make([]model.PagesDeployment, 7)
		for i := range deployments {
			uploadID := uint64(project.ID*100 + uint(i))
			if slug == "inactive" && i == 1 {
				uploadID = uint64(project.ID * 100)
			} else if err := db.Create(&model.Upload{
				ID: uploadID, UserID: 999, FileName: "site.zip", FilePath: fmt.Sprintf("pages/%d.zip", uploadID),
				Type: ofupload.ReservedPagesDeploymentType, Status: model.UploadStatusUsed, Size: 10,
			}).Error; err != nil {
				t.Fatalf("create upload %d: %v", uploadID, err)
			}
			deployments[i] = model.PagesDeployment{ProjectID: project.ID, DeploymentNumber: i + 1, UploadID: uploadID}
			if err := db.Create(&deployments[i]).Error; err != nil {
				t.Fatalf("create deployment %s/%d: %v", slug, i, err)
			}
			if err := db.Create(&model.PagesDeploymentFile{DeploymentID: deployments[i].ID, Path: "index.html"}).Error; err != nil {
				t.Fatalf("create deployment file: %v", err)
			}
		}
		activeID := deployments[0].ID
		if slug == "active" {
			if err := db.Model(&project).Update("active_deployment_id", activeID).Error; err != nil {
				t.Fatalf("set active deployment: %v", err)
			}
			deployments[1].UploadID = deployments[6].UploadID
			if err := db.Model(&deployments[1]).Update("upload_id", deployments[1].UploadID).Error; err != nil {
				t.Fatalf("share retained artifact: %v", err)
			}
		}
	}
	if err := db.Model(&model.PagesDeployment{}).Where("id = ?", 10).Update("upload_id", 102).Error; err != nil {
		t.Fatalf("share artifact across projects: %v", err)
	}
	for pass := range 2 {
		if err := CleanupRetainedVersions(ctx); err != nil {
			t.Fatalf("CleanupRetainedVersions() pass %d error = %v, want nil", pass, err)
		}
	}
	for projectID := uint(1); projectID <= 2; projectID++ {
		var ids []uint
		if err := db.Model(&model.PagesDeployment{}).Where("project_id = ?", projectID).Order("id").Pluck("id", &ids).Error; err != nil {
			t.Fatal(err)
		}
		want := []uint{1, 4, 5, 6, 7}
		if projectID == 2 {
			want = []uint{10, 11, 12, 13, 14}
		}
		if !reflect.DeepEqual(ids, want) {
			t.Errorf("CleanupRetainedVersions() project %d ids = %v, want %v", projectID, ids, want)
		}
	}
	var fileIDs []uint
	if err := db.Model(&model.PagesDeploymentFile{}).Order("deployment_id").Pluck("deployment_id", &fileIDs).Error; err != nil {
		t.Fatal(err)
	}
	if want := []uint{1, 4, 5, 6, 7, 10, 11, 12, 13, 14}; !reflect.DeepEqual(fileIDs, want) {
		t.Errorf("CleanupRetainedVersions() file deployment ids = %v, want %v", fileIDs, want)
	}
	for _, check := range []struct {
		id     uint64
		status model.UploadStatus
	}{{100, model.UploadStatusUsed}, {102, model.UploadStatusUsed}, {106, model.UploadStatusUsed}, {200, model.UploadStatusDeleted}} {
		var upload model.Upload
		if err := db.First(&upload, check.id).Error; err != nil {
			t.Fatal(err)
		}
		if upload.Status != check.status {
			t.Errorf("CleanupRetainedVersions() upload %d status = %s, want %s", check.id, upload.Status, check.status)
		}
	}
	if err := DeleteDeployment(ctx, 2, 10); err != nil {
		t.Fatalf("DeleteDeployment(2, 10) error = %v, want nil", err)
	}
	var upload model.Upload
	if err := db.First(&upload, 102).Error; err != nil {
		t.Fatal(err)
	}
	if upload.Status != model.UploadStatusDeleted {
		t.Errorf("DeleteDeployment(2, 10) upload 102 status = %s, want %s", upload.Status, model.UploadStatusDeleted)
	}
}
