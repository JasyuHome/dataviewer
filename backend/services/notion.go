package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

// NotionService handles Notion API interactions
type NotionService struct {
	client *resty.Client
}

// Database represents a Notion database
type Database struct {
	Object         string                 `json:"object"`
	ID             string                 `json:"id"`
	CreatedTime    string                 `json:"created_time"`
	LastEditedTime string                 `json:"last_edited_time"`
	Title          []RichText             `json:"title"`
	URL            string                 `json:"url"`
	Properties     map[string]interface{} `json:"properties"`
}

// RichText represents Notion rich text property
type RichText struct {
	Type      string `json:"type"`
	PlainText string `json:"plain_text"`
	Text      struct {
		Content string `json:"content"`
	} `json:"text"`
}

// Page represents a Notion page
type Page struct {
	Object         string                 `json:"object"`
	ID             string                 `json:"id"`
	CreatedTime    string                 `json:"created_time"`
	LastEditedTime string                 `json:"last_edited_time"`
	Properties     map[string]interface{} `json:"properties"`
	URL            string                 `json:"url"`
}

// QueryRequest defines query parameters for database queries
type QueryRequest struct {
	Filter      interface{} `json:"filter,omitempty"`
	Sorts       interface{} `json:"sorts,omitempty"`
	StartCursor string      `json:"start_cursor,omitempty"`
	PageSize    int         `json:"page_size,omitempty"`
}

// QueryResponse represents query results
type QueryResponse struct {
	Object     string  `json:"object"`
	Results    []Page  `json:"results"`
	HasMore    bool    `json:"has_more"`
	NextCursor *string `json:"next_cursor"`
	Type       string  `json:"type"`
}

// NotionConfig holds Notion API configuration
type NotionConfig struct {
	APIKey        string
	APIBaseURL    string
	NotionVersion string
	Timeout       time.Duration
}

// NewNotionService creates a new Notion service instance
func NewNotionService(config *NotionConfig) *NotionService {
	if config.APIBaseURL == "" {
		config.APIBaseURL = "https://api.notion.com/v1"
	}
	if config.NotionVersion == "" {
		config.NotionVersion = "2022-06-28"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	cli := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 5,
			MaxConnsPerHost:     20,
		},
	}

	return &NotionService{
		client: resty.NewWithClient(cli).
			SetBaseURL(config.APIBaseURL).
			SetHeader("Authorization", "Bearer "+config.APIKey).
			SetHeader("Content-Type", "application/json").
			SetHeader("Notion-Version", config.NotionVersion),
	}
}

// SearchDatabases searches for all databases
func (s *NotionService) SearchDatabases() ([]Database, error) {
	body := map[string]any{
		"filter": map[string]string{
			"property": "object",
			"value":    "database",
		},
	}

	resp, err := s.client.R().
		SetHeader("Notion-Version", "2022-06-28").
		SetBody(body).
		Post("/search")
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("Notion API error [%d]: %s", resp.StatusCode(), string(resp.Body()))
	}

	var result struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var databases []Database
	for _, raw := range result.Results {
		var item struct {
			Object string `json:"object"`
			ID     string `json:"id"`
		}
		if err := json.Unmarshal(raw, &item); err == nil && item.Object == "database" {
			var db Database
			if err := json.Unmarshal(raw, &db); err == nil {
				databases = append(databases, db)
			}
		}
	}

	return databases, nil
}

// GetDatabase retrieves a database by ID
func (s *NotionService) GetDatabase(id string) (*Database, error) {
	resp, err := s.client.R().
		SetResult(&Database{}).
		Get("/databases/" + id)

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("Notion API error [%d]: %s", resp.StatusCode(), string(resp.Body()))
	}

	return resp.Result().(*Database), nil
}

// QueryDatabase queries a database with filter and sorts
func (s *NotionService) QueryDatabase(id string, filter, sorts interface{}) ([]Page, error) {
	var allResults []Page
	var nextCursor string
	pageCount := 0

	for {
		pageCount++
		query := &QueryRequest{
			Filter:      filter,
			Sorts:       sorts,
			StartCursor: nextCursor,
			PageSize:    100,
		}

		result, err := s.QueryDatabaseWithParams(id, query)
		if err != nil {
			return nil, err
		}

		allResults = append(allResults, result.Results...)

		if !result.HasMore {
			break
		}
		if result.NextCursor != nil && *result.NextCursor != "" {
			nextCursor = *result.NextCursor
		} else {
			break
		}
	}

	return allResults, nil
}

// QueryDatabaseWithParams queries a database with query params
func (s *NotionService) QueryDatabaseWithParams(id string, query *QueryRequest) (*QueryResponse, error) {
	body := make(map[string]interface{})

	// Only add filter if it's not nil and not an empty map
	if query.Filter != nil {
		if filterMap, ok := query.Filter.(map[string]interface{}); ok && len(filterMap) > 0 {
			body["filter"] = filterMap
		}
	}

	// Only add sorts if it's not nil and not empty
	if query.Sorts != nil {
		if sortsArr, ok := query.Sorts.([]map[string]interface{}); ok && len(sortsArr) > 0 {
			body["sorts"] = sortsArr
		}
	}

	if query.StartCursor != "" {
		body["start_cursor"] = query.StartCursor
	}
	if query.PageSize > 0 {
		body["page_size"] = query.PageSize
	}

	resp, err := s.client.R().
		SetBody(body).
		SetResult(&QueryResponse{}).
		Post("/databases/" + id + "/query")

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("Notion API error [%d]: %s", resp.StatusCode(), string(resp.Body()))
	}

	return resp.Result().(*QueryResponse), nil
}

// GetPage retrieves a page by ID
func (s *NotionService) GetPage(id string) (*Page, error) {
	resp, err := s.client.R().
		SetResult(&Page{}).
		Get("/pages/" + id)

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("Notion API error [%d]: %s", resp.StatusCode(), string(resp.Body()))
	}

	return resp.Result().(*Page), nil
}

// CreatePage creates a new page in a database
func (s *NotionService) CreatePage(databaseID string, properties map[string]interface{}) (*Page, error) {
	body := map[string]interface{}{
		"parent": map[string]interface{}{
			"database_id": databaseID,
		},
		"properties": properties,
	}

	resp, err := s.client.R().
		SetBody(body).
		SetResult(&Page{}).
		Post("/pages")

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("Notion API error [%d]: %s", resp.StatusCode(), string(resp.Body()))
	}

	return resp.Result().(*Page), nil
}

// UpdatePage updates a page
func (s *NotionService) UpdatePage(id string, properties map[string]interface{}) (*Page, error) {
	body := map[string]interface{}{
		"properties": properties,
	}

	resp, err := s.client.R().
		SetBody(body).
		SetResult(&Page{}).
		Patch("/pages/" + id)

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("Notion API error [%d]: %s", resp.StatusCode(), string(resp.Body()))
	}

	return resp.Result().(*Page), nil
}

// ArchivePage archives (soft deletes) a page
func (s *NotionService) ArchivePage(id string) error {
	body := map[string]interface{}{
		"archived": true,
	}

	resp, err := s.client.R().
		SetBody(body).
		Patch("/pages/" + id)

	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("Notion API error [%d]: %s", resp.StatusCode(), string(resp.Body()))
	}

	return nil
}

// ListDatabases lists all databases (alias for SearchDatabases)
func (s *NotionService) ListDatabases() ([]Database, error) {
	return s.SearchDatabases()
}

// Search searches for pages or databases
func (s *NotionService) Search(query string, filterProperty string) ([]Page, error) {
	body := map[string]any{
		"query": query,
	}
	if filterProperty != "" {
		body["filter"] = map[string]string{
			"property": "object",
			"value":    filterProperty,
		}
	}

	resp, err := s.client.R().
		SetHeader("Notion-Version", "2022-06-28").
		SetBody(body).
		Post("/search")
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("Notion API error [%d]: %s", resp.StatusCode(), string(resp.Body()))
	}

	var result struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var pages []Page
	for _, raw := range result.Results {
		var item struct {
			Object string `json:"object"`
			ID     string `json:"id"`
		}
		if err := json.Unmarshal(raw, &item); err == nil && item.Object == "page" {
			var page Page
			if err := json.Unmarshal(raw, &page); err == nil {
				pages = append(pages, page)
			}
		}
	}

	return pages, nil
}
