package knowledge

import (
	"fmt"
	"strings"
	"unicode"
)

// ChunkConfig holds configuration for text chunking
type ChunkConfig struct {
	ChunkSize    int // Maximum size of each chunk in characters (default 512)
	ChunkOverlap int // Number of overlapping characters between chunks (default 50)
	MinChunkSize int // Minimum chunk size to keep (default 100)
}

// DefaultChunkConfig returns default chunking configuration
func DefaultChunkConfig() *ChunkConfig {
	return &ChunkConfig{
		ChunkSize:    512,
		ChunkOverlap: 50,
		MinChunkSize: 100,
	}
}

// Chunk represents a text chunk with metadata
type Chunk struct {
	ID          string            `json:"id"`
	Content     string            `json:"content"`
	Index       int               `json:"index"`
	StartOffset int               `json:"start_offset"`
	EndOffset   int               `json:"end_offset"`
	Metadata    map[string]string `json:"metadata"`
}

// TextChunker splits text into overlapping chunks
type TextChunker struct {
	config *ChunkConfig
}

// NewTextChunker creates a new text chunker with the given config
func NewTextChunker(config *ChunkConfig) *TextChunker {
	if config == nil {
		config = DefaultChunkConfig()
	}
	// Ensure valid defaults
	if config.ChunkSize <= 0 {
		config.ChunkSize = 512
	}
	if config.ChunkOverlap < 0 {
		config.ChunkOverlap = 0
	}
	if config.ChunkOverlap >= config.ChunkSize {
		config.ChunkOverlap = config.ChunkSize / 4
	}
	if config.MinChunkSize <= 0 {
		config.MinChunkSize = 100
	}
	if config.MinChunkSize > config.ChunkSize {
		config.MinChunkSize = config.ChunkSize / 4
	}
	return &TextChunker{config: config}
}

// Chunk splits text into chunks using semantic boundaries
func (tc *TextChunker) Chunk(text string) []Chunk {
	return tc.ChunkWithMetadata(text, nil)
}

// ChunkWithMetadata splits text into chunks with custom metadata
func (tc *TextChunker) ChunkWithMetadata(text string, metadata map[string]string) []Chunk {
	if text == "" {
		return nil
	}

	// Try sentence-based splitting first
	chunks := tc.splitBySentences(text)

	// If single chunk or too few chunks, try paragraph splitting
	if len(chunks) <= 1 {
		chunks = tc.splitByParagraphs(text)
	}

	// If still single chunk, use fixed-size splitting
	if len(chunks) <= 1 {
		chunks = tc.splitByFixedSize(text)
	}

	// Merge small chunks and create final output
	return tc.mergeAndEnumerate(chunks, metadata)
}

// splitBySentences splits text at sentence boundaries
func (tc *TextChunker) splitBySentences(text string) []string {
	if len(text) <= tc.config.ChunkSize {
		return []string{text}
	}

	var chunks []string
	var current strings.Builder

	sentences := tc.splitIntoSentences(text)

	for _, sentence := range sentences {
		if current.Len()+len(sentence) > tc.config.ChunkSize && current.Len() > 0 {
			chunks = append(chunks, current.String())
			// Keep overlap from previous chunk
			overlapStart := max(0, current.Len()-tc.config.ChunkOverlap)
			overlap := ""
			if overlapStart > 0 {
				overlapStart = strings.LastIndex(current.String()[:overlapStart], " ")
				if overlapStart > 0 {
					overlap = current.String()[overlapStart:]
				}
			}
			current.Reset()
			current.WriteString(overlap)
		}
		current.WriteString(sentence)
	}

	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}

	return chunks
}

// splitIntoSentences splits text into sentences
func (tc *TextChunker) splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	i := 0

	for i < len(runes) {
		current.WriteRune(runes[i])

		// Check for sentence ending punctuation
		if isSentenceEnd(runes, i) {
			sentences = append(sentences, current.String())
			current.Reset()
		}
		i++
	}

	// Add remaining text
	if current.Len() > 0 {
		sentences = append(sentences, current.String())
	}

	return sentences
}

// isSentenceEnd checks if current position is end of a sentence
func isSentenceEnd(runes []rune, i int) bool {
	if i >= len(runes)-1 {
		return false
	}

	r := runes[i]
	if r != '.' && r != '!' && r != '?' && r != ';' {
		return false
	}

	// Check if followed by space and uppercase or quote
	next := runes[i+1]
	if !unicode.IsSpace(next) && next != '"' && next != '\'' {
		return false
	}

	// Skip whitespace and check for sentence start
	j := i + 1
	for j < len(runes) && unicode.IsSpace(runes[j]) {
		j++
	}

	if j < len(runes) {
		return unicode.IsUpper(runes[j]) || runes[j] == '"' || runes[j] == '\''
	}

	return false
}

// splitByParagraphs splits text at paragraph boundaries
func (tc *TextChunker) splitByParagraphs(text string) []string {
	if len(text) <= tc.config.ChunkSize {
		return []string{text}
	}

	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	var current strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// Keep paragraph together if possible
		if current.Len()+len(para) > tc.config.ChunkSize && current.Len() > 0 {
			chunks = append(chunks, current.String())
			// Apply overlap
			overlapStart := max(0, current.Len()-tc.config.ChunkOverlap)
			overlap := ""
			if overlapStart > 0 {
				overlapStart = strings.LastIndex(current.String()[:overlapStart], " ")
				if overlapStart > 0 {
					overlap = current.String()[overlapStart:]
				}
			}
			current.Reset()
			current.WriteString(overlap)
		}

		if current.Len() > 0 && current.Len()+len(para)+1 <= tc.config.ChunkSize {
			current.WriteString(" ")
		}
		current.WriteString(para)
	}

	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}

	return chunks
}

// splitByFixedSize splits text by fixed character count with overlap
func (tc *TextChunker) splitByFixedSize(text string) []string {
	if len(text) <= tc.config.ChunkSize {
		return []string{text}
	}

	var chunks []string
	start := 0

	for start < len(text) {
		end := start + tc.config.ChunkSize
		if end > len(text) {
			end = len(text)
		}

		// Try to break at word boundary
		if end < len(text) {
			breakPoint := strings.LastIndex(text[start:end], " ")
			if breakPoint > tc.config.MinChunkSize {
				end = start + breakPoint
			}
		}

		chunks = append(chunks, text[start:end])

		// Move start, accounting for overlap
		start = end - tc.config.ChunkOverlap
		if start < 0 {
			start = 0
		}
	}

	return chunks
}

// mergeAndEnumerate merges small chunks and creates final Chunk objects
func (tc *TextChunker) mergeAndEnumerate(segments []string, metadata map[string]string) []Chunk {
	if len(segments) == 0 {
		return nil
	}

	// Merge small chunks
	var merged []string
	var current strings.Builder

	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		if current.Len()+len(seg) <= tc.config.ChunkSize {
			if current.Len() > 0 {
				current.WriteString(" ")
			}
			current.WriteString(seg)
		} else {
			if current.Len() > 0 {
				merged = append(merged, current.String())
			}
			// Check if current segment is too small to be standalone
			if len(seg) < tc.config.MinChunkSize && len(merged) > 0 {
				// Append to previous chunk
				merged[len(merged)-1] += " " + seg
			} else {
				current.Reset()
				current.WriteString(seg)
			}
		}
	}

	if current.Len() > 0 {
		// Merge small final chunk with previous if possible
		if len(merged) > 0 && current.Len() < tc.config.MinChunkSize {
			merged[len(merged)-1] += " " + current.String()
		} else {
			merged = append(merged, current.String())
		}
	}

	// Create Chunk objects with offsets
	var chunks []Chunk
	offset := 0

	for i, content := range merged {
		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}

		startOffset := offset
		endOffset := startOffset + len(content)

		chunkMetadata := make(map[string]string)
		if metadata != nil {
			for k, v := range metadata {
				chunkMetadata[k] = v
			}
		}
		chunkMetadata["chunk_size"] = fmt.Sprintf("%d", len(content))

		chunks = append(chunks, Chunk{
			ID:          fmt.Sprintf("chunk_%d_%x", i, len(content)),
			Content:     content,
			Index:       i,
			StartOffset: startOffset,
			EndOffset:   endOffset,
			Metadata:    chunkMetadata,
		})

		offset = endOffset
	}

	return chunks
}
