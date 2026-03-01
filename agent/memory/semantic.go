package memory

import (
	"math"
	"sort"
	"strings"
)

// SemanticMemoryStore extends the basic MemoryStore with semantic search capabilities
type SemanticMemoryStore struct {
	*MemoryStore
	vectorizer Vectorizer
}

// Vectorizer interface for converting text to vectors
type Vectorizer interface {
	Vectorize(text string) []float64
}

// SimpleVectorizer is a basic vectorizer that creates TF-IDF-like vectors
type SimpleVectorizer struct {
	documents    []string
	documentFreq map[string]int
	vocabulary   map[string]int
}

// NewSemanticMemoryStore creates a new semantic memory store
func NewSemanticMemoryStore(workspace string) *SemanticMemoryStore {
	return &SemanticMemoryStore{
		MemoryStore: NewMemoryStore(workspace),
		vectorizer:  NewSimpleVectorizer(),
	}
}

// NewSimpleVectorizer creates a new simple vectorizer
func NewSimpleVectorizer() *SimpleVectorizer {
	return &SimpleVectorizer{
		documents:    make([]string, 0),
		documentFreq: make(map[string]int),
		vocabulary:   make(map[string]int),
	}
}

// Vectorize converts text to a vector representation
func (sv *SimpleVectorizer) Vectorize(text string) []float64 {
	words := sv.tokenize(text)
	vector := make([]float64, len(sv.vocabulary))

	// Calculate term frequencies
	termFreq := make(map[string]int)
	for _, word := range words {
		termFreq[word]++
	}

	// Create vector with TF-IDF-like scores
	for word, idx := range sv.vocabulary {
		tf := float64(termFreq[word]) / float64(len(words)) // Term frequency
		idf := 1.0 // In a full implementation, this would be calculated properly
		if df, exists := sv.documentFreq[word]; exists {
			if len(sv.documents) > df {
				idf = math.Log(float64(len(sv.documents)) / float64(df))
			}
		}
		vector[idx] = tf * idf
	}

	return vector
}

// tokenize splits text into lowercase words
func (sv *SimpleVectorizer) tokenize(text string) []string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, ".", " ")
	text = strings.ReplaceAll(text, ",", " ")
	text = strings.ReplaceAll(text, "!", " ")
	text = strings.ReplaceAll(text, "?", " ")
	text = strings.ReplaceAll(text, ";", " ")
	text = strings.ReplaceAll(text, ":", " ")

	words := strings.Fields(text)
	return words
}

// AddDocument adds a document to the vectorizer
func (sv *SimpleVectorizer) AddDocument(text string) {
	sv.documents = append(sv.documents, text)
	seen := make(map[string]bool)

	words := sv.tokenize(text)
	for _, word := range words {
		if !seen[word] {
			sv.documentFreq[word]++
			seen[word] = true
		}

		if _, exists := sv.vocabulary[word]; !exists {
			sv.vocabulary[word] = len(sv.vocabulary)
		}
	}
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(vec1, vec2 []float64) float64 {
	if len(vec1) != len(vec2) {
		return 0
	}

	var dotProduct, norm1, norm2 float64
	for i := range vec1 { // Use range instead of index loop
		dotProduct += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}

	if norm1 == 0 || norm2 == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// SearchMemory performs semantic search on the long-term memory
func (sms *SemanticMemoryStore) SearchMemory(query string, limit int) ([]MemorySearchResult, error) {
	longTerm, err := sms.MemoryStore.ReadLongTerm()
	if err != nil {
		return nil, err
	}

	if longTerm == "" {
		return []MemorySearchResult{}, nil
	}

	// For simplicity, treat the entire long-term memory as a single text
	// In a more advanced implementation, we would break it into chunks
	segments := sms.segmentText(longTerm)

	queryVector := sms.vectorizer.Vectorize(query)
	results := make([]MemorySearchResult, 0)

	for i, segment := range segments {
		segmentVector := sms.vectorizer.Vectorize(segment)
		similarity := cosineSimilarity(queryVector, segmentVector)

		if similarity > 0.1 { // Threshold to filter out low-similarity results
			results = append(results, MemorySearchResult{
				Segment:    segment,
				Similarity: similarity,
				Index:      i,
			})
		}
	}

	// Sort results by similarity in descending order
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	// Limit results
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// segmentText breaks the text into manageable segments
func (sms *SemanticMemoryStore) segmentText(text string) []string {
	const maxSegmentLength = 500
	segments := make([]string, 0)

	// Split by paragraphs first
	paragraphs := strings.Split(text, "\n\n")

	for _, paragraph := range paragraphs {
		if len(paragraph) <= maxSegmentLength {
			segments = append(segments, paragraph)
		} else {
			// Break long paragraphs into sentences
			sentences := strings.Split(paragraph, ". ") // Use Split instead of SplitSeq for Go compatibility
			currentSegment := ""

			for _, sentence := range sentences {
				sentence = strings.TrimSpace(sentence) + ". "

				if len(currentSegment)+len(sentence) <= maxSegmentLength {
					currentSegment += sentence
				} else {
					if currentSegment != "" {
						segments = append(segments, currentSegment)
					}
					currentSegment = sentence
				}
			}

			if currentSegment != "" {
				segments = append(segments, currentSegment)
			}
		}
	}

	return segments
}

// MemorySearchResult represents a result from semantic memory search
type MemorySearchResult struct {
	Segment    string
	Similarity float64
	Index      int
}

// SearchHistory performs semantic search on the history log
func (sms *SemanticMemoryStore) SearchHistory(query string, limit int) ([]MemorySearchResult, error) {
	// Read the history file
	content, err := sms.readHistoryFile()
	if err != nil {
		return nil, err
	}

	if content == "" {
		return []MemorySearchResult{}, nil
	}

	// Segment the history content
	segments := sms.segmentText(content)
	queryVector := sms.vectorizer.Vectorize(query)
	results := make([]MemorySearchResult, 0)

	for i, segment := range segments {
		segmentVector := sms.vectorizer.Vectorize(segment)
		similarity := cosineSimilarity(queryVector, segmentVector)

		if similarity > 0.1 { // Threshold to filter out low-similarity results
			results = append(results, MemorySearchResult{
				Segment:    segment,
				Similarity: similarity,
				Index:      i,
			})
		}
	}

	// Sort results by similarity in descending order
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	// Limit results
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// readHistoryFile reads the history file content
func (sms *SemanticMemoryStore) readHistoryFile() (string, error) {
	// In a real implementation, this would read the actual history file
	// For now, returning empty string to indicate no implementation
	return "", nil
}