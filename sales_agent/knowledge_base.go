package sales_agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SalesKnowledgeBase handles sales-specific knowledge (products, pricing, FAQs)
type SalesKnowledgeBase struct {
	workspace     string
	products      map[string]Product
	pricing       map[string]PriceTier
	faq           map[string]string
	competitors   map[string]CompetitorInfo
}

// Product represents a product in the catalog
type Product struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Features    []string          `json:"features"`
	UseCases    []string          `json:"use_cases"`
	Price       float64           `json:"price"`
	Category    string            `json:"category"`
	Tags        []string          `json:"tags"`
	TechnicalSpecs map[string]string `json:"technical_specs"`
}

// PriceTier represents pricing structure
type PriceTier struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Price       float64           `json:"price"`
	Features    map[string]bool   `json:"features"`
	BillingCycle string           `json:"billing_cycle"` // monthly, annual
	MinQuantity  int              `json:"min_quantity"`
}

// CompetitorInfo represents competitor information
type CompetitorInfo struct {
	Name        string            `json:"name"`
	Strengths   []string          `json:"strengths"`
	Weaknesses  []string          `json:"weaknesses"`
	Differentiators []string       `json:"differentiators"`
	PricingComparison map[string]interface{} `json:"pricing_comparison"`
}

// NewSalesKnowledgeBase creates a new sales knowledge base
func NewSalesKnowledgeBase(workspace string) *SalesKnowledgeBase {
	skb := &SalesKnowledgeBase{
		workspace:   workspace,
		products:    make(map[string]Product),
		pricing:     make(map[string]PriceTier),
		faq:         make(map[string]string),
		competitors: make(map[string]CompetitorInfo),
	}

	// Initialize with sample data
	skb.initializeSampleData()

	// Try to load any saved data
	skb.loadData()

	return skb
}

// initializeSampleData sets up initial sample data
func (skb *SalesKnowledgeBase) initializeSampleData() {
	// Sample products
	skb.products["basic"] = Product{
		ID:          "basic",
		Name:        "Basic Plan",
		Description: "Our entry-level solution for small businesses",
		Features: []string{
			"Up to 5 users",
			"Basic analytics",
			"Email support",
			"Standard reporting",
			"1GB storage",
		},
		UseCases: []string{
			"Small team collaboration",
			"Basic project management",
			"Simple workflow automation",
		},
		Price:    29.00,
		Category: "starter",
		Tags:     []string{"affordable", "entry", "small-business"},
		TechnicalSpecs: map[string]string{
			"Deployment": "Cloud",
			"API Access": "Limited",
			"SSO":        "No",
			"Support":    "Email",
		},
	}

	skb.products["pro"] = Product{
		ID:          "pro",
		Name:        "Professional Plan",
		Description: "Advanced features for growing businesses",
		Features: []string{
			"Up to 25 users",
			"Advanced analytics",
			"Priority support",
			"Custom reporting",
			"10GB storage",
			"API access",
		},
		UseCases: []string{
			"Mid-market teams",
			"Advanced analytics needs",
			"Custom integrations",
		},
		Price:    79.00,
		Category: "professional",
		Tags:     []string{"scalable", "mid-market", "advanced"},
		TechnicalSpecs: map[string]string{
			"Deployment": "Cloud",
			"API Access": "Full",
			"SSO":        "Yes",
			"Support":    "Priority",
		},
	}

	skb.products["enterprise"] = Product{
		ID:          "enterprise",
		Name:        "Enterprise Plan",
		Description: "Fully customizable solution for large organizations",
		Features: []string{
			"Unlimited users",
			"Advanced security",
			"Dedicated support",
			"Custom development",
			"Unlimited storage",
			"Premium API",
			"Single sign-on (SSO)",
			"Advanced reporting",
		},
		UseCases: []string{
			"Large enterprise",
			"Regulated industries",
			"Custom implementations",
		},
		Price:    199.00,
		Category: "enterprise",
		Tags:     []string{"enterprise", "secure", "customizable"},
		TechnicalSpecs: map[string]string{
			"Deployment": "Cloud/On-premise",
			"API Access": "Premium",
			"SSO":        "Yes",
			"Support":    "Dedicated",
		},
	}

	// Sample pricing tiers
	skb.pricing["basic_monthly"] = PriceTier{
		Name:        "Basic Monthly",
		Description: "Monthly billing for the Basic Plan",
		Price:       29.00,
		Features: map[string]bool{
			"Up to 5 users":       true,
			"Basic analytics":     true,
			"Email support":       true,
			"Standard reporting":  true,
			"1GB storage":         true,
			"Advanced features":   false,
			"Priority support":    false,
			"API access":          false,
		},
		BillingCycle: "monthly",
		MinQuantity:  1,
	}

	skb.pricing["pro_annual"] = PriceTier{
		Name:        "Pro Annual",
		Description: "Annual billing for the Pro Plan (2 months free)",
		Price:       799.00, // $79/month * 10 months (2 free)
		Features: map[string]bool{
			"Up to 25 users":       true,
			"Advanced analytics":   true,
			"Priority support":     true,
			"Custom reporting":     true,
			"10GB storage":         true,
			"API access":           true,
			"Advanced features":    true,
			"Dedicated support":    false,
			"Unlimited storage":    false,
		},
		BillingCycle: "annual",
		MinQuantity:  1,
	}

	// Sample FAQ
	skb.faq["pricing"] = "We offer three pricing tiers: Basic ($29/month), Professional ($79/month), and Enterprise ($199/month). Each tier includes increasing levels of features and support. We also offer annual billing discounts."
	skb.faq["free_trial"] = "Yes, we offer a 14-day free trial for all our plans. No credit card required to start your trial."
	skb.faq["setup_time"] = "Most customers are up and running within 24 hours. Our onboarding team will guide you through the setup process and help customize the solution for your needs."
	skb.faq["integration"] = "We support integrations with popular tools including Salesforce, Slack, Microsoft Teams, Google Workspace, and many others. Our API allows for custom integrations."
	skb.faq["support"] = "All plans include email support. Professional and Enterprise plans include priority support with faster response times. Enterprise plans include dedicated account management."
	skb.faq["data_security"] = "We take data security seriously. All data is encrypted in transit and at rest. We comply with GDPR, CCPA, and SOC 2 Type II standards."

	// Sample competitors
	skb.competitors["competitor_a"] = CompetitorInfo{
		Name:      "Competitor A",
		Strengths: []string{"Market leader", "Strong brand recognition", "Extensive feature set"},
		Weaknesses: []string{"Higher price point", "Complex interface", "Steep learning curve"},
		Differentiators: []string{"Better user experience", "More affordable", "Easier to use", "Superior customer support"},
		PricingComparison: map[string]interface{}{
			"our_price":        79.00,
			"competitor_price": 129.00,
			"savings_percent":  39,
		},
	}

	skb.competitors["competitor_b"] = CompetitorInfo{
		Name:      "Competitor B",
		Strengths: []string{"Open source", "Highly customizable", "Active community"},
		Weaknesses: []string{"Requires technical expertise", "Limited out-of-box features", "Support challenges"},
		Differentiators: []string{"Ready-to-use solution", "Professional support", "Continuous updates", "No maintenance overhead"},
		PricingComparison: map[string]interface{}{
			"value_proposition": "We provide a complete, maintained solution with professional support",
			"total_cost_of_ownership": "Lower when factoring in internal resources needed for open-source solution",
		},
	}
}

// loadData loads saved knowledge data from files
func (skb *SalesKnowledgeBase) loadData() error {
	dataDir := filepath.Join(skb.workspace, "sales_data")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return nil // Directory doesn't exist yet, that's fine
	}

	// Load products
	productsFile := filepath.Join(dataDir, "products.json")
	if _, err := os.Stat(productsFile); err == nil {
		content, err := os.ReadFile(productsFile)
		if err == nil {
			var products []Product
			if err := json.Unmarshal(content, &products); err == nil {
				// Convert slice to map
				for _, p := range products {
					skb.products[p.ID] = p
				}
			}
		}
	}

	// Load pricing
	pricingFile := filepath.Join(dataDir, "pricing.json")
	if _, err := os.Stat(pricingFile); err == nil {
		content, err := os.ReadFile(pricingFile)
		if err == nil {
			var pricing []PriceTier
			if err := json.Unmarshal(content, &pricing); err == nil {
				// Convert slice to map
				for _, pt := range pricing {
					skb.pricing[pt.Name] = pt
				}
			}
		}
	}

	// Load FAQ
	faqFile := filepath.Join(dataDir, "faq.json")
	if _, err := os.Stat(faqFile); err == nil {
		content, err := os.ReadFile(faqFile)
		if err == nil {
			if err := json.Unmarshal(content, &skb.faq); err == nil {
				// Success
			}
		}
	}

	return nil
}

// saveData saves knowledge data to files
func (skb *SalesKnowledgeBase) saveData() error {
	dataDir := filepath.Join(skb.workspace, "sales_data")
	os.MkdirAll(dataDir, 0755)

	// Save products
	productsFile := filepath.Join(dataDir, "products.json")
	products := make([]Product, 0, len(skb.products))
	for _, p := range skb.products {
		products = append(products, p)
	}
	productsJSON, _ := json.MarshalIndent(products, "", "  ")
	os.WriteFile(productsFile, productsJSON, 0644)

	// Save pricing
	pricingFile := filepath.Join(dataDir, "pricing.json")
	pricing := make([]PriceTier, 0, len(skb.pricing))
	for _, pt := range skb.pricing {
		pricing = append(pricing, pt)
	}
	pricingJSON, _ := json.MarshalIndent(pricing, "", "  ")
	os.WriteFile(pricingFile, pricingJSON, 0644)

	// Save FAQ
	faqFile := filepath.Join(dataDir, "faq.json")
	faqJSON, _ := json.MarshalIndent(skb.faq, "", "  ")
	os.WriteFile(faqFile, faqJSON, 0644)

	return nil
}

// QueryProduct searches for product information
func (skb *SalesKnowledgeBase) QueryProduct(query string) []Product {
	query = strings.ToLower(query)
	results := []Product{}

	for _, product := range skb.products {
		// Check name, description, features, and tags
		if strings.Contains(strings.ToLower(product.Name), query) ||
		   strings.Contains(strings.ToLower(product.Description), query) {
			results = append(results, product)
			continue
		}

		// Check features
		for _, feature := range product.Features {
			if strings.Contains(strings.ToLower(feature), query) {
				results = append(results, product)
				break
			}
		}

		// Check tags
		for _, tag := range product.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				results = append(results, product)
				break
			}
		}
	}

	return results
}

// QueryPricing searches for pricing information
func (skb *SalesKnowledgeBase) QueryPricing(query string) []PriceTier {
	query = strings.ToLower(query)
	results := []PriceTier{}

	for _, tier := range skb.pricing {
		if strings.Contains(strings.ToLower(tier.Name), query) ||
		   strings.Contains(strings.ToLower(tier.Description), query) {
			results = append(results, tier)
		}
	}

	return results
}

// QueryFAQ searches for FAQ answers
func (skb *SalesKnowledgeBase) QueryFAQ(query string) map[string]string {
	query = strings.ToLower(query)
	results := make(map[string]string)

	for question, answer := range skb.faq {
		if strings.Contains(strings.ToLower(question), query) ||
		   strings.Contains(strings.ToLower(answer), query) {
			results[question] = answer
		}
	}

	return results
}

// GetProductByID returns a product by its ID
func (skb *SalesKnowledgeBase) GetProductByID(id string) (*Product, bool) {
	product, exists := skb.products[id]
	if !exists {
		return nil, false
	}
	return &product, true
}

// GetPricingByName returns pricing info by name
func (skb *SalesKnowledgeBase) GetPricingByName(name string) (*PriceTier, bool) {
	priceTier, exists := skb.pricing[name]
	if !exists {
		return nil, false
	}
	return &priceTier, true
}

// CompareWithCompetitor compares our offering with a competitor
func (skb *SalesKnowledgeBase) CompareWithCompetitor(competitorName string) (*CompetitorInfo, bool) {
	comp, exists := skb.competitors[strings.ToLower(competitorName)]
	if !exists {
		// Try to find partial match
		for name, competitor := range skb.competitors {
			if strings.Contains(strings.ToLower(name), strings.ToLower(competitorName)) ||
			   strings.Contains(strings.ToLower(competitor.Name), strings.ToLower(competitorName)) {
				return &competitor, true
			}
		}
		return nil, false
	}
	return &comp, true
}

// GetAllProducts returns all products
func (skb *SalesKnowledgeBase) GetAllProducts() []Product {
	products := make([]Product, 0, len(skb.products))
	for _, product := range skb.products {
		products = append(products, product)
	}
	return products
}

// GetAllPricing returns all pricing tiers
func (skb *SalesKnowledgeBase) GetAllPricing() []PriceTier {
	pricing := make([]PriceTier, 0, len(skb.pricing))
	for _, tier := range skb.pricing {
		pricing = append(pricing, tier)
	}
	return pricing
}

// ValidateProductInfo checks if the provided product information is accurate
func (skb *SalesKnowledgeBase) ValidateProductInfo(productID, feature string) (bool, string) {
	product, exists := skb.products[productID]
	if !exists {
		return false, fmt.Sprintf("Product with ID '%s' not found in our catalog", productID)
	}

	// Check if feature is in the product's feature list
	featureFound := false
	for _, prodFeature := range product.Features {
		if strings.EqualFold(prodFeature, feature) {
			featureFound = true
			break
		}
	}

	if !featureFound {
		return false, fmt.Sprintf("Feature '%s' is not available in the %s plan. Available features include: %s",
			feature, product.Name, strings.Join(product.Features, ", "))
	}

	return true, ""
}