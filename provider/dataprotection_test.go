package adaptive

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func sampleMasks() []DataProtectionMask {
	return []DataProtectionMask{{
		DatabaseName: "shop",
		Tables: []DataProtectionTable{{
			TableName: "users",
			Schema:    "public",
			MaskedColumns: []DataProtectionColumnMask{
				{ColumnName: "email", MaskingType: "email"},
				{ColumnName: "ssn", MaskingType: "ssn"},
			},
		}},
	}}
}

func TestApplyDataProtectionReadScopedDefault(t *testing.T) {
	tr := true
	// Unset scoped reading back the server default (true) is not drift.
	got := applyDataProtectionRead(DataProtectionArgs{Resource: "db", Masks: sampleMasks()},
		&DataProtectionReadResponse{ResourceName: "db", Scoped: &tr, Masks: sampleMasks()}, false)
	if got.Scoped != nil {
		t.Errorf("scoped adopted server default: %v", *got.Scoped)
	}
	// scoped=false must always surface.
	f := false
	got = applyDataProtectionRead(DataProtectionArgs{Resource: "db", Masks: sampleMasks()},
		&DataProtectionReadResponse{ResourceName: "db", Scoped: &f, Masks: sampleMasks()}, false)
	if got.Scoped == nil || *got.Scoped {
		t.Errorf("scoped=false drift missed: %v", got.Scoped)
	}
}

func TestApplyDataProtectionReadMaskEquality(t *testing.T) {
	prior := DataProtectionArgs{Resource: "db", Masks: sampleMasks()}

	// Same rules, different column ordering: user's spelling kept.
	server := sampleMasks()
	server[0].Tables[0].MaskedColumns = []DataProtectionColumnMask{
		{ColumnName: "ssn", MaskingType: "ssn"},
		{ColumnName: "email", MaskingType: "email"},
	}
	got := applyDataProtectionRead(prior, &DataProtectionReadResponse{ResourceName: "db", Masks: server}, false)
	if !reflect.DeepEqual(got.Masks, prior.Masks) {
		t.Errorf("semantically equal masks not preserved: %+v", got.Masks)
	}

	// Real drift: masking type changed out-of-band.
	server = sampleMasks()
	server[0].Tables[0].MaskedColumns[0].MaskingType = "redact"
	got = applyDataProtectionRead(prior, &DataProtectionReadResponse{ResourceName: "db", Masks: server}, false)
	if !reflect.DeepEqual(got.Masks, server) {
		t.Errorf("mask drift missed: %+v", got.Masks)
	}

	// Turned off out-of-band: masks cleared.
	got = applyDataProtectionRead(prior, &DataProtectionReadResponse{ResourceName: "db"}, false)
	if got.Masks != nil {
		t.Errorf("cleared masks not reflected: %+v", got.Masks)
	}
}

func TestApplyDataProtectionReadImport(t *testing.T) {
	f := false
	got := applyDataProtectionRead(DataProtectionArgs{}, &DataProtectionReadResponse{
		ResourceName: "db", Scoped: &f, Masks: sampleMasks(),
	}, true)
	if got.Resource != "db" || got.Scoped == nil || *got.Scoped ||
		!reflect.DeepEqual(got.Masks, sampleMasks()) {
		t.Errorf("import did not adopt server values: %+v", got)
	}
}

func TestDataProtectionClientCRUD(t *testing.T) {
	var createBody map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/terraform/dataprotection/create":
			_ = json.NewDecoder(r.Body).Decode(&createBody)
			_ = json.NewEncoder(w).Encode(DataProtectionReadResponse{
				ID: "res-1", ResourceName: "db", AuthorizationName: "masked_db", Status: "saved",
			})
		case "/api/v1/terraform/dataprotection/read/res-1":
			_ = json.NewEncoder(w).Encode(DataProtectionReadResponse{
				ID: "res-1", ResourceName: "db", AuthorizationName: "masked_db", Masks: sampleMasks(),
			})
		case "/api/v1/terraform/dataprotection/update/res-1":
			_ = json.NewEncoder(w).Encode(DataProtectionReadResponse{ID: "res-1", ResourceName: "db", AuthorizationName: "masked_db"})
		case "/api/v1/terraform/dataprotection/delete/res-1":
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	ctx := context.Background()

	created, err := c.CreateDataProtection(ctx, DataProtectionRequest{ResourceName: "db", Masks: sampleMasks()})
	if err != nil || created.ID != "res-1" || created.AuthorizationName != "masked_db" {
		t.Fatalf("create: (%+v, %v)", created, err)
	}
	if createBody["resourceName"] != "db" {
		t.Errorf("create payload: %v", createBody)
	}
	masks, ok := createBody["masks"].([]any)
	if !ok || len(masks) != 1 {
		t.Fatalf("create payload masks: %v", createBody["masks"])
	}
	db := masks[0].(map[string]any)
	if db["databaseName"] != "shop" {
		t.Errorf("mask payload: %v", db)
	}

	got, err := c.ReadDataProtection(ctx, "res-1")
	if err != nil || got == nil || got.Masks[0].Tables[0].TableName != "users" {
		t.Fatalf("read: (%+v, %v)", got, err)
	}
	if _, err := c.UpdateDataProtection(ctx, "res-1", DataProtectionRequest{ResourceName: "db"}); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteDataProtection(ctx, "res-1"); err != nil {
		t.Fatal(err)
	}
}
