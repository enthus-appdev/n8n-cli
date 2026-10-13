package sourcecontrol

import (
	"reflect"
	"strings"
	"testing"

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
