package datatable

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/enthus-appdev/n8n-cli/internal/api"
)

func TestParseWhere(t *testing.T) {
	tests := []struct {
		expr string
		want api.DataTableFilterCondition
	}{
		{"status=done", api.DataTableFilterCondition{ColumnName: "status", Condition: "eq", Value: "done"}},
		{"status!=done", api.DataTableFilterCondition{ColumnName: "status", Condition: "neq", Value: "done"}},
		{"count>=10", api.DataTableFilterCondition{ColumnName: "count", Condition: "gte", Value: "10"}},
		{"count<=10", api.DataTableFilterCondition{ColumnName: "count", Condition: "lte", Value: "10"}},
		{"count>10", api.DataTableFilterCondition{ColumnName: "count", Condition: "gt", Value: "10"}},
		{"count<10", api.DataTableFilterCondition{ColumnName: "count", Condition: "lt", Value: "10"}},
		{"name~%inv%", api.DataTableFilterCondition{ColumnName: "name", Condition: "ilike", Value: "%inv%"}},
		{"a_b1=x=y", api.DataTableFilterCondition{ColumnName: "a_b1", Condition: "eq", Value: "x=y"}},
		{"note=", api.DataTableFilterCondition{ColumnName: "note", Condition: "eq", Value: ""}},
		{"cmp=>=", api.DataTableFilterCondition{ColumnName: "cmp", Condition: "eq", Value: ">="}},
	}
	for _, tt := range tests {
		got, err := parseWhere(tt.expr)
		if err != nil {
			t.Errorf("parseWhere(%q): %v", tt.expr, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseWhere(%q) = %+v, want %+v", tt.expr, got, tt.want)
		}
	}
}

func TestParseWhereErrors(t *testing.T) {
	for _, expr := range []string{"", "=x", "status", "status:done", "status!x"} {
		if _, err := parseWhere(expr); err == nil {
			t.Errorf("parseWhere(%q) succeeded, want error", expr)
		}
	}
}

func TestBuildFilter(t *testing.T) {
	f, err := buildFilter([]string{"a=1", "b=2"}, "", true)
	if err != nil {
		t.Fatalf("buildFilter: %v", err)
	}
	if f.Type != "or" || len(f.Filters) != 2 {
		t.Errorf("filter = %+v", f)
	}

	f, err = buildFilter(nil, `{"filters":[{"columnName":"id","condition":"eq","value":1}]}`, false)
	if err != nil {
		t.Fatalf("buildFilter raw: %v", err)
	}
	if f.Filters[0].Value != json.Number("1") {
		t.Errorf("raw filter value = %#v, want number 1", f.Filters[0].Value)
	}

	big := `{"filters":[{"columnName":"id","condition":"eq","value":9007199254740993}]}`
	f, err = buildFilter(nil, big, false)
	if err != nil {
		t.Fatalf("buildFilter big: %v", err)
	}
	out, _ := json.Marshal(f.Filters[0])
	if !strings.Contains(string(out), `"value":9007199254740993`) {
		t.Errorf("big integer re-encoded as %s", out)
	}

	if f, err := buildFilter(nil, "", false); f != nil || err != nil {
		t.Errorf("empty = %+v, %v; want nil, nil", f, err)
	}
}

func TestBuildFilterErrors(t *testing.T) {
	tests := []struct {
		wheres  []string
		raw     string
		wantErr string
	}{
		{[]string{"a=1"}, `{"filters":[]}`, "either --where or --filter"},
		{nil, `{"filters":[]}`, "at least one entry"},
		{nil, `not json`, "invalid --filter JSON"},
		{[]string{"bad"}, "", "invalid --where"},
	}
	for _, tt := range tests {
		_, err := buildFilter(tt.wheres, tt.raw, false)
		if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("buildFilter(%v, %q) err = %v, want %q", tt.wheres, tt.raw, err, tt.wantErr)
		}
	}
}

func TestParseRows(t *testing.T) {
	rows, err := parseRows([]byte(` [{"a":1},{"a":2}]`))
	if err != nil || len(rows) != 2 {
		t.Errorf("array: rows = %v, err = %v", rows, err)
	}
	rows, err = parseRows([]byte(`{"a":1}`))
	if err != nil || len(rows) != 1 {
		t.Errorf("object: rows = %v, err = %v", rows, err)
	}
	rows, err = parseRows([]byte(`{"n":9007199254740993}`))
	if err != nil {
		t.Fatalf("big: %v", err)
	}
	if out, _ := json.Marshal(rows[0]); string(out) != `{"n":9007199254740993}` {
		t.Errorf("big integer re-encoded as %s", out)
	}
	for _, bad := range []string{`[]`, `{}`, `null`, `"x"`, `[1]`, `{"a":1} {"b":2}`} {
		if _, err := parseRows([]byte(bad)); err == nil {
			t.Errorf("parseRows(%s) succeeded, want error", bad)
		}
	}
}

func TestReadData(t *testing.T) {
	b, err := readData("-", strings.NewReader(`{"a":1}`))
	if err != nil || string(b) != `{"a":1}` {
		t.Errorf("stdin: %q, %v", b, err)
	}
	if _, err := readData("", strings.NewReader("")); err == nil {
		t.Error("empty --data succeeded, want error")
	}
}
