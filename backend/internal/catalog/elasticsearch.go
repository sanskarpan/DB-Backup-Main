// Package catalog provides advanced search and cataloging capabilities for backups
// using Elasticsearch/OpenSearch for full-text search and metadata indexing.
package catalog

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// ElasticsearchClient wraps the Elasticsearch client with custom functionality.
type ElasticsearchClient struct {
	client *elasticsearch.Client
	config *ElasticsearchConfig
}

// ElasticsearchConfig holds configuration for Elasticsearch connection.
type ElasticsearchConfig struct {
	// Addresses is the list of Elasticsearch nodes
	Addresses []string

	// Username for authentication
	Username string

	// Password for authentication
	Password string

	// APIKey for authentication (alternative to username/password)
	APIKey string

	// CloudID for Elastic Cloud deployments
	CloudID string

	// CACert is the path to CA certificate for TLS verification
	CACert string

	// InsecureSkipVerify skips TLS certificate verification (not recommended)
	InsecureSkipVerify bool

	// MaxRetries is the maximum number of retries for failed requests
	MaxRetries int

	// Timeout for requests
	Timeout time.Duration

	// IndexPrefix is the prefix for all backup-related indices
	IndexPrefix string
}

// DefaultElasticsearchConfig returns a default configuration.
func DefaultElasticsearchConfig() *ElasticsearchConfig {
	return &ElasticsearchConfig{
		Addresses:   []string{"http://localhost:9200"},
		MaxRetries:  3,
		Timeout:     30 * time.Second,
		IndexPrefix: "db-backup",
	}
}

// NewElasticsearchClient creates a new Elasticsearch client.
func NewElasticsearchClient(config *ElasticsearchConfig) (*ElasticsearchClient, error) {
	if config == nil {
		config = DefaultElasticsearchConfig()
	}

	cfg := elasticsearch.Config{
		Addresses: config.Addresses,
		Username:  config.Username,
		Password:  config.Password,
		APIKey:    config.APIKey,
		CloudID:   config.CloudID,
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: config.Timeout,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.InsecureSkipVerify,
			},
		},
		MaxRetries:    config.MaxRetries,
		RetryOnStatus: []int{502, 503, 504, 429},
		RetryBackoff: func(i int) time.Duration {
			// Exponential backoff: 100ms, 200ms, 400ms, ...
			return time.Duration(100*i*i) * time.Millisecond
		},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	return &ElasticsearchClient{
		client: client,
		config: config,
	}, nil
}

// Ping checks if Elasticsearch is reachable.
func (es *ElasticsearchClient) Ping(ctx context.Context) error {
	res, err := es.client.Ping(
		es.client.Ping.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch returned error: %s", res.Status())
	}

	return nil
}

// CreateIndex creates a new index with the given mapping.
func (es *ElasticsearchClient) CreateIndex(ctx context.Context, indexName string, mapping map[string]interface{}) error {
	// Add prefix to index name
	fullIndexName := es.getIndexName(indexName)

	// Check if index exists
	exists, err := es.IndexExists(ctx, indexName)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("index %s already exists", fullIndexName)
	}

	// Create index with mapping
	body := map[string]interface{}{
		"mappings": mapping,
		"settings": map[string]interface{}{
			"number_of_shards":   3,
			"number_of_replicas": 1,
			"refresh_interval":   "1s",
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal index body: %w", err)
	}

	req := esapi.IndicesCreateRequest{
		Index: fullIndexName,
		Body:  bytes.NewReader(bodyBytes),
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("elasticsearch returned error: %s - %s", res.Status(), string(bodyBytes))
	}

	return nil
}

// IndexExists checks if an index exists.
func (es *ElasticsearchClient) IndexExists(ctx context.Context, indexName string) (bool, error) {
	fullIndexName := es.getIndexName(indexName)

	req := esapi.IndicesExistsRequest{
		Index: []string{fullIndexName},
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return false, fmt.Errorf("failed to check index existence: %w", err)
	}
	defer res.Body.Close()

	return res.StatusCode == 200, nil
}

// DeleteIndex deletes an index.
func (es *ElasticsearchClient) DeleteIndex(ctx context.Context, indexName string) error {
	fullIndexName := es.getIndexName(indexName)

	req := esapi.IndicesDeleteRequest{
		Index: []string{fullIndexName},
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("elasticsearch returned error: %s - %s", res.Status(), string(bodyBytes))
	}

	return nil
}

// IndexDocument indexes a document.
func (es *ElasticsearchClient) IndexDocument(ctx context.Context, indexName, documentID string, document interface{}) error {
	fullIndexName := es.getIndexName(indexName)

	bodyBytes, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	req := esapi.IndexRequest{
		Index:      fullIndexName,
		DocumentID: documentID,
		Body:       bytes.NewReader(bodyBytes),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("elasticsearch returned error: %s - %s", res.Status(), string(bodyBytes))
	}

	return nil
}

// BulkIndex indexes multiple documents in a single request.
func (es *ElasticsearchClient) BulkIndex(ctx context.Context, indexName string, documents []BulkDocument) error {
	if len(documents) == 0 {
		return nil
	}

	fullIndexName := es.getIndexName(indexName)

	var buf bytes.Buffer
	for _, doc := range documents {
		// Index action
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": fullIndexName,
				"_id":    doc.ID,
			},
		}
		actionBytes, _ := json.Marshal(action)
		buf.Write(actionBytes)
		buf.WriteByte('\n')

		// Document body
		docBytes, err := json.Marshal(doc.Document)
		if err != nil {
			return fmt.Errorf("failed to marshal document %s: %w", doc.ID, err)
		}
		buf.Write(docBytes)
		buf.WriteByte('\n')
	}

	req := esapi.BulkRequest{
		Body:    &buf,
		Refresh: "true",
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to bulk index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("elasticsearch returned error: %s - %s", res.Status(), string(bodyBytes))
	}

	// Parse response to check for individual failures
	var response BulkResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to parse bulk response: %w", err)
	}

	if response.Errors {
		var failedIDs []string
		for _, item := range response.Items {
			if item.Index.Error.Type != "" {
				failedIDs = append(failedIDs, item.Index.ID)
			}
		}
		return fmt.Errorf("bulk indexing failed for documents: %v", failedIDs)
	}

	return nil
}

// GetDocument retrieves a document by ID.
func (es *ElasticsearchClient) GetDocument(ctx context.Context, indexName, documentID string) (map[string]interface{}, error) {
	fullIndexName := es.getIndexName(indexName)

	req := esapi.GetRequest{
		Index:      fullIndexName,
		DocumentID: documentID,
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == 404 {
			return nil, fmt.Errorf("document not found")
		}
		bodyBytes, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("elasticsearch returned error: %s - %s", res.Status(), string(bodyBytes))
	}

	var response GetResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Source, nil
}

// DeleteDocument deletes a document by ID.
func (es *ElasticsearchClient) DeleteDocument(ctx context.Context, indexName, documentID string) error {
	fullIndexName := es.getIndexName(indexName)

	req := esapi.DeleteRequest{
		Index:      fullIndexName,
		DocumentID: documentID,
		Refresh:    "true",
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 404 {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("elasticsearch returned error: %s - %s", res.Status(), string(bodyBytes))
	}

	return nil
}

// Search performs a search query.
func (es *ElasticsearchClient) Search(ctx context.Context, indexName string, query map[string]interface{}) (*SearchResponse, error) {
	fullIndexName := es.getIndexName(indexName)

	bodyBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	req := esapi.SearchRequest{
		Index: []string{fullIndexName},
		Body:  bytes.NewReader(bodyBytes),
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("elasticsearch returned error: %s - %s", res.Status(), string(bodyBytes))
	}

	var response SearchResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// Count returns the number of documents matching a query.
func (es *ElasticsearchClient) Count(ctx context.Context, indexName string, query map[string]interface{}) (int64, error) {
	fullIndexName := es.getIndexName(indexName)

	bodyBytes, err := json.Marshal(map[string]interface{}{"query": query})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal query: %w", err)
	}

	req := esapi.CountRequest{
		Index: []string{fullIndexName},
		Body:  bytes.NewReader(bodyBytes),
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return 0, fmt.Errorf("failed to count: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return 0, fmt.Errorf("elasticsearch returned error: %s - %s", res.Status(), string(bodyBytes))
	}

	var response CountResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Count, nil
}

// Refresh refreshes the index to make recent changes visible.
func (es *ElasticsearchClient) Refresh(ctx context.Context, indexName string) error {
	fullIndexName := es.getIndexName(indexName)

	req := esapi.IndicesRefreshRequest{
		Index: []string{fullIndexName},
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to refresh index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("elasticsearch returned error: %s - %s", res.Status(), string(bodyBytes))
	}

	return nil
}

// getIndexName returns the full index name with prefix.
func (es *ElasticsearchClient) getIndexName(indexName string) string {
	if es.config.IndexPrefix == "" {
		return indexName
	}
	return fmt.Sprintf("%s-%s", es.config.IndexPrefix, indexName)
}

// BulkDocument represents a document to be bulk indexed.
type BulkDocument struct {
	ID       string
	Document interface{}
}

// BulkResponse represents the response from a bulk request.
type BulkResponse struct {
	Errors bool `json:"errors"`
	Items  []struct {
		Index struct {
			ID     string `json:"_id"`
			Status int    `json:"status"`
			Error  struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			} `json:"error"`
		} `json:"index"`
	} `json:"items"`
}

// GetResponse represents the response from a get request.
type GetResponse struct {
	Found  bool                   `json:"found"`
	Source map[string]interface{} `json:"_source"`
}

// SearchResponse represents the response from a search request.
type SearchResponse struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Hits     struct {
		Total struct {
			Value    int64  `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		MaxScore *float64 `json:"max_score"`
		Hits     []struct {
			Index  string                 `json:"_index"`
			ID     string                 `json:"_id"`
			Score  *float64               `json:"_score"`
			Source map[string]interface{} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
	Aggregations map[string]interface{} `json:"aggregations,omitempty"`
}

// CountResponse represents the response from a count request.
type CountResponse struct {
	Count int64 `json:"count"`
}

// GetHits returns the search hits as a slice of documents.
func (sr *SearchResponse) GetHits() []map[string]interface{} {
	results := make([]map[string]interface{}, len(sr.Hits.Hits))
	for i, hit := range sr.Hits.Hits {
		results[i] = hit.Source
		results[i]["_id"] = hit.ID
		if hit.Score != nil {
			results[i]["_score"] = *hit.Score
		}
	}
	return results
}

// GetTotal returns the total number of matching documents.
func (sr *SearchResponse) GetTotal() int64 {
	return sr.Hits.Total.Value
}

// IsEmpty returns true if no results were found.
func (sr *SearchResponse) IsEmpty() bool {
	return len(sr.Hits.Hits) == 0
}
