package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// VectorStore defines the interface for vector storage backends
type VectorStore interface {
	// Upsert inserts or updates vectors
	Upsert(ctx context.Context, collection string, vectors []Vector) error
	// Search performs similarity search
	Search(ctx context.Context, collection string, query Vector, limit int) ([]SearchResult, error)
	// Delete removes vectors by IDs
	Delete(ctx context.Context, collection string, ids []string) error
	// Get retrieves vectors by IDs
	Get(ctx context.Context, collection string, ids []string) ([]Vector, error)
}

// Vector represents a document with its embedding
type Vector struct {
	ID       string                 `json:"id"`
	Vector   []float64              `json:"vector"`
	Metadata map[string]interface{} `json:"metadata"`
	Text     string                 `json:"text"`
	Score    float64                `json:"score,omitempty"`
}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	Vector
	Score float64 `json:"score"`
}

// EmbeddingProvider defines the interface for embedding generation
type EmbeddingProvider interface {
	Embed(ctx context.Context, texts []string) ([][]float64, error)
	EmbeddingDimension() int
}

// RAGSystem implements Retrieval-Augmented Generation
type RAGSystem struct {
	vectorStore       VectorStore
	embeddingProvider EmbeddingProvider
	collection        string
	mu                sync.RWMutex
	documents         map[string]*Document
}

// Document represents a document in the knowledge base
type Document struct {
	ID          string            `json:"id"`
	Content     string            `json:"content"`
	Title       string            `json:"title"`
	Source      string            `json:"source"`
	Category    string            `json:"category"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata"`
	Embedding   []float64         `json:"embedding,omitempty"`
	ChunkIndex  int               `json:"chunk_index,omitempty"`
	TotalChunks int               `json:"total_chunks,omitempty"`
}

// RAGConfig holds configuration for the RAG system
type RAGConfig struct {
	Collection        string
	ChunkSize         int
	ChunkOverlap      int
	MaxSearchResults  int
	ScoreThreshold    float64
}

// DefaultRAGConfig returns default RAG configuration
func DefaultRAGConfig() *RAGConfig {
	return &RAGConfig{
		Collection:       "salesmate_kb",
		ChunkSize:        512,
		ChunkOverlap:     50,
		MaxSearchResults: 5,
		ScoreThreshold:   0.7,
	}
}

// NewRAGSystem creates a new RAG system
func NewRAGSystem(store VectorStore, embedder EmbeddingProvider, config *RAGConfig) *RAGSystem {
	if config == nil {
		config = DefaultRAGConfig()
	}
	return &RAGSystem{
		vectorStore:       store,
		embeddingProvider: embedder,
		collection:        config.Collection,
		documents:         make(map[string]*Document),
	}
}

// IngestDocument adds a document to the knowledge base
func (r *RAGSystem) IngestDocument(ctx context.Context, doc *Document) error {
	if doc.ID == "" {
		doc.ID = generateDocID(doc.Content)
	}

	// Generate embedding if not provided
	if len(doc.Embedding) == 0 {
		embeddings, err := r.embeddingProvider.Embed(ctx, []string{doc.Content})
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}
		if len(embeddings) > 0 {
			doc.Embedding = embeddings[0]
		}
	}

	// Store the document
	r.mu.Lock()
	r.documents[doc.ID] = doc
	r.mu.Unlock()

	// Store in vector database
	vector := Vector{
		ID:       doc.ID,
		Vector:   doc.Embedding,
		Text:     doc.Content,
		Metadata: documentToMetadata(doc),
	}

	return r.vectorStore.Upsert(ctx, r.collection, []Vector{vector})
}

// IngestDocuments adds multiple documents to the knowledge base
func (r *RAGSystem) IngestDocuments(ctx context.Context, docs []*Document) error {
	for _, doc := range docs {
		if err := r.IngestDocument(ctx, doc); err != nil {
			return fmt.Errorf("failed to ingest document %s: %w", doc.ID, err)
		}
	}
	return nil
}

// Search performs semantic search on the knowledge base
func (r *RAGSystem) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 5
	}

	// Generate embedding for query
	embeddings, err := r.embeddingProvider.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding generated for query")
	}

	// Perform vector search
	queryVector := Vector{Vector: embeddings[0]}
	results, err := r.vectorStore.Search(ctx, r.collection, queryVector, limit)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	return results, nil
}

// QueryWithRAG performs RAG-based query: retrieve relevant docs and generate context
func (r *RAGSystem) QueryWithRAG(ctx context.Context, query string, config *RAGConfig) (*RAGResponse, error) {
	if config == nil {
		config = DefaultRAGConfig()
	}

	// Search for relevant documents
	results, err := r.Search(ctx, query, config.MaxSearchResults)
	if err != nil {
		return nil, err
	}

	// Filter by score threshold
	var relevantDocs []SearchResult
	for _, result := range results {
		if result.Score >= config.ScoreThreshold {
			relevantDocs = append(relevantDocs, result)
		}
	}

	// Build context from retrieved documents
	var contextBuilder strings.Builder
	contextBuilder.WriteString("Retrieved relevant information:\n\n")
	for i, doc := range relevantDocs {
		contextBuilder.WriteString(fmt.Sprintf("[%d] %s\n", i+1, doc.Text))
		if i < len(relevantDocs)-1 {
			contextBuilder.WriteString("\n---\n\n")
		}
	}

	return &RAGResponse{
		Query:         query,
		Results:       relevantDocs,
		Context:       contextBuilder.String(),
		HasResults:    len(relevantDocs) > 0,
		Confidence:    calculateConfidence(relevantDocs),
		Sources:       extractSources(relevantDocs),
	}, nil
}

// RAGResponse represents the response from a RAG query
type RAGResponse struct {
	Query      string         `json:"query"`
	Results    []SearchResult `json:"results"`
	Context    string         `json:"context"`
	HasResults bool           `json:"has_results"`
	Confidence float64        `json:"confidence"`
	Sources    []string       `json:"sources"`
}

// ValidateAnswer checks if an answer is grounded in the retrieved context
func (r *RAGSystem) ValidateAnswer(ctx context.Context, answer string, sources []SearchResult) *ValidationResult {
	result := &ValidationResult{
		IsValid:    true,
		Confidence: 1.0,
		Warnings:   []string{},
	}

	// Check if answer makes claims not supported by sources
	// This is a simplified validation - in production, you'd use an LLM to verify
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check for specific claims that might not be in sources
	unsupportedClaims := []string{}
	sensitiveTerms := []string{"guarantee", "always", "never", "100%", "certainly", "definitely"}

	answerLower := strings.ToLower(answer)
	for _, term := range sensitiveTerms {
		if strings.Contains(answerLower, term) {
			unsupportedClaims = append(unsupportedClaims, fmt.Sprintf("Contains absolute claim: '%s'", term))
			result.Confidence -= 0.2
		}
	}

	if len(unsupportedClaims) > 0 {
		result.Warnings = unsupportedClaims
		result.RequiresReview = true
	}

	// Calculate overall confidence
	if result.Confidence < 0.5 {
		result.IsValid = false
		result.Message = "Answer may contain unverified claims"
	}

	return result
}

// ValidationResult represents the result of answer validation
type ValidationResult struct {
	IsValid        bool     `json:"is_valid"`
	Confidence     float64  `json:"confidence"`
	Message        string   `json:"message"`
	Warnings       []string `json:"warnings"`
	RequiresReview bool     `json:"requires_review"`
}

// DeleteDocument removes a document from the knowledge base
func (r *RAGSystem) DeleteDocument(ctx context.Context, id string) error {
	r.mu.Lock()
	delete(r.documents, id)
	r.mu.Unlock()

	return r.vectorStore.Delete(ctx, r.collection, []string{id})
}

// GetDocument retrieves a document by ID
func (r *RAGSystem) GetDocument(id string) (*Document, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	doc, exists := r.documents[id]
	return doc, exists
}

// ListDocuments returns all documents
func (r *RAGSystem) ListDocuments() []*Document {
	r.mu.RLock()
	defer r.mu.RUnlock()

	docs := make([]*Document, 0, len(r.documents))
	for _, doc := range r.documents {
		docs = append(docs, doc)
	}
	return docs
}

// LoadFromDirectory loads documents from a directory
func (r *RAGSystem) LoadFromDirectory(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".md") && !strings.HasSuffix(entry.Name(), ".txt") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Warning: failed to read file %s: %v\n", filePath, err)
			continue
		}

		doc := &Document{
			ID:       generateDocID(string(content)),
			Content:  string(content),
			Title:    entry.Name(),
			Source:   filePath,
			Category: "knowledge",
			Metadata: map[string]string{
				"filename": entry.Name(),
				"path":     filePath,
			},
		}

		if err := r.IngestDocument(ctx, doc); err != nil {
			fmt.Printf("Warning: failed to ingest %s: %v\n", filePath, err)
		}
	}

	return nil
}

// Helper functions

func generateDocID(content string) string {
	return fmt.Sprintf("doc_%x", len(content))
}

func documentToMetadata(doc *Document) map[string]interface{} {
	metadata := make(map[string]interface{})
	metadata["title"] = doc.Title
	metadata["source"] = doc.Source
	metadata["category"] = doc.Category
	for k, v := range doc.Metadata {
		metadata[k] = v
	}
	return metadata
}

func calculateConfidence(results []SearchResult) float64 {
	if len(results) == 0 {
		return 0.0
	}

	var totalScore float64
	for _, r := range results {
		totalScore += r.Score
	}
	return totalScore / float64(len(results))
}

func extractSources(results []SearchResult) []string {
	sources := make([]string, 0, len(results))
	for _, r := range results {
		if source, ok := r.Metadata["source"].(string); ok && source != "" {
			sources = append(sources, source)
		}
	}
	return sources
}

// MockEmbeddingProvider provides mock embeddings for testing
type MockEmbeddingProvider struct {
	Dimension int
}

func NewMockEmbeddingProvider() *MockEmbeddingProvider {
	return &MockEmbeddingProvider{Dimension: 384}
}

func (m *MockEmbeddingProvider) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))
	for i := range texts {
		// Generate deterministic mock embedding based on text length
		embedding := make([]float64, m.Dimension)
		for j := 0; j < m.Dimension; j++ {
			embedding[j] = float64(len(texts[i])*j%100) / 100.0
		}
		embeddings[i] = embedding
	}
	return embeddings, nil
}

func (m *MockEmbeddingProvider) EmbeddingDimension() int {
	return m.Dimension
}

// MemoryVectorStore is an in-memory vector store for testing
type MemoryVectorStore struct {
	mu      sync.RWMutex
	vectors map[string]map[string]Vector // collection -> id -> vector
}

func NewMemoryVectorStore() *MemoryVectorStore {
	return &MemoryVectorStore{
		vectors: make(map[string]map[string]Vector),
	}
}

func (m *MemoryVectorStore) Upsert(ctx context.Context, collection string, vectors []Vector) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.vectors[collection]; !exists {
		m.vectors[collection] = make(map[string]Vector)
	}

	for _, v := range vectors {
		m.vectors[collection][v.ID] = v
	}
	return nil
}

func (m *MemoryVectorStore) Search(ctx context.Context, collection string, query Vector, limit int) ([]SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	collectionVectors, exists := m.vectors[collection]
	if !exists {
		return []SearchResult{}, nil
	}

	// Calculate similarity scores
	type scoredVector struct {
		vector Vector
		score  float64
	}

	var scored []scoredVector
	for _, v := range collectionVectors {
		score := cosineSimilarity(query.Vector, v.Vector)
		scored = append(scored, scoredVector{vector: v, score: score})
	}

	// Sort by score (descending)
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Return top results
	results := make([]SearchResult, 0, limit)
	for i := 0; i < len(scored) && i < limit; i++ {
		results = append(results, SearchResult{
			Vector: scored[i].vector,
			Score:  scored[i].score,
		})
	}

	return results, nil
}

func (m *MemoryVectorStore) Delete(ctx context.Context, collection string, ids []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if collectionVectors, exists := m.vectors[collection]; exists {
		for _, id := range ids {
			delete(collectionVectors, id)
		}
	}
	return nil
}

func (m *MemoryVectorStore) Get(ctx context.Context, collection string, ids []string) ([]Vector, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	collectionVectors, exists := m.vectors[collection]
	if !exists {
		return []Vector{}, nil
	}

	vectors := make([]Vector, 0, len(ids))
	for _, id := range ids {
		if v, exists := collectionVectors[id]; exists {
			vectors = append(vectors, v)
		}
	}
	return vectors, nil
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// JSON marshaling for persistence
func (r *RAGSystem) SaveToFile(path string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	docs := make([]*Document, 0, len(r.documents))
	for _, doc := range r.documents {
		docs = append(docs, doc)
	}

	data, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (r *RAGSystem) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet
		}
		return err
	}

	var docs []*Document
	if err := json.Unmarshal(data, &docs); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, doc := range docs {
		r.documents[doc.ID] = doc
	}

	return nil
}