package execution

import (
	"strings"
	"testing"
)

func TestStopArgValidation(t *testing.T) {
	tests := []struct {
		args    []string
		wantErr string
	}{
		{nil, "pass an execution ID, or --status"},
		{[]string{"42", "--status", "running"}, "either an execution ID or bulk filters"},
		{[]string{"42", "--workflow", "wf1"}, "either an execution ID or bulk filters"},
		{[]string{"--workflow", "wf1"}, "pass an execution ID, or --status"},
		{[]string{"--status", "error"}, `invalid --status "error"`},
	}
	for _, tt := range tests {
		cmd := newStopCmd()
		cmd.SetArgs(tt.args)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("args %v: err = %v, want %q", tt.args, err, tt.wantErr)
		}
	}
}
