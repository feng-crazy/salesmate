package knowledge

import (
	"salesmate/agent/memory"
	"salesmate/sales_agent"
)

// SalesKnowledgeBaseHandler provides an interface to interact with sales-specific knowledge
type SalesKnowledgeBaseHandler struct {
	kb *sales_agent.SalesKnowledgeBase
	semStore *memory.SemanticMemoryStore
}

// NewSalesKnowledgeBase creates a new sales knowledge base handler
func NewSalesKnowledgeBase(workspace string) *SalesKnowledgeBaseHandler {
	return &SalesKnowledgeBaseHandler{
		kb: sales_agent.NewSalesKnowledgeBase(workspace),
		semStore: memory.NewSemanticMemoryStore(workspace),
	}
}

// QueryProduct searches for product information
func (skbh *SalesKnowledgeBaseHandler) QueryProduct(query string) []sales_agent.Product {
	return skbh.kb.QueryProduct(query)
}

// QueryPricing searches for pricing information
func (skbh *SalesKnowledgeBaseHandler) QueryPricing(query string) []sales_agent.PriceTier {
	return skbh.kb.QueryPricing(query)
}

// QueryFAQ searches for FAQ answers
func (skbh *SalesKnowledgeBaseHandler) QueryFAQ(query string) map[string]string {
	return skbh.kb.QueryFAQ(query)
}

// GetProductByID returns a product by its ID
func (skbh *SalesKnowledgeBaseHandler) GetProductByID(id string) (*sales_agent.Product, bool) {
	return skbh.kb.GetProductByID(id)
}

// GetPricingByName returns pricing info by name
func (skbh *SalesKnowledgeBaseHandler) GetPricingByName(name string) (*sales_agent.PriceTier, bool) {
	return skbh.kb.GetPricingByName(name)
}

// CompareWithCompetitor compares our offering with a competitor
func (skbh *SalesKnowledgeBaseHandler) CompareWithCompetitor(competitorName string) (*sales_agent.CompetitorInfo, bool) {
	return skbh.kb.CompareWithCompetitor(competitorName)
}

// GetAllProducts returns all products
func (skbh *SalesKnowledgeBaseHandler) GetAllProducts() []sales_agent.Product {
	return skbh.kb.GetAllProducts()
}

// GetAllPricing returns all pricing tiers
func (skbh *SalesKnowledgeBaseHandler) GetAllPricing() []sales_agent.PriceTier {
	return skbh.kb.GetAllPricing()
}

// ValidateProductInfo checks if the provided product information is accurate
func (skbh *SalesKnowledgeBaseHandler) ValidateProductInfo(productID, feature string) (bool, string) {
	return skbh.kb.ValidateProductInfo(productID, feature)
}

// SearchMemory performs semantic search on the sales knowledge
func (skbh *SalesKnowledgeBaseHandler) SearchMemory(query string, limit int) ([]memory.MemorySearchResult, error) {
	return skbh.semStore.SearchMemory(query, limit)
}

// SearchHistory performs semantic search on the history
func (skbh *SalesKnowledgeBaseHandler) SearchHistory(query string, limit int) ([]memory.MemorySearchResult, error) {
	return skbh.semStore.SearchHistory(query, limit)
}