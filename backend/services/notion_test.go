package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock Notion API server
func newMockNotionServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		switch {
		// Search databases endpoint
		case r.URL.Path == "/v1/search" && r.Method == http.MethodPost:
			var reqBody map[string]interface{}
			json.NewDecoder(r.Body).Decode(&reqBody)

			filter, ok := reqBody["filter"].(map[string]interface{})
			if ok && filter["value"] == "database" {
				// Return mock databases
				response := map[string]interface{}{
					"object": "list",
					"results": []map[string]interface{}{
						{
							"object":           "database",
							"id":               "db-123",
							"created_time":     "2024-01-01T00:00:00.000Z",
							"last_edited_time": "2024-01-02T00:00:00.000Z",
							"title": []map[string]interface{}{
								{"type": "text", "plain_text": "Test Database", "text": map[string]interface{}{"content": "Test Database"}},
							},
							"url": "https://notion.so/db-123",
							"properties": map[string]interface{}{
								"Name": map[string]interface{}{"id": "title", "type": "title"},
							},
						},
					},
					"has_more": false,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			// Regular search
			response := map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{
						"object":           "page",
						"id":               "page-123",
						"created_time":     "2024-01-01T00:00:00.000Z",
						"last_edited_time": "2024-01-02T00:00:00.000Z",
						"url":              "https://notion.so/page-123",
						"properties": map[string]interface{}{
							"Name": map[string]interface{}{"title": []map[string]interface{}{
								{"type": "text", "plain_text": "Test Page"},
							}},
						},
					},
				},
				"has_more": false,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)

		// Get database endpoint
		case r.URL.Path == "/v1/databases/db-123" && r.Method == http.MethodGet:
			response := map[string]interface{}{
				"object":           "database",
				"id":               "db-123",
				"created_time":     "2024-01-01T00:00:00.000Z",
				"last_edited_time": "2024-01-02T00:00:00.000Z",
				"title": []map[string]interface{}{
					{"type": "text", "plain_text": "Test Database", "text": map[string]interface{}{"content": "Test Database"}},
				},
				"url": "https://notion.so/db-123",
				"properties": map[string]interface{}{
					"Name":   map[string]interface{}{"id": "title", "type": "title"},
					"Status": map[string]interface{}{"id": "select", "type": "select"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)

		// Query database endpoint
		case r.URL.Path == "/v1/databases/db-123/query" && r.Method == http.MethodPost:
			response := map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{
						"object":           "page",
						"id":               "page-1",
						"created_time":     "2024-01-01T00:00:00.000Z",
						"last_edited_time": "2024-01-02T00:00:00.000Z",
						"url":              "https://notion.so/page-1",
						"properties": map[string]interface{}{
							"Name": map[string]interface{}{
								"title": []map[string]interface{}{
									{"type": "text", "plain_text": "Item 1"},
								},
							},
							"Status": map[string]interface{}{
								"select": map[string]interface{}{"name": "Done"},
							},
						},
					},
				},
				"has_more": false,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)

		// Get page endpoint
		case r.URL.Path == "/v1/pages/page-123" && r.Method == http.MethodGet:
			response := map[string]interface{}{
				"object":           "page",
				"id":               "page-123",
				"created_time":     "2024-01-01T00:00:00.000Z",
				"last_edited_time": "2024-01-02T00:00:00.000Z",
				"url":              "https://notion.so/page-123",
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"title": []map[string]interface{}{
							{"type": "text", "plain_text": "Test Page"},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)

		// Create page endpoint
		case r.URL.Path == "/v1/pages" && r.Method == http.MethodPost:
			var reqBody map[string]interface{}
			json.NewDecoder(r.Body).Decode(&reqBody)

			response := map[string]interface{}{
				"object":           "page",
				"id":               "new-page-123",
				"created_time":     "2024-01-03T00:00:00.000Z",
				"last_edited_time": "2024-01-03T00:00:00.000Z",
				"url":              "https://notion.so/new-page-123",
				"properties":       reqBody["properties"],
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)

		// Update page endpoint
		case r.URL.Path == "/v1/pages/page-123" && r.Method == http.MethodPatch:
			var reqBody map[string]interface{}
			json.NewDecoder(r.Body).Decode(&reqBody)

			response := map[string]interface{}{
				"object":           "page",
				"id":               "page-123",
				"created_time":     "2024-01-01T00:00:00.000Z",
				"last_edited_time": "2024-01-04T00:00:00.000Z",
				"url":              "https://notion.so/page-123",
				"properties":       reqBody["properties"],
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)

		// Archive page endpoint
		case r.URL.Path == "/v1/pages/page-to-delete" && r.Method == http.MethodPatch:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       "page-to-delete",
				"archived": true,
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestNewNotionService(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		config := &NotionConfig{
			APIKey: "test-key",
		}
		service := NewNotionService(config)

		assert.NotNil(t, service)
		assert.NotNil(t, service.client)
	})

	t.Run("custom configuration", func(t *testing.T) {
		config := &NotionConfig{
			APIKey:        "test-key",
			APIBaseURL:    "https://custom.api.notion.com",
			NotionVersion: "2025-01-01",
			Timeout:       60 * time.Second,
		}
		service := NewNotionService(config)

		assert.NotNil(t, service)
		assert.NotNil(t, service.client)
	})
}

func TestSearchDatabases(t *testing.T) {
	mockServer := newMockNotionServer(t)
	defer mockServer.Close()

	config := &NotionConfig{
		APIKey:     "test-key",
		APIBaseURL: mockServer.URL + "/v1",
	}
	service := NewNotionService(config)

	databases, err := service.SearchDatabases()

	require.NoError(t, err)
	assert.Len(t, databases, 1)
	assert.Equal(t, "db-123", databases[0].ID)
	assert.Equal(t, "database", databases[0].Object)
	assert.Equal(t, "Test Database", databases[0].Title[0].PlainText)
	assert.Equal(t, "https://notion.so/db-123", databases[0].URL)
}

func TestGetDatabase(t *testing.T) {
	mockServer := newMockNotionServer(t)
	defer mockServer.Close()

	config := &NotionConfig{
		APIKey:     "test-key",
		APIBaseURL: mockServer.URL + "/v1",
	}
	service := NewNotionService(config)

	db, err := service.GetDatabase("db-123")

	require.NoError(t, err)
	assert.NotNil(t, db)
	assert.Equal(t, "db-123", db.ID)
	assert.Equal(t, "database", db.Object)
	assert.Equal(t, "Test Database", db.Title[0].PlainText)
	assert.Contains(t, db.Properties, "Name")
	assert.Contains(t, db.Properties, "Status")
}

func TestGetDatabase_Error(t *testing.T) {
	mockServer := newMockNotionServer(t)
	defer mockServer.Close()

	config := &NotionConfig{
		APIKey:     "test-key",
		APIBaseURL: mockServer.URL + "/v1",
	}
	service := NewNotionService(config)

	_, err := service.GetDatabase("non-existent")

	assert.Error(t, err)
}

func TestQueryDatabase(t *testing.T) {
	mockServer := newMockNotionServer(t)
	defer mockServer.Close()

	config := &NotionConfig{
		APIKey:     "test-key",
		APIBaseURL: mockServer.URL + "/v1",
	}
	service := NewNotionService(config)

	filter := map[string]interface{}{
		"property": "Status",
		"select": map[string]interface{}{
			"equals": "Done",
		},
	}

	sorts := []map[string]interface{}{
		{"property": "Name", "direction": "ascending"},
	}

	pages, err := service.QueryDatabase("db-123", filter, sorts)

	require.NoError(t, err)
	assert.Len(t, pages, 1)
	assert.Equal(t, "page-1", pages[0].ID)
	assert.Equal(t, "page", pages[0].Object)
}

func TestQueryDatabaseWithParams(t *testing.T) {
	mockServer := newMockNotionServer(t)
	defer mockServer.Close()

	config := &NotionConfig{
		APIKey:     "test-key",
		APIBaseURL: mockServer.URL + "/v1",
	}
	service := NewNotionService(config)

	query := &QueryRequest{
		Filter: map[string]interface{}{
			"property": "Status",
			"select": map[string]interface{}{
				"equals": "Done",
			},
		},
		Sorts: []map[string]interface{}{
			{"property": "Name", "direction": "ascending"},
		},
		PageSize: 50,
	}

	result, err := service.QueryDatabaseWithParams("db-123", query)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "list", result.Object)
	assert.Len(t, result.Results, 1)
	assert.False(t, result.HasMore)
}

func TestGetPage(t *testing.T) {
	mockServer := newMockNotionServer(t)
	defer mockServer.Close()

	config := &NotionConfig{
		APIKey:     "test-key",
		APIBaseURL: mockServer.URL + "/v1",
	}
	service := NewNotionService(config)

	page, err := service.GetPage("page-123")

	require.NoError(t, err)
	assert.NotNil(t, page)
	assert.Equal(t, "page-123", page.ID)
	assert.Equal(t, "page", page.Object)
	assert.Equal(t, "https://notion.so/page-123", page.URL)
}

func TestCreatePage(t *testing.T) {
	mockServer := newMockNotionServer(t)
	defer mockServer.Close()

	config := &NotionConfig{
		APIKey:     "test-key",
		APIBaseURL: mockServer.URL + "/v1",
	}
	service := NewNotionService(config)

	properties := map[string]interface{}{
		"Name": map[string]interface{}{
			"title": []map[string]interface{}{
				{"type": "text", "text": map[string]interface{}{"content": "New Page"}},
			},
		},
	}

	page, err := service.CreatePage("db-123", properties)

	require.NoError(t, err)
	assert.NotNil(t, page)
	assert.Equal(t, "new-page-123", page.ID)
	assert.Equal(t, "page", page.Object)
	assert.Equal(t, "https://notion.so/new-page-123", page.URL)
}

func TestUpdatePage(t *testing.T) {
	mockServer := newMockNotionServer(t)
	defer mockServer.Close()

	config := &NotionConfig{
		APIKey:     "test-key",
		APIBaseURL: mockServer.URL + "/v1",
	}
	service := NewNotionService(config)

	properties := map[string]interface{}{
		"Status": map[string]interface{}{
			"select": map[string]interface{}{"name": "In Progress"},
		},
	}

	page, err := service.UpdatePage("page-123", properties)

	require.NoError(t, err)
	assert.NotNil(t, page)
	assert.Equal(t, "page-123", page.ID)
}

func TestArchivePage(t *testing.T) {
	mockServer := newMockNotionServer(t)
	defer mockServer.Close()

	config := &NotionConfig{
		APIKey:     "test-key",
		APIBaseURL: mockServer.URL + "/v1",
	}
	service := NewNotionService(config)

	err := service.ArchivePage("page-to-delete")

	require.NoError(t, err)
}

func TestListDatabases(t *testing.T) {
	mockServer := newMockNotionServer(t)
	defer mockServer.Close()

	config := &NotionConfig{
		APIKey:     "test-key",
		APIBaseURL: mockServer.URL + "/v1",
	}
	service := NewNotionService(config)

	databases, err := service.ListDatabases()

	require.NoError(t, err)
	assert.Len(t, databases, 1)
	assert.Equal(t, "db-123", databases[0].ID)
}

func TestSearch(t *testing.T) {
	mockServer := newMockNotionServer(t)
	defer mockServer.Close()

	config := &NotionConfig{
		APIKey:     "test-key",
		APIBaseURL: mockServer.URL + "/v1",
	}
	service := NewNotionService(config)

	// Search with query
	pages, err := service.Search("test query", "")

	require.NoError(t, err)
	assert.Len(t, pages, 1)
	assert.Equal(t, "page-123", pages[0].ID)
	assert.Equal(t, "page", pages[0].Object)

	// Search with filter
	pages, err = service.Search("", "page")

	require.NoError(t, err)
	assert.Len(t, pages, 1)
}

func TestSearch_EmptyFilter(t *testing.T) {
	mockServer := newMockNotionServer(t)
	defer mockServer.Close()

	config := &NotionConfig{
		APIKey:     "test-key",
		APIBaseURL: mockServer.URL + "/v1",
	}
	service := NewNotionService(config)

	pages, err := service.Search("test", "")

	require.NoError(t, err)
	assert.NotNil(t, pages)
}

func TestRichText_Unmarshal(t *testing.T) {
	jsonData := `{
		"type": "text",
		"plain_text": "Hello World",
		"text": {
			"content": "Hello World Content"
		}
	}`

	var rt RichText
	err := json.Unmarshal([]byte(jsonData), &rt)

	require.NoError(t, err)
	assert.Equal(t, "text", rt.Type)
	assert.Equal(t, "Hello World", rt.PlainText)
	assert.Equal(t, "Hello World Content", rt.Text.Content)
}

func TestPage_Unmarshal(t *testing.T) {
	jsonData := `{
		"object": "page",
		"id": "test-page-id",
		"created_time": "2024-01-01T00:00:00.000Z",
		"last_edited_time": "2024-01-02T00:00:00.000Z",
		"url": "https://notion.so/test-page",
		"properties": {
			"Name": {
				"title": [
					{"type": "text", "plain_text": "Test Name"}
				]
			}
		}
	}`

	var page Page
	err := json.Unmarshal([]byte(jsonData), &page)

	require.NoError(t, err)
	assert.Equal(t, "page", page.Object)
	assert.Equal(t, "test-page-id", page.ID)
	assert.Equal(t, "https://notion.so/test-page", page.URL)
	assert.Contains(t, page.Properties, "Name")
}

func TestQueryResponse_Unmarshal(t *testing.T) {
	jsonData := `{
		"object": "list",
		"results": [
			{
				"object": "page",
				"id": "result-1",
				"created_time": "2024-01-01T00:00:00.000Z"
			}
		],
		"has_more": false,
		"next_cursor": null,
		"type": "page_or_database"
	}`

	var resp QueryResponse
	err := json.Unmarshal([]byte(jsonData), &resp)

	require.NoError(t, err)
	assert.Equal(t, "list", resp.Object)
	assert.Len(t, resp.Results, 1)
	assert.Equal(t, "result-1", resp.Results[0].ID)
	assert.False(t, resp.HasMore)
	assert.Nil(t, resp.NextCursor)
}
