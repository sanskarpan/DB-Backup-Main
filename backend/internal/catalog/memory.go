package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// memoryStoreFileName is the single JSON file that holds all indexed backup
// documents when the in-memory engine is configured with a persistence dir.
const memoryStoreFileName = "catalog.json"

// InMemorySearchEngine satisfies the same interface the API handlers use, so it
// can replace the Elasticsearch-backed SearchEngine transparently.
var _ SearchEngineInterface = (*InMemorySearchEngine)(nil)

// InMemorySearchEngine is a dependency-free implementation of
// SearchEngineInterface. It keeps indexed backup documents in a map guarded by
// a mutex, so /catalog/* endpoints return real results even when Elasticsearch
// is not configured. When a persistence directory is supplied, the index is
// written to a JSON file (atomically) after every mutation and reloaded on
// construction, so it survives a restart.
type InMemorySearchEngine struct {
	entries map[string]*BackupDocument
	dir     string
	mu      sync.RWMutex
}

// NewInMemorySearchEngine creates an in-memory search engine that keeps its
// index only in memory (no persistence). It is always available.
func NewInMemorySearchEngine() *InMemorySearchEngine {
	return &InMemorySearchEngine{
		entries: make(map[string]*BackupDocument),
	}
}

// NewPersistentInMemorySearchEngine creates an in-memory search engine backed by
// a JSON file under dir. Previously indexed documents are loaded on construction
// so the catalog survives a restart. The directory is created if it does not
// exist.
func NewPersistentInMemorySearchEngine(dir string) (*InMemorySearchEngine, error) {
	if dir == "" {
		return nil, fmt.Errorf("catalog: persistence directory is required")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("catalog: create store dir: %w", err)
	}

	e := &InMemorySearchEngine{
		entries: make(map[string]*BackupDocument),
		dir:     dir,
	}
	if err := e.load(); err != nil {
		return nil, err
	}
	return e, nil
}

// IsAvailable always returns true for the in-memory engine.
func (e *InMemorySearchEngine) IsAvailable() bool {
	return e != nil
}

// path returns the full path to the persistence file.
func (e *InMemorySearchEngine) path() string {
	return filepath.Join(e.dir, memoryStoreFileName)
}

// load reads all documents from disk. A missing file is not an error. Callers
// must not hold e.mu.
func (e *InMemorySearchEngine) load() error {
	data, err := os.ReadFile(e.path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("catalog: read store: %w", err)
	}
	if len(data) == 0 {
		return nil
	}

	var docs []*BackupDocument
	if err := json.Unmarshal(data, &docs); err != nil {
		return fmt.Errorf("catalog: parse store: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	for _, d := range docs {
		if d != nil && d.ID != "" {
			e.entries[d.ID] = d
		}
	}
	return nil
}

// persist writes all documents to disk atomically (temp file + rename). It is a
// no-op when no persistence directory is configured. Callers must hold e.mu.
func (e *InMemorySearchEngine) persist() error {
	if e.dir == "" {
		return nil
	}

	docs := make([]*BackupDocument, 0, len(e.entries))
	for _, d := range e.entries {
		docs = append(docs, d)
	}

	data, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return fmt.Errorf("catalog: marshal store: %w", err)
	}

	tmp, err := os.CreateTemp(e.dir, memoryStoreFileName+".tmp-*")
	if err != nil {
		return fmt.Errorf("catalog: create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("catalog: write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("catalog: sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("catalog: close temp: %w", err)
	}
	if err := os.Rename(tmpName, e.path()); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("catalog: rename store: %w", err)
	}
	return nil
}

// cloneDocument returns a deep-enough copy of a document so callers cannot
// mutate stored state through the returned pointer.
func cloneDocument(doc *BackupDocument) *BackupDocument {
	if doc == nil {
		return nil
	}
	out := *doc
	if doc.Tags != nil {
		out.Tags = make(map[string]string, len(doc.Tags))
		for k, v := range doc.Tags {
			out.Tags[k] = v
		}
	}
	if doc.Tables != nil {
		out.Tables = append([]string(nil), doc.Tables...)
	}
	if doc.Schemas != nil {
		out.Schemas = append([]string(nil), doc.Schemas...)
	}
	if doc.ChildBackupIDs != nil {
		out.ChildBackupIDs = append([]string(nil), doc.ChildBackupIDs...)
	}
	return &out
}

// IndexBackup adds or replaces a backup document in the in-memory catalog. It
// mirrors CatalogIndexer.IndexBackup so callers can index into either backend.
func (e *InMemorySearchEngine) IndexBackup(_ context.Context, backup *BackupDocument) error {
	if backup == nil {
		return fmt.Errorf("backup document cannot be nil")
	}
	if backup.ID == "" {
		return fmt.Errorf("backup ID cannot be empty")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	stored := cloneDocument(backup)
	e.entries[backup.ID] = stored
	if err := e.persist(); err != nil {
		delete(e.entries, backup.ID)
		return err
	}
	return nil
}

// IndexBackups adds or replaces multiple backup documents in a single write.
func (e *InMemorySearchEngine) IndexBackups(_ context.Context, backups []*BackupDocument) error {
	if len(backups) == 0 {
		return nil
	}
	for i, backup := range backups {
		if backup == nil {
			return fmt.Errorf("backup at index %d is nil", i)
		}
		if backup.ID == "" {
			return fmt.Errorf("backup at index %d has empty ID", i)
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	added := make([]string, 0, len(backups))
	for _, backup := range backups {
		e.entries[backup.ID] = cloneDocument(backup)
		added = append(added, backup.ID)
	}
	if err := e.persist(); err != nil {
		for _, id := range added {
			delete(e.entries, id)
		}
		return err
	}
	return nil
}

// GetBackup retrieves a backup document by ID.
func (e *InMemorySearchEngine) GetBackup(_ context.Context, backupID string) (*BackupDocument, error) {
	if backupID == "" {
		return nil, fmt.Errorf("backup ID cannot be empty")
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	doc, ok := e.entries[backupID]
	if !ok {
		return nil, fmt.Errorf("backup %q not found", backupID)
	}
	return cloneDocument(doc), nil
}

// DeleteBackup removes a backup document by ID. Deleting a missing document is
// not an error.
func (e *InMemorySearchEngine) DeleteBackup(_ context.Context, backupID string) error {
	if backupID == "" {
		return fmt.Errorf("backup ID cannot be empty")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	existing, ok := e.entries[backupID]
	if !ok {
		return nil
	}
	delete(e.entries, backupID)
	if err := e.persist(); err != nil {
		e.entries[backupID] = existing
		return err
	}
	return nil
}

// ParseQueryString parses a simple query string into a SearchQuery. It shares
// the backend-independent parser with the Elasticsearch engine.
func (e *InMemorySearchEngine) ParseQueryString(queryString string) (*SearchQuery, error) {
	return parseQueryString(queryString)
}

// Search performs an in-memory search that mirrors the filters supported by the
// Elasticsearch engine: free text, keyword filters, tags, and size/date ranges.
func (e *InMemorySearchEngine) Search(_ context.Context, query *SearchQuery) (*SearchResults, error) {
	start := time.Now()
	if query == nil {
		query = &SearchQuery{}
	}
	if query.Limit <= 0 {
		query.Limit = 100
	}
	if query.SortBy == "" {
		query.SortBy = "created_at"
	}
	if query.SortOrder == "" {
		query.SortOrder = "desc"
	}

	e.mu.RLock()
	matched := make([]*SearchResult, 0)
	for _, doc := range e.entries {
		if documentMatches(doc, query) {
			matched = append(matched, &SearchResult{
				Backup: cloneDocument(doc),
				Score:  1.0,
			})
		}
	}
	e.mu.RUnlock()

	sortResults(matched, query.SortBy, query.SortOrder)

	total := int64(len(matched))
	page := paginate(matched, query.Offset, query.Limit)

	return &SearchResults{
		Results: page,
		Total:   total,
		Took:    int(time.Since(start).Milliseconds()),
	}, nil
}

// paginate returns the slice of results for the requested offset and limit.
func paginate(results []*SearchResult, offset, limit int) []*SearchResult {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(results) {
		return []*SearchResult{}
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}
	return results[offset:end]
}

// documentMatches reports whether a document satisfies every filter in query.
func documentMatches(doc *BackupDocument, q *SearchQuery) bool {
	return matchesText(doc, q.Text) &&
		matchesKeyword(doc.DatabaseName, q.DatabaseNames) &&
		matchesKeyword(doc.DatabaseType, q.DatabaseTypes) &&
		matchesKeyword(doc.BackupType, q.BackupTypes) &&
		matchesKeyword(doc.Status, q.Status) &&
		matchesKeyword(doc.StorageProvider, q.StorageProviders) &&
		matchesAny(doc.Tables, q.Tables) &&
		matchesAny(doc.Schemas, q.Schemas) &&
		matchesTags(doc.Tags, q.Tags) &&
		matchesDateRange(doc.CreatedAt, q.DateFrom, q.DateTo) &&
		matchesInt64Range(doc.SizeBytes, q.MinSize, q.MaxSize) &&
		matchesFloatRange(doc.Duration, q.MinDuration, q.MaxDuration) &&
		matchesParent(doc.ParentBackupID, q.ParentBackupID) &&
		matchesExpiry(doc.ExpiresAt, q.IncludeExpired)
}

// matchesText reports whether the free-text term is empty or appears (case
// insensitively) in any of the searchable text fields.
func matchesText(doc *BackupDocument, text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return true
	}
	needle := strings.ToLower(text)

	fields := []string{doc.DatabaseName, doc.StoragePath, doc.ErrorMessage}
	fields = append(fields, doc.Tables...)
	fields = append(fields, doc.Schemas...)
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), needle) {
			return true
		}
	}
	return false
}

// matchesKeyword reports whether value equals one of the allowed values. An
// empty allow-list matches everything.
func matchesKeyword(value string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}

// matchesAny reports whether the document slice contains at least one of the
// wanted values. An empty want-list matches everything.
func matchesAny(values, want []string) bool {
	if len(want) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	for _, w := range want {
		if _, ok := set[w]; ok {
			return true
		}
	}
	return false
}

// matchesTags reports whether the document carries every requested tag with the
// requested value.
func matchesTags(docTags, want map[string]string) bool {
	if len(want) == 0 {
		return true
	}
	for k, v := range want {
		if docTags[k] != v {
			return false
		}
	}
	return true
}

// matchesDateRange reports whether created falls within [from, to] (inclusive).
// Nil bounds are treated as open.
func matchesDateRange(created time.Time, from, to *time.Time) bool {
	if from != nil && created.Before(*from) {
		return false
	}
	if to != nil && created.After(*to) {
		return false
	}
	return true
}

// matchesInt64Range reports whether value falls within [min, max] (inclusive).
func matchesInt64Range(value int64, minValue, maxValue *int64) bool {
	if minValue != nil && value < *minValue {
		return false
	}
	if maxValue != nil && value > *maxValue {
		return false
	}
	return true
}

// matchesFloatRange reports whether value falls within [min, max] (inclusive).
func matchesFloatRange(value float64, minValue, maxValue *float64) bool {
	if minValue != nil && value < *minValue {
		return false
	}
	if maxValue != nil && value > *maxValue {
		return false
	}
	return true
}

// matchesParent reports whether the document's parent matches the requested
// parent backup ID. An empty want matches everything.
func matchesParent(parentID, want string) bool {
	return want == "" || parentID == want
}

// matchesExpiry reports whether the document should be kept given the expiry
// policy. Expired documents are excluded unless includeExpired is set.
func matchesExpiry(expiresAt *time.Time, includeExpired bool) bool {
	if includeExpired || expiresAt == nil {
		return true
	}
	return !expiresAt.Before(time.Now())
}

// sortResults sorts results in place by the requested field and order.
func sortResults(results []*SearchResult, sortBy, order string) {
	asc := strings.EqualFold(order, "asc")
	sort.SliceStable(results, func(i, j int) bool {
		if asc {
			return lessByField(results[i].Backup, results[j].Backup, sortBy)
		}
		return lessByField(results[j].Backup, results[i].Backup, sortBy)
	})
}

// lessByField reports whether a should sort before b for the given field in
// ascending order.
func lessByField(a, b *BackupDocument, field string) bool {
	switch field {
	case "size_bytes":
		return a.SizeBytes < b.SizeBytes
	case "duration":
		return a.Duration < b.Duration
	default: // created_at
		return a.CreatedAt.Before(b.CreatedAt)
	}
}

// Suggest returns up to limit unique values of field that start with prefix.
func (e *InMemorySearchEngine) Suggest(_ context.Context, prefix, field string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	field = strings.TrimSuffix(field, ".keyword")
	lowerPrefix := strings.ToLower(prefix)

	e.mu.RLock()
	seen := make(map[string]struct{})
	suggestions := make([]string, 0)
	for _, doc := range e.entries {
		for _, value := range fieldValues(doc, field) {
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			if !strings.HasPrefix(strings.ToLower(value), lowerPrefix) {
				continue
			}
			seen[value] = struct{}{}
			suggestions = append(suggestions, value)
		}
	}
	e.mu.RUnlock()

	sort.Strings(suggestions)
	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}
	return suggestions, nil
}

// fieldValues returns the value(s) of a suggestible field for a document.
func fieldValues(doc *BackupDocument, field string) []string {
	switch field {
	case "database_name":
		return []string{doc.DatabaseName}
	case "database_type":
		return []string{doc.DatabaseType}
	case "backup_type":
		return []string{doc.BackupType}
	case "status":
		return []string{doc.Status}
	case "storage_provider":
		return []string{doc.StorageProvider}
	case "storage_path":
		return []string{doc.StoragePath}
	case "tables":
		return doc.Tables
	case "schemas":
		return doc.Schemas
	default:
		return nil
	}
}

// GetStats returns real aggregate statistics over every indexed document.
func (e *InMemorySearchEngine) GetStats(_ context.Context) (*CatalogStats, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := &CatalogStats{
		ByStatus:       make(map[string]int64),
		ByType:         make(map[string]int64),
		ByDatabaseType: make(map[string]int64),
		ByProvider:     make(map[string]int64),
	}

	var totalDuration float64
	for _, doc := range e.entries {
		stats.TotalBackups++
		stats.TotalSize += doc.SizeBytes
		totalDuration += doc.Duration
		incrementCount(stats.ByStatus, doc.Status)
		incrementCount(stats.ByType, doc.BackupType)
		incrementCount(stats.ByDatabaseType, doc.DatabaseType)
		incrementCount(stats.ByProvider, doc.StorageProvider)
	}

	if stats.TotalBackups > 0 {
		stats.AverageSize = stats.TotalSize / stats.TotalBackups
		stats.AverageDuration = totalDuration / float64(stats.TotalBackups)
	}
	return stats, nil
}

// incrementCount increments the count for key, ignoring empty keys.
func incrementCount(m map[string]int64, key string) {
	if key == "" {
		return
	}
	m[key]++
}
