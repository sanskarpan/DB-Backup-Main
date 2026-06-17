package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/sanskarpan/db-backup/internal/catalog"
)

// MockSearchEngine is a mock implementation of catalog.SearchEngineInterface
type MockSearchEngine struct {
	mock.Mock
}

func (m *MockSearchEngine) IsAvailable() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSearchEngine) Search(ctx context.Context, query *catalog.SearchQuery) (*catalog.SearchResults, error) {
	args := m.Called(ctx, query)
	if result := args.Get(0); result != nil {
		return result.(*catalog.SearchResults), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSearchEngine) ParseQueryString(queryString string) (*catalog.SearchQuery, error) {
	args := m.Called(queryString)
	if result := args.Get(0); result != nil {
		return result.(*catalog.SearchQuery), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSearchEngine) Suggest(ctx context.Context, prefix string, field string, limit int) ([]string, error) {
	args := m.Called(ctx, prefix, field, limit)
	if result := args.Get(0); result != nil {
		return result.([]string), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSearchEngine) GetStats(ctx context.Context) (*catalog.CatalogStats, error) {
	args := m.Called(ctx)
	if result := args.Get(0); result != nil {
		return result.(*catalog.CatalogStats), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestHandleSearchCatalog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful search with filters", func(t *testing.T) {
		mockEngine := new(MockSearchEngine)
		mockEngine.On("IsAvailable").Return(true)
		server := &Server{searchEngine: mockEngine}

		// Setup mock expectations
		mockResults := &catalog.SearchResults{
			Results: []*catalog.SearchResult{
				{
					Backup: &catalog.BackupDocument{
						ID:              "backup-1",
						DatabaseName:    "production",
						DatabaseType:    "postgres",
						BackupType:      "full",
						Status:          "success",
						SizeBytes:       1024*1024*100, // 100MB
						Duration:        45.5,
						StoragePath:     "/backups/prod-20240101.sql.gz",
						StorageProvider: "s3",
						CreatedAt:       time.Now(),
						Tags:            map[string]string{"env": "production"},
					},
					Score: 1.0,
				},
			},
			Total: 1,
			Took:  50,
		}

		mockEngine.On("Search", mock.Anything, mock.MatchedBy(func(q *catalog.SearchQuery) bool {
			return q.Text == "production" && len(q.DatabaseTypes) == 1 && q.DatabaseTypes[0] == "postgres"
		})).Return(mockResults, nil)

		// Create request
		reqBody := SearchRequest{
			Text:          "production",
			DatabaseTypes: []string{"postgres"},
			Status:        []string{"success"},
			Limit:         100,
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/catalog/search", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		// Execute
		server.handleSearchCatalog(c)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)

		var response SearchResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, int64(1), response.Total)
		assert.Equal(t, 1, len(response.Results))
		assert.Equal(t, "backup-1", response.Results[0].BackupID)
		assert.Equal(t, "production", response.Results[0].DatabaseName)
		assert.Equal(t, "postgres", response.Results[0].DatabaseType)
		assert.False(t, response.HasMore)
		assert.Nil(t, response.NextOffset)

		mockEngine.AssertExpectations(t)
	})

	t.Run("search with date filters", func(t *testing.T) {
		mockEngine := new(MockSearchEngine)
		mockEngine.On("IsAvailable").Return(true)
		server := &Server{searchEngine: mockEngine}

		dateFrom := time.Now().Add(-7 * 24 * time.Hour)
		dateTo := time.Now()

		mockResults := &catalog.SearchResults{
			Results: []*catalog.SearchResult{},
			Total:   0,
			Took:    10,
		}

		mockEngine.On("Search", mock.Anything, mock.MatchedBy(func(q *catalog.SearchQuery) bool {
			return q.DateFrom != nil && q.DateTo != nil
		})).Return(mockResults, nil)

		reqBody := SearchRequest{
			DateFrom: dateFrom.Format(time.RFC3339),
			DateTo:   dateTo.Format(time.RFC3339),
			Limit:    50,
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/catalog/search", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		server.handleSearchCatalog(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockEngine.AssertExpectations(t)
	})

	t.Run("search with size filters", func(t *testing.T) {
		mockEngine := new(MockSearchEngine)
		mockEngine.On("IsAvailable").Return(true)
		server := &Server{searchEngine: mockEngine}

		minSize := int64(1024 * 1024 * 100)  // 100MB
		maxSize := int64(1024 * 1024 * 1000) // 1GB

		mockResults := &catalog.SearchResults{
			Results: []*catalog.SearchResult{},
			Total:   0,
			Took:    15,
		}

		mockEngine.On("Search", mock.Anything, mock.MatchedBy(func(q *catalog.SearchQuery) bool {
			return q.MinSize != nil && *q.MinSize == minSize &&
				q.MaxSize != nil && *q.MaxSize == maxSize
		})).Return(mockResults, nil)

		reqBody := SearchRequest{
			MinSize: &minSize,
			MaxSize: &maxSize,
			Limit:   100,
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/catalog/search", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		server.handleSearchCatalog(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockEngine.AssertExpectations(t)
	})

	t.Run("pagination with has_more", func(t *testing.T) {
		mockEngine := new(MockSearchEngine)
		mockEngine.On("IsAvailable").Return(true)
		server := &Server{searchEngine: mockEngine}

		mockResults := &catalog.SearchResults{
			Results: make([]*catalog.SearchResult, 10),
			Total:   50,
			Took:    20,
		}

		// Initialize results with dummy data
		for i := 0; i < 10; i++ {
			mockResults.Results[i] = &catalog.SearchResult{
				Backup: &catalog.BackupDocument{
					ID: "backup-" + string(rune(i)),
				},
			}
		}

		mockEngine.On("Search", mock.Anything, mock.Anything).Return(mockResults, nil)

		reqBody := SearchRequest{
			Limit:  10,
			Offset: 0,
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/catalog/search", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		server.handleSearchCatalog(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response SearchResponse
		json.Unmarshal(w.Body.Bytes(), &response)

		assert.Equal(t, int64(50), response.Total)
		assert.True(t, response.HasMore, "should have more results")
		require.NotNil(t, response.NextOffset)
		assert.Equal(t, 10, *response.NextOffset)

		mockEngine.AssertExpectations(t)
	})

	t.Run("invalid date format", func(t *testing.T) {
		server := &Server{}

		reqBody := SearchRequest{
			DateFrom: "invalid-date",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/catalog/search", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		server.handleSearchCatalog(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		server := &Server{}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/catalog/search",
			bytes.NewBufferString("invalid json"))
		c.Request.Header.Set("Content-Type", "application/json")

		server.handleSearchCatalog(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandleSearchCatalogSimple(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("natural language query", func(t *testing.T) {
		mockEngine := new(MockSearchEngine)
		mockEngine.On("IsAvailable").Return(true)
		server := &Server{searchEngine: mockEngine}

		// Mock ParseQueryString
		mockQuery := &catalog.SearchQuery{
			Text:          "customers",
			DatabaseTypes: []string{"postgres"},
			Status:        []string{"success"},
			Limit:         100,
		}
		mockEngine.On("ParseQueryString", "database:customers type:postgres status:success").
			Return(mockQuery, nil)

		// Mock Search
		mockResults := &catalog.SearchResults{
			Results: []*catalog.SearchResult{},
			Total:   0,
			Took:    10,
		}
		mockEngine.On("Search", mock.Anything, mockQuery).Return(mockResults, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet,
			"/api/v1/catalog/search?q=database:customers+type:postgres+status:success", nil)

		server.handleSearchCatalogSimple(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockEngine.AssertExpectations(t)
	})
}

func TestHandleSuggestCatalog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful autocomplete", func(t *testing.T) {
		mockEngine := new(MockSearchEngine)
		mockEngine.On("IsAvailable").Return(true)
		server := &Server{searchEngine: mockEngine}

		mockSuggestions := []string{"production", "prod-replica", "prod-archive"}
		mockEngine.On("Suggest", mock.Anything, "prod", "database_name.keyword", 10).
			Return(mockSuggestions, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet,
			"/api/v1/catalog/suggest?field=database_name&prefix=prod&limit=10", nil)

		server.handleSuggestCatalog(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response SuggestResponse
		json.Unmarshal(w.Body.Bytes(), &response)

		assert.Equal(t, "database_name", response.Field)
		assert.Equal(t, "prod", response.Prefix)
		assert.Equal(t, 3, response.Count)
		assert.Equal(t, mockSuggestions, response.Suggestions)

		mockEngine.AssertExpectations(t)
	})

	t.Run("missing field parameter", func(t *testing.T) {
		server := &Server{}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet,
			"/api/v1/catalog/suggest?prefix=prod", nil)

		server.handleSuggestCatalog(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing prefix parameter", func(t *testing.T) {
		server := &Server{}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet,
			"/api/v1/catalog/suggest?field=database_name", nil)

		server.handleSuggestCatalog(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid field", func(t *testing.T) {
		server := &Server{}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet,
			"/api/v1/catalog/suggest?field=invalid_field&prefix=test", nil)

		server.handleSuggestCatalog(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("default limit", func(t *testing.T) {
		mockEngine := new(MockSearchEngine)
		mockEngine.On("IsAvailable").Return(true)
		server := &Server{searchEngine: mockEngine}

		mockSuggestions := []string{"postgres", "postgresql"}
		mockEngine.On("Suggest", mock.Anything, "pos", "database_type", 10).
			Return(mockSuggestions, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet,
			"/api/v1/catalog/suggest?field=database_type&prefix=pos", nil)

		server.handleSuggestCatalog(c)

		assert.Equal(t, http.StatusOK, w.Code)
		mockEngine.AssertExpectations(t)
	})
}

func TestHandleGetCatalogStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful stats retrieval", func(t *testing.T) {
		mockEngine := new(MockSearchEngine)
		mockEngine.On("IsAvailable").Return(true)
		server := &Server{searchEngine: mockEngine}

		mockStats := &catalog.CatalogStats{
			TotalBackups:    1000,
			TotalSize:       1024 * 1024 * 1024 * 500, // 500GB
			AverageSize:     1024 * 1024 * 100,        // 100MB
			AverageDuration: 45.5,
			ByStatus: map[string]int64{
				"success": 950,
				"failed":  50,
			},
			ByType: map[string]int64{
				"full":        300,
				"incremental": 700,
			},
			ByDatabaseType: map[string]int64{
				"postgres": 600,
				"mysql":    400,
			},
			ByProvider: map[string]int64{
				"s3":    700,
				"azure": 300,
			},
		}

		mockEngine.On("GetStats", mock.Anything).Return(mockStats, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/catalog/stats", nil)

		server.handleGetCatalogStats(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response catalog.CatalogStats
		json.Unmarshal(w.Body.Bytes(), &response)

		assert.Equal(t, int64(1000), response.TotalBackups)
		assert.Equal(t, int64(950), response.ByStatus["success"])
		assert.Equal(t, int64(600), response.ByDatabaseType["postgres"])

		mockEngine.AssertExpectations(t)
	})
}

func TestHandleQueryExamples(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("get query examples", func(t *testing.T) {
		server := &Server{}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/catalog/query-examples", nil)

		server.handleQueryExamples(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response QueryExamplesResponse
		json.Unmarshal(w.Body.Bytes(), &response)

		assert.Greater(t, len(response.Examples), 0)
		assert.NotEmpty(t, response.Examples[0].Description)
		assert.NotEmpty(t, response.Examples[0].Query)
	})
}

func TestSearchRequestValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		request        SearchRequest
		expectedStatus int
	}{
		{
			name: "valid request with all fields",
			request: SearchRequest{
				Text:          "test",
				DatabaseNames: []string{"db1", "db2"},
				Limit:         50,
				SortBy:        "created_at",
				SortOrder:     "desc",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "empty request (should use defaults)",
			request: SearchRequest{
				Limit: 100,
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEngine := new(MockSearchEngine)
			mockEngine.On("IsAvailable").Return(true)
			server := &Server{searchEngine: mockEngine}

			mockResults := &catalog.SearchResults{
				Results: []*catalog.SearchResult{},
				Total:   0,
				Took:    5,
			}
			mockEngine.On("Search", mock.Anything, mock.Anything).Return(mockResults, nil)

			body, _ := json.Marshal(tt.request)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/catalog/search",
				bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			server.handleSearchCatalog(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
