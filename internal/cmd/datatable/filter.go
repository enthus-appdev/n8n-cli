package datatable

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/enthus-appdev/n8n-cli/internal/api"
)

// whereOperators maps --where operators to API conditions. Two-character
// operators come first so ">=" is not read as ">" followed by "=value".
var whereOperators = []struct{ op, condition string }{
	{">=", "gte"}, {"<=", "lte"}, {"!=", "neq"}, {"~", "ilike"},
	{"=", "eq"}, {">", "gt"}, {"<", "lt"},
}

// parseWhere turns "column<op>value" into a filter condition. Column names
// may only contain letters, digits and underscores, so the operator starts at
// the first other character. Values are sent as strings; n8n converts them to
// the column type.
func parseWhere(expr string) (api.DataTableFilterCondition, error) {
	i := strings.IndexFunc(expr, func(r rune) bool { return !isColumnChar(r) })
	if i <= 0 {
		return api.DataTableFilterCondition{}, fmt.Errorf("invalid --where %q (expected <column><op><value>, op one of = != > >= < <= ~)", expr)
	}
	column, rest := expr[:i], expr[i:]
	for _, o := range whereOperators {
		if value, ok := strings.CutPrefix(rest, o.op); ok {
			return api.DataTableFilterCondition{ColumnName: column, Condition: o.condition, Value: value}, nil
		}
	}
	return api.DataTableFilterCondition{}, fmt.Errorf("invalid operator in --where %q (use = != > >= < <= ~)", expr)
}

func isColumnChar(r rune) bool {
	return r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
}

// buildFilter combines --where conditions or a raw --filter JSON object. It
// returns nil when neither is set.
func buildFilter(wheres []string, rawFilter string, or bool) (*api.DataTableFilter, error) {
	if rawFilter != "" && len(wheres) > 0 {
		return nil, fmt.Errorf("use either --where or --filter, not both")
	}
	if rawFilter != "" {
		var f api.DataTableFilter
		if err := json.Unmarshal([]byte(rawFilter), &f); err != nil {
			return nil, fmt.Errorf("invalid --filter JSON: %w", err)
		}
		if len(f.Filters) == 0 {
			return nil, fmt.Errorf("--filter needs at least one entry in \"filters\"")
		}
		return &f, nil
	}
	if len(wheres) == 0 {
		return nil, nil
	}
	f := &api.DataTableFilter{Type: "and"}
	if or {
		f.Type = "or"
	}
	for _, w := range wheres {
		cond, err := parseWhere(w)
		if err != nil {
			return nil, err
		}
		f.Filters = append(f.Filters, cond)
	}
	return f, nil
}

// readData parses --data, reading stdin when the value is "-".
func readData(data string, stdin io.Reader) ([]byte, error) {
	if data == "-" {
		b, err := io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read stdin: %w", err)
		}
		return b, nil
	}
	if data == "" {
		return nil, fmt.Errorf("--data is required (JSON, or - to read stdin)")
	}
	return []byte(data), nil
}

// parseRows accepts a JSON object or an array of objects.
func parseRows(raw []byte) ([]map[string]interface{}, error) {
	trimmed := strings.TrimSpace(string(raw))
	if strings.HasPrefix(trimmed, "[") {
		var rows []map[string]interface{}
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, fmt.Errorf("invalid --data JSON: %w", err)
		}
		if len(rows) == 0 {
			return nil, fmt.Errorf("--data array is empty")
		}
		return rows, nil
	}
	row, err := parseRow(raw)
	if err != nil {
		return nil, err
	}
	return []map[string]interface{}{row}, nil
}

func parseRow(raw []byte) (map[string]interface{}, error) {
	var row map[string]interface{}
	if err := json.Unmarshal(raw, &row); err != nil {
		return nil, fmt.Errorf("invalid --data JSON object: %w", err)
	}
	if len(row) == 0 {
		return nil, fmt.Errorf("--data object is empty")
	}
	return row, nil
}

var stdin io.Reader = os.Stdin
