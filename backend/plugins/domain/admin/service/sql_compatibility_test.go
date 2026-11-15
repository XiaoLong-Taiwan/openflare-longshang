package service_test

import (
	"context"
	"testing"

	"Wavelet/plugins/domain/admin/service"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSQLAPIServiceCompatibility(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	connection, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	service.SetDBService(&testDBService{db: db})
	t.Cleanup(service.ResetServices)
	ctx := context.Background()
	result, err := service.ExecuteCustomSQL(ctx, "SELECT 42 AS value")
	if err != nil || result.Type != "select" || result.AffectedRows != 1 || len(result.Columns) != 1 || result.Columns[0] != "value" {
		t.Fatalf("ExecuteCustomSQL(select) = %+v, %v, want one result column and row", result, err)
	}
	if _, err := service.ExecuteCustomSQL(ctx, "CREATE TABLE compatibility (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	result, err = service.ExecuteCustomSQL(ctx, "INSERT INTO compatibility VALUES (42)")
	if err != nil || result.Type != "exec" || result.AffectedRows != 1 {
		t.Errorf("ExecuteCustomSQL(insert) = %+v, %v, want exec and one affected row", result, err)
	}
}
