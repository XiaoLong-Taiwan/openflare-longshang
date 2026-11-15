package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"Wavelet/plugins/domain/admin/errs"
	"Wavelet/plugins/domain/admin/model"
	"Wavelet/plugins/domain/admin/service"
)

func TestMaintenanceSelectionCompatibility(t *testing.T) {
	handler := &service.DatabaseMaintenanceHandler{}
	payload, err := handler.ValidatePayload([]byte(`{"confirmed":true}`))
	if err != nil {
		t.Fatal(err)
	}
	var request model.DatabaseMaintenanceRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatal(err)
	}
	if request.CleanupTaskExecutions == nil || !*request.CleanupTaskExecutions || request.ClearDiskCache == nil || !*request.ClearDiskCache || request.OptimizeDatabase == nil || !*request.OptimizeDatabase {
		t.Fatalf("ValidatePayload(legacy) = %+v, want all scopes selected", request)
	}
	for _, invalid := range []string{`{}`, `null`, `{"confirmed":false,"clear_disk_cache":true}`, `{"confirmed":true,"clear_disk_cache":false}`} {
		if _, err := handler.ValidatePayload([]byte(invalid)); err == nil {
			t.Errorf("ValidatePayload(%s) error = nil, want rejection", invalid)
		}
	}
	payload, err = handler.ValidatePayload([]byte(`{"confirmed":true,"cleanup_task_executions":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatal(err)
	}
	var selected map[string]bool
	if err := json.Unmarshal(payload, &selected); err != nil {
		t.Fatal(err)
	}
	if selected["clear_disk_cache"] || selected["optimize_database"] {
		t.Errorf("ValidatePayload(single scope) = %s, want no other scopes", payload)
	}
	service.ResetServices()
	t.Cleanup(service.ResetServices)
	if _, err := handler.Execute(context.Background(), []byte(`{}`)); !errors.Is(err, errs.ErrDatabaseUninitialized) {
		t.Errorf("Execute(queued legacy) error = %v, want ErrDatabaseUninitialized after validation", err)
	}
}
