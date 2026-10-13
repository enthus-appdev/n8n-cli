package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// DataTable is an n8n data table.
type DataTable struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	ProjectID string            `json:"projectId"`
	Columns   []DataTableColumn `json:"columns"`
	SizeBytes int64             `json:"sizeBytes"`
	CreatedAt *time.Time        `json:"createdAt,omitempty"`
	UpdatedAt *time.Time        `json:"updatedAt,omitempty"`
}

// DataTableColumn is a column of a data table.
type DataTableColumn struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Index int    `json:"index"`
}

// DataTableFilter matches rows. Type is "and" (server default) or "or".
type DataTableFilter struct {
	Type    string                     `json:"type,omitempty"`
	Filters []DataTableFilterCondition `json:"filters"`
}

// DataTableFilterCondition compares a column to a value. Condition is one of
// eq, neq, like, ilike, gt, gte, lt, lte.
type DataTableFilterCondition struct {
	ColumnName string      `json:"columnName"`
	Condition  string      `json:"condition"`
	Value      interface{} `json:"value"`
}

func dataTablePath(id string) string {
	return "/data-tables/" + url.PathEscape(id)
}

// ListDataTables returns one page of data tables.
func (c *Client) ListDataTables(limit int, cursor string) (*ListResult[DataTable], error) {
	return getJSON[ListResult[DataTable]](c, pagedPath("/data-tables", limit, cursor, nil))
}

// GetDataTable returns a data table with its columns.
func (c *Client) GetDataTable(id string) (*DataTable, error) {
	return getJSON[DataTable](c, dataTablePath(id))
}

// ListRowsOptions selects rows of a data table.
type ListRowsOptions struct {
	Filter *DataTableFilter
	Search string
	SortBy string // column:asc or column:desc
	Limit  int
	Cursor string
}

// ListDataTableRows returns one page of rows.
func (c *Client) ListDataTableRows(id string, opts ListRowsOptions) (*ListResult[map[string]interface{}], error) {
	extra := url.Values{}
	if opts.Filter != nil {
		f, err := json.Marshal(opts.Filter)
		if err != nil {
			return nil, err
		}
		extra.Set("filter", string(f))
	}
	if opts.Search != "" {
		extra.Set("search", opts.Search)
	}
	if opts.SortBy != "" {
		extra.Set("sortBy", opts.SortBy)
	}
	return getJSON[ListResult[map[string]interface{}]](c, pagedPath(dataTablePath(id)+"/rows", opts.Limit, opts.Cursor, extra))
}

// InsertDataTableRows inserts rows and returns the inserted rows.
func (c *Client) InsertDataTableRows(id string, rows []map[string]interface{}) (json.RawMessage, error) {
	return c.rawJSON(http.MethodPost, dataTablePath(id)+"/rows", map[string]interface{}{"data": rows, "returnType": "all"})
}

// RowWriteOptions controls an update, upsert, or delete of rows matching Filter.
type RowWriteOptions struct {
	Filter DataTableFilter
	Data   map[string]interface{}
	DryRun bool
}

// UpdateDataTableRows updates every row matching opts.Filter and returns the affected rows.
func (c *Client) UpdateDataTableRows(id string, opts RowWriteOptions) (json.RawMessage, error) {
	return c.rawJSON(http.MethodPatch, dataTablePath(id)+"/rows/update", map[string]interface{}{
		"filter": opts.Filter, "data": opts.Data, "returnData": true, "dryRun": opts.DryRun,
	})
}

// UpsertDataTableRow updates the rows matching opts.Filter, or inserts opts.Data
// when none match, and returns the affected rows.
func (c *Client) UpsertDataTableRow(id string, opts RowWriteOptions) (json.RawMessage, error) {
	return c.rawJSON(http.MethodPost, dataTablePath(id)+"/rows/upsert", map[string]interface{}{
		"filter": opts.Filter, "data": opts.Data, "returnData": true, "dryRun": opts.DryRun,
	})
}

// DeleteDataTableRows deletes the rows matching opts.Filter and returns them.
// The server requires a filter, so this cannot clear a whole table.
func (c *Client) DeleteDataTableRows(id string, opts RowWriteOptions) (json.RawMessage, error) {
	f, err := json.Marshal(opts.Filter)
	if err != nil {
		return nil, err
	}
	params := url.Values{
		"filter":     {string(f)},
		"returnData": {"true"},
		"dryRun":     {strconv.FormatBool(opts.DryRun)},
	}
	return c.rawJSON(http.MethodDelete, dataTablePath(id)+"/rows/delete?"+params.Encode(), nil)
}

func (c *Client) rawJSON(method, path string, body interface{}) (json.RawMessage, error) {
	respBody, err := c.request(method, path, body)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(respBody), nil
}
