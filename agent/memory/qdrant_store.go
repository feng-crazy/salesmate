package memory

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/qdrant/go-client/qdrant"
)

// QdrantVectorStore implements VectorStore using Qdrant vector database
type QdrantVectorStore struct {
	pointsClient qdrant.PointsClient
	collection   string
	dimension    int
}

// NewQdrantVectorStore creates a new Qdrant vector store
func NewQdrantVectorStore(endpoint, apiKey, collection string, dimension int) (*QdrantVectorStore, error) {
	// Parse endpoint into host and port
	host := "localhost"
	port := 6333
	if endpoint != "" {
		fmt.Sscanf(endpoint, "%s:%d", &host, &port)
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   host,
		Port:   port,
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	store := &QdrantVectorStore{
		pointsClient: client.GetPointsClient(),
		collection:   collection,
		dimension:    dimension,
	}

	// Create collection if it doesn't exist
	if err := store.ensureCollection(client); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	return store, nil
}

// ensureCollection creates the collection if it doesn't exist
func (s *QdrantVectorStore) ensureCollection(client *qdrant.Client) error {
	ctx := context.Background()
	exists, err := client.CollectionExists(ctx, s.collection)
	if err != nil {
		return fmt.Errorf("failed to check collection exists: %w", err)
	}

	if !exists {
		return client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: s.collection,
			VectorsConfig: &qdrant.VectorsConfig{
				Config: &qdrant.VectorsConfig_Params{
					Params: &qdrant.VectorParams{
						Size:     uint64(s.dimension),
						Distance: qdrant.Distance_Cosine,
					},
				},
			},
		})
	}

	return nil
}

// Insert adds documents to the vector store
func (s *QdrantVectorStore) Insert(ctx context.Context, docs []*VectorDocument) error {
	if len(docs) == 0 {
		return nil
	}

	points := make([]*qdrant.PointStruct, len(docs))
	for i, doc := range docs {
		if doc.Vector == nil {
			continue
		}

		// Convert []float64 to []float32
		vec32 := make([]float32, len(doc.Vector))
		for j, v := range doc.Vector {
			vec32[j] = float32(v)
		}

		points[i] = &qdrant.PointStruct{
			Id:      qdrant.NewID(doc.ID),
			Vectors: qdrant.NewVectorsDense(vec32),
			Payload: s.payloadFromDoc(doc),
		}
	}

	_, err := s.pointsClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collection,
		Points:         points,
	})

	if err != nil {
		return fmt.Errorf("failed to insert documents: %w", err)
	}

	return nil
}

// Search performs cosine similarity search
func (s *QdrantVectorStore) Search(ctx context.Context, query []float64, limit int, filter map[string]interface{}) ([]*VectorSearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	// Convert []float64 to []float32
	query32 := make([]float32, len(query))
	for i, v := range query {
		query32[i] = float32(v)
	}

	var filterProto *qdrant.Filter
	if filter != nil && len(filter) > 0 {
		filterProto = s.filterFromMap(filter)
	}

	limitUint := uint64(limit)
	results, err := s.pointsClient.Search(ctx, &qdrant.SearchPoints{
		CollectionName: s.collection,
		Vector:         query32,
		Limit:          limitUint,
		Filter:         filterProto,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	searchResults := make([]*VectorSearchResult, 0, len(results.GetResult()))
	for _, result := range results.GetResult() {
		doc := s.docFromScoredPoint(result)
		searchResults = append(searchResults, &VectorSearchResult{
			Document: doc,
			Score:    float64(result.GetScore()),
		})
	}

	return searchResults, nil
}

// Delete removes documents by IDs
func (s *QdrantVectorStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	pointIDs := make([]*qdrant.PointId, len(ids))
	for i, id := range ids {
		pointIDs[i] = qdrant.NewID(id)
	}

	_, err := s.pointsClient.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: s.collection,
		Points:         qdrant.NewPointsSelectorIDs(pointIDs),
	})

	if err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}

	return nil
}

// Get retrieves a document by ID
func (s *QdrantVectorStore) Get(ctx context.Context, id string) (*VectorDocument, error) {
	pointID := qdrant.NewID(id)
	results, err := s.pointsClient.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.collection,
		Ids:            []*qdrant.PointId{pointID},
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	result := results.GetResult()
	if len(result) == 0 {
		return nil, fmt.Errorf("document not found: %s", id)
	}

	return s.docFromRetrievedPoint(result[0]), nil
}

// Update updates an existing document
func (s *QdrantVectorStore) Update(ctx context.Context, doc *VectorDocument) error {
	return s.Insert(ctx, []*VectorDocument{doc})
}

// Clear removes all documents from the collection - simplified version using Scroll to get all then delete
func (s *QdrantVectorStore) Clear(ctx context.Context) error {
	// Scroll through all points and delete them in batches
	limit := uint32(100)
	offset := (*qdrant.PointId)(nil)

	for {
		scrollReq := &qdrant.ScrollPoints{
			CollectionName: s.collection,
			Limit:          &limit,
			Offset:         offset,
			WithPayload:    qdrant.NewWithPayload(false),
		}

		scrollResp, err := s.pointsClient.Scroll(ctx, scrollReq)
		if err != nil {
			return fmt.Errorf("failed to scroll: %w", err)
		}

		points := scrollResp.GetResult()
		if len(points) == 0 {
			break
		}

		// Collect IDs
		pointIDs := make([]*qdrant.PointId, len(points))
		for i, p := range points {
			pointIDs[i] = p.GetId()
		}

		// Delete these points
		_, _ = s.pointsClient.Delete(ctx, &qdrant.DeletePoints{
			CollectionName: s.collection,
			Points:         qdrant.NewPointsSelectorIDs(pointIDs),
		})

		// Check if there are more - offset is the last point's ID
		if len(points) < int(limit) {
			break
		}
		offset = points[len(points)-1].GetId()
	}

	return nil
}

// Type returns the vector store type
func (s *QdrantVectorStore) Type() VectorStoreType {
	return VectorStoreQdrant
}

// payloadFromDoc converts a VectorDocument to Qdrant payload
func (s *QdrantVectorStore) payloadFromDoc(doc *VectorDocument) map[string]*qdrant.Value {
	payload := map[string]interface{}{
		"id":         doc.ID,
		"content":    doc.Content,
		"created_at": doc.CreatedAt.Unix(),
	}

	for k, v := range doc.Metadata {
		payload[k] = v
	}

	return qdrant.NewValueMap(payload)
}

// docFromScoredPoint converts a ScoredPoint to VectorDocument
func (s *QdrantVectorStore) docFromScoredPoint(point *qdrant.ScoredPoint) *VectorDocument {
	doc := &VectorDocument{
		Metadata: make(map[string]interface{}),
	}

	if point.GetId() != nil {
		doc.ID = point.GetId().GetUuid()
		if doc.ID == "" {
			doc.ID = strconv.FormatUint(point.GetId().GetNum(), 10)
		}
	}

	doc.Score = float64(point.GetScore())

	// Extract payload
	if payload := point.GetPayload(); payload != nil {
		s.extractPayloadToDoc(payload, doc)
	}

	return doc
}

// docFromRetrievedPoint converts a RetrievedPoint to VectorDocument
func (s *QdrantVectorStore) docFromRetrievedPoint(point *qdrant.RetrievedPoint) *VectorDocument {
	doc := &VectorDocument{
		Metadata: make(map[string]interface{}),
	}

	if point.GetId() != nil {
		doc.ID = point.GetId().GetUuid()
		if doc.ID == "" {
			doc.ID = strconv.FormatUint(point.GetId().GetNum(), 10)
		}
	}

	// Extract payload
	if payload := point.GetPayload(); payload != nil {
		s.extractPayloadToDoc(payload, doc)
	}

	return doc
}

// extractPayloadToDoc extracts payload data into VectorDocument
func (s *QdrantVectorStore) extractPayloadToDoc(payload map[string]*qdrant.Value, doc *VectorDocument) {
	if content, ok := payload["content"]; ok {
		doc.Content = content.GetStringValue()
	}

	if createdAt, ok := payload["created_at"]; ok {
		intVal := createdAt.GetIntegerValue()
		if intVal != 0 {
			doc.CreatedAt = time.Unix(intVal, 0)
		}
	}

	// Copy all other fields to metadata
	for k, v := range payload {
		if k == "id" || k == "content" || k == "created_at" {
			continue
		}
		doc.Metadata[k] = s.extractValue(v)
	}
}

// extractValue extracts Go value from Qdrant Value
func (s *QdrantVectorStore) extractValue(v *qdrant.Value) interface{} {
	switch v.GetKind().(type) {
	case *qdrant.Value_StringValue:
		return v.GetStringValue()
	case *qdrant.Value_IntegerValue:
		return v.GetIntegerValue()
	case *qdrant.Value_DoubleValue:
		return v.GetDoubleValue()
	case *qdrant.Value_BoolValue:
		return v.GetBoolValue()
	case *qdrant.Value_ListValue:
		list := v.GetListValue()
		items := make([]interface{}, len(list.Values))
		for i, item := range list.Values {
			items[i] = s.extractValue(item)
		}
		return items
	case *qdrant.Value_NullValue:
		return nil
	}
	return nil
}

// filterFromMap converts a map filter to Qdrant Filter
func (s *QdrantVectorStore) filterFromMap(filter map[string]interface{}) *qdrant.Filter {
	conditions := make([]*qdrant.Condition, 0, len(filter))

	for key, value := range filter {
		// Use the appropriate match function based on value type
		switch v := value.(type) {
		case string:
			conditions = append(conditions, qdrant.NewMatchKeyword(key, v))
		case int, int64:
			conditions = append(conditions, qdrant.NewMatchInt(key, int64(v.(int))))
		case bool:
			conditions = append(conditions, qdrant.NewMatchBool(key, v))
		default:
			conditions = append(conditions, qdrant.NewMatch(key, fmt.Sprintf("%v", v)))
		}
	}

	return &qdrant.Filter{
		Must: conditions,
	}
}
