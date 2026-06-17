package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sanskarpan/db-backup/internal/catalog"
)

// SearchRequest represents a catalog search request
type SearchRequest struct {
	// Text search query
	Text string `json:"text" form:"text"`

	// Filters
	DatabaseNames    []string          `json:"database_names" form:"database_names"`
	DatabaseTypes    []string          `json:"database_types" form:"database_types"`
	BackupTypes      []string          `json:"backup_types" form:"backup_types"`
	Status           []string          `json:"status" form:"status"`
	StorageProviders []string          `json:"storage_providers" form:"storage_providers"`
	Tags             map[string]string `json:"tags" form:"tags"`

	// Date range
	DateFrom string `json:"date_from" form:"date_from"` // RFC3339 format
	DateTo   string `json:"date_to" form:"date_to"`     // RFC3339 format

	// Size range (bytes)
	MinSize *int64 `json:"min_size" form:"min_size"`
	MaxSize *int64 `json:"max_size" form:"max_size"`

	// Duration range (seconds)
	MinDuration *float64 `json:"min_duration" form:"min_duration"`
	MaxDuration *float64 `json:"max_duration" form:"max_duration"`

	// Table/schema filters
	Tables  []string `json:"tables" form:"tables"`
	Schemas []string `json:"schemas" form:"schemas"`

	// Parent backup
	ParentBackupID string `json:"parent_backup_id" form:"parent_backup_id"`

	// Options
	IncludeExpired bool `json:"include_expired" form:"include_expired"`

	// Pagination
	Limit  int `json:"limit" form:"limit"`
	Offset int `json:"offset" form:"offset"`

	// Sorting
	SortBy    string `json:"sort_by" form:"sort_by"`       // created_at, size_bytes, duration
	SortOrder string `json:"sort_order" form:"sort_order"` // asc, desc
}

// SearchResponse represents a search response
type SearchResponse struct {
	Results    []*SearchResultItem `json:"results"`
	Total      int64               `json:"total"`
	Took       int                 `json:"took_ms"`
	Limit      int                 `json:"limit"`
	Offset     int                 `json:"offset"`
	HasMore    bool                `json:"has_more"`
	NextOffset *int                `json:"next_offset,omitempty"`
}

// SearchResultItem represents a single search result
type SearchResultItem struct {
	BackupID      string            `json:"backup_id"`
	DatabaseName  string            `json:"database_name"`
	DatabaseType  string            `json:"database_type"`
	BackupType    string            `json:"backup_type"`
	Status        string            `json:"status"`
	SizeBytes     int64             `json:"size_bytes"`
	Duration      float64           `json:"duration"`
	StoragePath   string            `json:"storage_path"`
	StorageProvider string          `json:"storage_provider"`
	CreatedAt     time.Time         `json:"created_at"`
	ExpiresAt     *time.Time        `json:"expires_at,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	Tables        []string          `json:"tables,omitempty"`
	Schemas       []string          `json:"schemas,omitempty"`
	Score         float64           `json:"score,omitempty"`
}

// handleSearchCatalog handles POST /api/v1/catalog/search
// @Summary Search backup catalog
// @Description Perform full-text search with advanced filtering on backup catalog
// @Tags catalog
// @Accept json
// @Produce json
// @Param request body SearchRequest true "Search request"
// @Success 200 {object} SearchResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/catalog/search [post]
func (s *Server) handleSearchCatalog(c *gin.Context) {
	var req SearchRequest

	// Parse and validate request before checking availability (returns 400 for bad input)
	if c.Request.Method == http.MethodPost {
		if err := c.ShouldBindJSON(&req); err != nil {
			s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
			return
		}
	} else {
		if err := c.ShouldBindQuery(&req); err != nil {
			s.respondError(c, http.StatusBadRequest, err, "Invalid query parameters")
			return
		}
	}

	// Build search query (validate dates before checking availability)
	query := &catalog.SearchQuery{
		Text:             req.Text,
		DatabaseNames:    req.DatabaseNames,
		DatabaseTypes:    req.DatabaseTypes,
		BackupTypes:      req.BackupTypes,
		Status:           req.Status,
		StorageProviders: req.StorageProviders,
		Tags:             req.Tags,
		Tables:           req.Tables,
		Schemas:          req.Schemas,
		ParentBackupID:   req.ParentBackupID,
		IncludeExpired:   req.IncludeExpired,
		Limit:            req.Limit,
		Offset:           req.Offset,
		SortBy:           req.SortBy,
		SortOrder:        req.SortOrder,
		MinSize:          req.MinSize,
		MaxSize:          req.MaxSize,
		MinDuration:      req.MinDuration,
		MaxDuration:      req.MaxDuration,
	}

	if req.DateFrom != "" {
		dateFrom, err := time.Parse(time.RFC3339, req.DateFrom)
		if err != nil {
			s.respondError(c, http.StatusBadRequest, err, "Invalid date_from format (use RFC3339)")
			return
		}
		query.DateFrom = &dateFrom
	}

	if req.DateTo != "" {
		dateTo, err := time.Parse(time.RFC3339, req.DateTo)
		if err != nil {
			s.respondError(c, http.StatusBadRequest, err, "Invalid date_to format (use RFC3339)")
			return
		}
		query.DateTo = &dateTo
	}

	// Check if search engine is available
	if s.searchEngine == nil || !s.searchEngine.IsAvailable() {
		s.respondError(c, http.StatusServiceUnavailable,
			fmt.Errorf("catalog search not available"),
			"Elasticsearch is not configured")
		return
	}

	// Execute search
	results, err := s.searchEngine.Search(c.Request.Context(), query)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Search failed")
		return
	}

	// Convert results to response format
	items := make([]*SearchResultItem, len(results.Results))
	for i, result := range results.Results {
		items[i] = &SearchResultItem{
			BackupID:        result.Backup.ID,
			DatabaseName:    result.Backup.DatabaseName,
			DatabaseType:    result.Backup.DatabaseType,
			BackupType:      result.Backup.BackupType,
			Status:          result.Backup.Status,
			SizeBytes:       result.Backup.SizeBytes,
			Duration:        result.Backup.Duration,
			StoragePath:     result.Backup.StoragePath,
			StorageProvider: result.Backup.StorageProvider,
			CreatedAt:       result.Backup.CreatedAt,
			ExpiresAt:       result.Backup.ExpiresAt,
			Tags:            result.Backup.Tags,
			Tables:          result.Backup.Tables,
			Schemas:         result.Backup.Schemas,
			Score:           result.Score,
		}
	}

	// Calculate pagination metadata
	hasMore := false
	var nextOffset *int
	if query.Offset+query.Limit < int(results.Total) {
		hasMore = true
		next := query.Offset + query.Limit
		nextOffset = &next
	}

	response := &SearchResponse{
		Results:    items,
		Total:      results.Total,
		Took:       results.Took,
		Limit:      query.Limit,
		Offset:     query.Offset,
		HasMore:    hasMore,
		NextOffset: nextOffset,
	}

	c.JSON(http.StatusOK, response)
}

// handleSearchCatalogSimple handles GET /api/v1/catalog/search
// @Summary Simple search (GET)
// @Description Perform simple search using query parameters or natural language query
// @Tags catalog
// @Accept json
// @Produce json
// @Param q query string false "Natural language search query (e.g., 'database:mydb type:mysql status:success')"
// @Param text query string false "Free text search"
// @Param database_names query []string false "Filter by database names"
// @Param database_types query []string false "Filter by database types"
// @Param status query []string false "Filter by status"
// @Param limit query int false "Maximum results (default: 100)"
// @Param offset query int false "Result offset (default: 0)"
// @Success 200 {object} SearchResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/catalog/search [get]
func (s *Server) handleSearchCatalogSimple(c *gin.Context) {
	// Check if search engine is available
	if s.searchEngine == nil || !s.searchEngine.IsAvailable() {
		s.respondError(c, http.StatusServiceUnavailable,
			fmt.Errorf("catalog search not available"),
			"Elasticsearch is not configured")
		return
	}

	// Check if using natural language query
	if q := c.Query("q"); q != "" {
		// Parse natural language query
		query, err := s.searchEngine.ParseQueryString(q)
		if err != nil {
			s.respondError(c, http.StatusBadRequest, err, "Failed to parse query")
			return
		}

		// Execute search
		results, err := s.searchEngine.Search(c.Request.Context(), query)
		if err != nil {
			s.respondError(c, http.StatusInternalServerError, err, "Search failed")
			return
		}

		// Convert and return results
		s.convertAndReturnSearchResults(c, results, query)
		return
	}

	// Otherwise use handleSearchCatalog with query params
	s.handleSearchCatalog(c)
}

// handleSuggestCatalog handles GET /api/v1/catalog/suggest
// @Summary Get autocomplete suggestions
// @Description Get autocomplete suggestions for search fields
// @Tags catalog
// @Accept json
// @Produce json
// @Param field query string true "Field to get suggestions for (database_name, database_type, status, storage_provider)"
// @Param prefix query string true "Prefix to match"
// @Param limit query int false "Maximum suggestions (default: 10)"
// @Success 200 {object} SuggestResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/catalog/suggest [get]
func (s *Server) handleSuggestCatalog(c *gin.Context) {
	field := c.Query("field")
	prefix := c.Query("prefix")
	limitStr := c.DefaultQuery("limit", "10")

	// Validate parameters before checking availability (returns 400 for bad input)
	if field == "" {
		s.respondError(c, http.StatusBadRequest, nil, "field parameter is required")
		return
	}

	if prefix == "" {
		s.respondError(c, http.StatusBadRequest, nil, "prefix parameter is required")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	// Map user-friendly field names to Elasticsearch field names
	fieldMap := map[string]string{
		"database_name":    "database_name.keyword",
		"database_type":    "database_type",
		"status":           "status",
		"storage_provider": "storage_provider",
		"backup_type":      "backup_type",
	}

	esField, ok := fieldMap[field]
	if !ok {
		s.respondError(c, http.StatusBadRequest, nil,
			"Invalid field. Supported: database_name, database_type, status, storage_provider, backup_type")
		return
	}

	// Check if search engine is available
	if s.searchEngine == nil || !s.searchEngine.IsAvailable() {
		s.respondError(c, http.StatusServiceUnavailable,
			fmt.Errorf("catalog search not available"),
			"Elasticsearch is not configured")
		return
	}

	// Get suggestions
	suggestions, err := s.searchEngine.Suggest(c.Request.Context(), prefix, esField, limit)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to get suggestions")
		return
	}

	c.JSON(http.StatusOK, &SuggestResponse{
		Field:       field,
		Prefix:      prefix,
		Suggestions: suggestions,
		Count:       len(suggestions),
	})
}

// handleGetCatalogStats handles GET /api/v1/catalog/stats
// @Summary Get catalog statistics
// @Description Get comprehensive statistics about the backup catalog
// @Tags catalog
// @Accept json
// @Produce json
// @Success 200 {object} catalog.CatalogStats
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/catalog/stats [get]
func (s *Server) handleGetCatalogStats(c *gin.Context) {
	// Check if search engine is available
	if s.searchEngine == nil || !s.searchEngine.IsAvailable() {
		s.respondError(c, http.StatusServiceUnavailable,
			fmt.Errorf("catalog search not available"),
			"Elasticsearch is not configured")
		return
	}

	stats, err := s.searchEngine.GetStats(c.Request.Context())
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to get statistics")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// handleQueryExamples handles GET /api/v1/catalog/query-examples
// @Summary Get query examples
// @Description Get example search queries for documentation
// @Tags catalog
// @Accept json
// @Produce json
// @Success 200 {object} QueryExamplesResponse
// @Router /api/v1/catalog/query-examples [get]
func (s *Server) handleQueryExamples(c *gin.Context) {
	examples := &QueryExamplesResponse{
		Examples: []QueryExample{
			{
				Description: "Search for MySQL backups",
				Query:       "type:mysql",
				Filters: map[string]interface{}{
					"database_types": []string{"mysql"},
				},
			},
			{
				Description: "Search for failed backups from last week",
				Query:       "status:failed date:2024-01-01..2024-01-07",
				Filters: map[string]interface{}{
					"status":    []string{"failed"},
					"date_from": "2024-01-01T00:00:00Z",
					"date_to":   "2024-01-07T23:59:59Z",
				},
			},
			{
				Description: "Find large backups (>1GB)",
				Query:       "size:>1073741824",
				Filters: map[string]interface{}{
					"min_size": 1073741824,
				},
			},
			{
				Description: "Search for specific database with table name",
				Query:       "database:production table:users",
				Filters: map[string]interface{}{
					"database_names": []string{"production"},
					"tables":         []string{"users"},
				},
			},
			{
				Description: "Find backups in S3 storage",
				Query:       "provider:s3",
				Filters: map[string]interface{}{
					"storage_providers": []string{"s3"},
				},
			},
			{
				Description: "Complex search with text",
				Query:       "customers status:success type:postgres",
				Filters: map[string]interface{}{
					"text":           "customers",
					"status":         []string{"success"},
					"database_types": []string{"postgres"},
				},
			},
		},
	}

	c.JSON(http.StatusOK, examples)
}

// Helper method to convert search results
func (s *Server) convertAndReturnSearchResults(c *gin.Context, results *catalog.SearchResults, query *catalog.SearchQuery) {
	items := make([]*SearchResultItem, len(results.Results))
	for i, result := range results.Results {
		items[i] = &SearchResultItem{
			BackupID:        result.Backup.ID,
			DatabaseName:    result.Backup.DatabaseName,
			DatabaseType:    result.Backup.DatabaseType,
			BackupType:      result.Backup.BackupType,
			Status:          result.Backup.Status,
			SizeBytes:       result.Backup.SizeBytes,
			Duration:        result.Backup.Duration,
			StoragePath:     result.Backup.StoragePath,
			StorageProvider: result.Backup.StorageProvider,
			CreatedAt:       result.Backup.CreatedAt,
			ExpiresAt:       result.Backup.ExpiresAt,
			Tags:            result.Backup.Tags,
			Tables:          result.Backup.Tables,
			Schemas:         result.Backup.Schemas,
			Score:           result.Score,
		}
	}

	hasMore := false
	var nextOffset *int
	if query.Offset+query.Limit < int(results.Total) {
		hasMore = true
		next := query.Offset + query.Limit
		nextOffset = &next
	}

	response := &SearchResponse{
		Results:    items,
		Total:      results.Total,
		Took:       results.Took,
		Limit:      query.Limit,
		Offset:     query.Offset,
		HasMore:    hasMore,
		NextOffset: nextOffset,
	}

	c.JSON(http.StatusOK, response)
}

// SuggestResponse represents autocomplete suggestion response
type SuggestResponse struct {
	Field       string   `json:"field"`
	Prefix      string   `json:"prefix"`
	Suggestions []string `json:"suggestions"`
	Count       int      `json:"count"`
}

// QueryExamplesResponse contains example queries
type QueryExamplesResponse struct {
	Examples []QueryExample `json:"examples"`
}

// QueryExample represents a single query example
type QueryExample struct {
	Description string                 `json:"description"`
	Query       string                 `json:"query"`
	Filters     map[string]interface{} `json:"filters"`
}
