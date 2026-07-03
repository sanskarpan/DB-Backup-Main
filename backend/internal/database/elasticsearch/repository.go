package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/go-elasticsearch/v8"
)

// RepositoryManager handles Elasticsearch snapshot repositories.
type RepositoryManager struct {
	client *elasticsearch.Client
}

// NewRepositoryManager creates a new repository manager.
func NewRepositoryManager(client *elasticsearch.Client) *RepositoryManager {
	return &RepositoryManager{client: client}
}

// CreateRepository creates a snapshot repository.
func (r *RepositoryManager) CreateRepository(ctx context.Context, name, location string) error {
	// Check if repository exists
	exists, err := r.RepositoryExists(ctx, name)
	if err != nil {
		return err
	}

	if exists {
		return nil // Repository already exists
	}

	// Create repository
	body := map[string]interface{}{
		"type": "fs",
		"settings": map[string]interface{}{
			"location": location,
			"compress": true,
		},
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal repository body: %w", err)
	}

	res, err := r.client.Snapshot.CreateRepository(
		name,
		bytes.NewReader(bodyJSON),
	)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("repository creation failed: %s, body: %s", res.Status(), string(body))
	}

	return nil
}

// DeleteRepository deletes a snapshot repository.
func (r *RepositoryManager) DeleteRepository(ctx context.Context, name string) error {
	res, err := r.client.Snapshot.DeleteRepository([]string{name})
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to delete repository: %s", res.Status())
	}

	return nil
}

// RepositoryExists checks if a repository exists.
func (r *RepositoryManager) RepositoryExists(ctx context.Context, name string) (bool, error) {
	res, err := r.client.Snapshot.GetRepository(
		r.client.Snapshot.GetRepository.WithRepository(name),
	)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	return !res.IsError(), nil
}

// ListRepositories lists all snapshot repositories.
func (r *RepositoryManager) ListRepositories(ctx context.Context) ([]string, error) {
	res, err := r.client.Snapshot.GetRepository()
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var repos map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&repos); err != nil {
		return nil, err
	}

	var repoNames []string
	for name := range repos {
		repoNames = append(repoNames, name)
	}

	return repoNames, nil
}
