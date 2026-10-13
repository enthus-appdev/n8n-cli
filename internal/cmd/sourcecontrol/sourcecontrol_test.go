package sourcecontrol

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/api"
)

func TestBuildPushItems(t *testing.T) {
	got, err := buildPushItems([]string{"wf1"}, []string{"credential:c1", "datatable:dt1"})
	if err != nil {
		t.Fatalf("buildPushItems: %v", err)
	}
	want := []api.SourceControlItem{
		{ID: "wf1", Type: "workflow"},
		{ID: "c1", Type: "credential"},
		{ID: "dt1", Type: "datatable"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestBuildPushItemsErrors(t *testing.T) {
	tests := []struct {
		name      string
		workflows []string
		items     []string
		wantErr   string
	}{
		{"nothing selected", nil, nil, "select at least one item"},
		{"empty workflow", []string{""}, nil, "non-empty ID"},
		{"missing colon", nil, []string{"workflow"}, "expected <type>:<id>"},
		{"empty id", nil, []string{"workflow:"}, "expected <type>:<id>"},
		{"unknown type", nil, []string{"node:abc"}, `invalid --item type "node"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := buildPushItems(tt.workflows, tt.items)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("err = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestConflictErrorJSON(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("json", true, "")

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	conflictErr := &api.SourceControlConflictError{Message: "Conflicts found", Conflicts: []api.SourceControlFile{{ID: "wf1", Type: "workflow", Conflict: true}}}
	got := conflictError(cmd, "push", conflictErr)
	os.Stdout = stdout
	_ = w.Close()
	out, _ := io.ReadAll(r)

	if got == nil || !errors.Is(got, conflictErr) {
		t.Fatalf("err = %v, want wrapped conflict error", got)
	}
	var body struct {
		Error     string                  `json:"error"`
		Conflicts []api.SourceControlFile `json:"conflicts"`
	}
	if err := json.Unmarshal(out, &body); err != nil {
		t.Fatalf("stdout is not one JSON object: %v\n%s", err, out)
	}
	if !strings.Contains(body.Error, "push rejected") || len(body.Conflicts) != 1 || body.Conflicts[0].ID != "wf1" {
		t.Errorf("body = %+v", body)
	}
}
