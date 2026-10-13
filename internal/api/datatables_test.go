package api

import (
	"net/http"
	"testing"
)

var testFilter = DataTableFilter{Type: "and", Filters: []DataTableFilterCondition{{ColumnName: "status", Condition: "eq", Value: "pending"}}}

const testFilterJSON = `{"type":"and","filters":[{"columnName":"status","condition":"eq","value":"pending"}]}`

func TestListDataTableRows(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusOK, `{"data":[{"id":1,"status":"pending"}],"nextCursor":"c2"}`)

	result, err := client.ListDataTableRows("dt1", ListRowsOptions{Filter: &testFilter, Search: "foo", SortBy: "id:desc", Limit: 5})
	if err != nil {
		t.Fatalf("ListDataTableRows: %v", err)
	}
	if rec.Path != "/api/v1/data-tables/dt1/rows" {
		t.Errorf("path = %s", rec.Path)
	}
	assertJSONEqual(t, rec.Query.Get("filter"), testFilterJSON)
	if rec.Query.Get("search") != "foo" || rec.Query.Get("sortBy") != "id:desc" || rec.Query.Get("limit") != "5" {
		t.Errorf("query = %s", rec.Query.Encode())
	}
	if len(result.Data) != 1 || result.NextCursor != "c2" {
		t.Errorf("result = %+v", result)
	}
}

func TestListDataTableRowsWithoutFilter(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusOK, `{"data":[],"nextCursor":null}`)

	if _, err := client.ListDataTableRows("dt1", ListRowsOptions{}); err != nil {
		t.Fatalf("ListDataTableRows: %v", err)
	}
	if len(rec.Query) != 0 {
		t.Errorf("query = %s, want none", rec.Query.Encode())
	}
}

func TestInsertDataTableRows(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusOK, `[{"id":7,"name":"a"}]`)

	out, err := client.InsertDataTableRows("dt1", []map[string]interface{}{{"name": "a"}})
	if err != nil {
		t.Fatalf("InsertDataTableRows: %v", err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/api/v1/data-tables/dt1/rows" {
		t.Errorf("request = %s %s", rec.Method, rec.Path)
	}
	assertJSONEqual(t, rec.Body, `{"data":[{"name":"a"}],"returnType":"all"}`)
	assertJSONEqual(t, string(out), `[{"id":7,"name":"a"}]`)
}

func TestUpdateAndUpsertDataTableRows(t *testing.T) {
	opts := RowWriteOptions{Filter: testFilter, Data: map[string]interface{}{"status": "done"}, DryRun: true}
	want := `{"filter":` + testFilterJSON + `,"data":{"status":"done"},"returnData":true,"dryRun":true}`

	client, rec := newRecordingServer(t, http.StatusOK, `[]`)
	if _, err := client.UpdateDataTableRows("dt1", opts); err != nil {
		t.Fatalf("UpdateDataTableRows: %v", err)
	}
	if rec.Method != http.MethodPatch || rec.Path != "/api/v1/data-tables/dt1/rows/update" {
		t.Errorf("update request = %s %s", rec.Method, rec.Path)
	}
	assertJSONEqual(t, rec.Body, want)

	client, rec = newRecordingServer(t, http.StatusOK, `{"id":1}`)
	if _, err := client.UpsertDataTableRow("dt1", opts); err != nil {
		t.Fatalf("UpsertDataTableRow: %v", err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/api/v1/data-tables/dt1/rows/upsert" {
		t.Errorf("upsert request = %s %s", rec.Method, rec.Path)
	}
	assertJSONEqual(t, rec.Body, want)
}

func TestDeleteDataTableRows(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusOK, `[{"id":1}]`)

	if _, err := client.DeleteDataTableRows("dt1", RowWriteOptions{Filter: testFilter}); err != nil {
		t.Fatalf("DeleteDataTableRows: %v", err)
	}
	if rec.Method != http.MethodDelete || rec.Path != "/api/v1/data-tables/dt1/rows/delete" {
		t.Errorf("request = %s %s", rec.Method, rec.Path)
	}
	assertJSONEqual(t, rec.Query.Get("filter"), testFilterJSON)
	if rec.Query.Get("returnData") != "true" || rec.Query.Get("dryRun") != "false" {
		t.Errorf("query = %s", rec.Query.Encode())
	}
}
