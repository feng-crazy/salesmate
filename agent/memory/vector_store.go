package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type VectorStoreType string

const (
	VectorStoreMemory   VectorStoreType = "memory"
	VectorStoreChroma   VectorStoreType = "chroma"
	VectorStorePinecone VectorStoreType = "pinecone"
	VectorStoreMilvus   VectorStoreType = "milvus"
	VectorStoreQdrant   VectorStoreType = "qdrant"
)

type VectorDocument struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Vector    []float64              `json:"vector,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
	Score     float64                `json:"score,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type VectorSearchResult struct {
	Document *VectorDocument `json:"document"`
	Score    float64         `json:"score"`
}

type VectorStoreConfig struct {
	Type          VectorStoreType
	WorkspacePath string
	APIKey        string
	Endpoint      string
	Collection    string
	Dimension     int
}

type VectorStore interface {
	Insert(ctx context.Context, docs []*VectorDocument) error
	Search(ctx context.Context, query []float64, limit int, filter map[string]interface{}) ([]*VectorSearchResult, error)
	Delete(ctx context.Context, ids []string) error
	Get(ctx context.Context, id string) (*VectorDocument, error)
	Update(ctx context.Context, doc *VectorDocument) error
	Clear(ctx context.Context) error
	Type() VectorStoreType
}

type InMemoryVectorStore struct {
	mu        sync.RWMutex
	documents map[string]*VectorDocument
	dimension int
	index     map[string][]int
}

func NewInMemoryVectorStore(dimension int) *InMemoryVectorStore {
	return &InMemoryVectorStore{
		documents: make(map[string]*VectorDocument),
		dimension: dimension,
		index:     make(map[string][]int),
	}
}

func (s *InMemoryVectorStore) Insert(ctx context.Context, docs []*VectorDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, doc := range docs {
		doc.CreatedAt = time.Now()
		s.documents[doc.ID] = doc
	}
	return nil
}

func (s *InMemoryVectorStore) Search(ctx context.Context, query []float64, limit int, filter map[string]interface{}) ([]*VectorSearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]*VectorSearchResult, 0)

	for _, doc := range s.documents {
		if doc.Vector == nil {
			continue
		}

		if !s.matchesFilter(doc, filter) {
			continue
		}

		score := cosineSimilarity(query, doc.Vector)
		if score > 0 {
			results = append(results, &VectorSearchResult{
				Document: doc,
				Score:    score,
			})
		}
	}

	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Score < results[j].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func (s *InMemoryVectorStore) matchesFilter(doc *VectorDocument, filter map[string]interface{}) bool {
	if filter == nil {
		return true
	}

	for key, value := range filter {
		if doc.Metadata == nil {
			return false
		}
		if docVal, exists := doc.Metadata[key]; !exists || docVal != value {
			return false
		}
	}
	return true
}

func (s *InMemoryVectorStore) Delete(ctx context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		delete(s.documents, id)
	}
	return nil
}

func (s *InMemoryVectorStore) Get(ctx context.Context, id string) (*VectorDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if doc, exists := s.documents[id]; exists {
		return doc, nil
	}
	return nil, fmt.Errorf("document not found: %s", id)
}

func (s *InMemoryVectorStore) Update(ctx context.Context, doc *VectorDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.documents[doc.ID]; !exists {
		return fmt.Errorf("document not found: %s", doc.ID)
	}

	s.documents[doc.ID] = doc
	return nil
}

func (s *InMemoryVectorStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.documents = make(map[string]*VectorDocument)
	return nil
}

func (s *InMemoryVectorStore) Type() VectorStoreType {
	return VectorStoreMemory
}

type PersistentVectorStore struct {
	*InMemoryVectorStore
	dataPath string
	mu       sync.RWMutex
}

func NewPersistentVectorStore(workspace string, dimension int) *PersistentVectorStore {
	dataPath := filepath.Join(workspace, "vectors")
	os.MkdirAll(dataPath, 0755)

	store := &PersistentVectorStore{
		InMemoryVectorStore: NewInMemoryVectorStore(dimension),
		dataPath:            dataPath,
	}

	store.loadFromDisk()
	return store
}

func (s *PersistentVectorStore) loadFromDisk() error {
	files, err := os.ReadDir(s.dataPath)
	if err != nil {
		return nil
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		path := filepath.Join(s.dataPath, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var doc VectorDocument
		if err := json.Unmarshal(data, &doc); err != nil {
			continue
		}

		s.documents[doc.ID] = &doc
	}

	return nil
}

func (s *PersistentVectorStore) saveToDisk(doc *VectorDocument) error {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(s.dataPath, doc.ID+".json")
	return os.WriteFile(path, data, 0644)
}

func (s *PersistentVectorStore) Insert(ctx context.Context, docs []*VectorDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, doc := range docs {
		doc.CreatedAt = time.Now()
		s.documents[doc.ID] = doc
		if err := s.saveToDisk(doc); err != nil {
			return err
		}
	}
	return nil
}

func (s *PersistentVectorStore) Delete(ctx context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		delete(s.documents, id)
		path := filepath.Join(s.dataPath, id+".json")
		os.Remove(path)
	}
	return nil
}

func (s *PersistentVectorStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	os.RemoveAll(s.dataPath)
	os.MkdirAll(s.dataPath, 0755)
	s.documents = make(map[string]*VectorDocument)
	return nil
}

func (s *PersistentVectorStore) Type() VectorStoreType {
	return VectorStoreMemory
}

type EmbeddingClient interface {
	Embed(ctx context.Context, texts []string) ([][]float64, error)
}

type MockEmbeddingClient struct {
	Dimension int
}

func (c *MockEmbeddingClient) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))
	for i := range texts {
		embeddings[i] = make([]float64, c.Dimension)
		for j := range embeddings[i] {
			embeddings[i][j] = 0.1
		}
	}
	return embeddings, nil
}

type VectorStoreManager struct {
	store    VectorStore
	embedder EmbeddingClient
	mu       sync.RWMutex
}

func NewVectorStoreManager(cfg *VectorStoreConfig) (*VectorStoreManager, error) {
	var store VectorStore
	var err error

	switch cfg.Type {
	case VectorStoreMemory:
		store = NewPersistentVectorStore(cfg.WorkspacePath, cfg.Dimension)
	case VectorStoreQdrant:
		store, err = NewQdrantVectorStore(cfg.Endpoint, cfg.APIKey, cfg.Collection, cfg.Dimension)
	default:
		store = NewPersistentVectorStore(cfg.WorkspacePath, cfg.Dimension)
	}

	if err != nil {
		return nil, err
	}

	return &VectorStoreManager{
		store:    store,
		embedder: &MockEmbeddingClient{Dimension: cfg.Dimension},
	}, nil
}

func (m *VectorStoreManager) SetEmbedder(embedder EmbeddingClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.embedder = embedder
}

func (m *VectorStoreManager) AddDocument(ctx context.Context, id, content string, metadata map[string]interface{}) error {
	embeddings, err := m.embedder.Embed(ctx, []string{content})
	if err != nil {
		return fmt.Errorf("failed to create embedding: %w", err)
	}

	doc := &VectorDocument{
		ID:       id,
		Content:  content,
		Vector:   embeddings[0],
		Metadata: metadata,
	}

	return m.store.Insert(ctx, []*VectorDocument{doc})
}

func (m *VectorStoreManager) AddDocuments(ctx context.Context, docs []*VectorDocument) error {
	if len(docs) == 0 {
		return nil
	}

	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}

	embeddings, err := m.embedder.Embed(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to create embeddings: %w", err)
	}

	for i, doc := range docs {
		doc.Vector = embeddings[i]
	}

	return m.store.Insert(ctx, docs)
}

func (m *VectorStoreManager) Search(ctx context.Context, query string, limit int, filter map[string]interface{}) ([]*VectorSearchResult, error) {
	embeddings, err := m.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}

	return m.store.Search(ctx, embeddings[0], limit, filter)
}

func (m *VectorStoreManager) SearchByVector(ctx context.Context, vector []float64, limit int, filter map[string]interface{}) ([]*VectorSearchResult, error) {
	return m.store.Search(ctx, vector, limit, filter)
}

func (m *VectorStoreManager) Delete(ctx context.Context, ids []string) error {
	return m.store.Delete(ctx, ids)
}

func (m *VectorStoreManager) Get(ctx context.Context, id string) (*VectorDocument, error) {
	return m.store.Get(ctx, id)
}

func (m *VectorStoreManager) Clear(ctx context.Context) error {
	return m.store.Clear(ctx)
}

func (m *VectorStoreManager) GetStore() VectorStore {
	return m.store
}

type KnowledgeEntry struct {
	ID         string
	Question   string
	Answer     string
	Category   string
	Source     string
	Confidence float64
	Tags       []string
}

func (m *VectorStoreManager) IndexKnowledge(ctx context.Context, entries []*KnowledgeEntry) error {
	docs := make([]*VectorDocument, len(entries))
	for i, entry := range entries {
		docs[i] = &VectorDocument{
			ID:      entry.ID,
			Content: fmt.Sprintf("Q: %s\nA: %s", entry.Question, entry.Answer),
			Metadata: map[string]interface{}{
				"question":   entry.Question,
				"answer":     entry.Answer,
				"category":   entry.Category,
				"source":     entry.Source,
				"confidence": entry.Confidence,
				"tags":       entry.Tags,
				"type":       "knowledge",
			},
		}
	}
	return m.AddDocuments(ctx, docs)
}

func (m *VectorStoreManager) SearchKnowledge(ctx context.Context, query string, limit int, category string) ([]*KnowledgeEntry, error) {
	filter := map[string]interface{}{"type": "knowledge"}
	if category != "" {
		filter["category"] = category
	}

	results, err := m.Search(ctx, query, limit, filter)
	if err != nil {
		return nil, err
	}

	entries := make([]*KnowledgeEntry, len(results))
	for i, result := range results {
		entry := &KnowledgeEntry{
			ID: result.Document.ID,
		}
		if q, ok := result.Document.Metadata["question"].(string); ok {
			entry.Question = q
		}
		if a, ok := result.Document.Metadata["answer"].(string); ok {
			entry.Answer = a
		}
		if c, ok := result.Document.Metadata["category"].(string); ok {
			entry.Category = c
		}
		if s, ok := result.Document.Metadata["source"].(string); ok {
			entry.Source = s
		}
		if conf, ok := result.Document.Metadata["confidence"].(float64); ok {
			entry.Confidence = conf
		}
		entries[i] = entry
	}

	return entries, nil
}
