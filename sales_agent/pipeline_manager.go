package sales_agent

import (
	"fmt"
	"time"
)

// SalesPipelineManager manages the sales pipeline and stage transitions
type SalesPipelineManager struct {
	leads map[string]*LeadProfile
}

// LeadProfile represents a customer's profile and pipeline state
type LeadProfile struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Company          string                 `json:"company"`
	Email            string                 `json:"email"`
	Phone            string                 `json:"phone"`
	JobTitle         string                 `json:"job_title"`
	Industry         string                 `json:"industry"`
	CompanySize      string                 `json:"company_size"` // Small, Medium, Large, Enterprise
	CurrentStage     SalesStage             `json:"current_stage"`
	PreviousStages   []SalesStage           `json:"previous_stages"`
	EngagementLevel  int                    `json:"engagement_level"` // 1-5 scale
	LastContactDate  time.Time              `json:"last_contact_date"`
	NextActionDate   time.Time              `json:"next_action_date"`
	Notes            []string               `json:"notes"`
	InteractionCount int                    `json:"interaction_count"`
	EstimatedValue   float64                `json:"estimated_value"`
	CloseProbability float64                `json:"close_probability"` // 0.0 to 1.0
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// NewSalesPipelineManager creates a new sales pipeline manager
func NewSalesPipelineManager() *SalesPipelineManager {
	return &SalesPipelineManager{
		leads: make(map[string]*LeadProfile),
	}
}

// CreateLead creates a new lead profile in the pipeline
func (spm *SalesPipelineManager) CreateLead(id, name, company, email string) error {
	if _, exists := spm.leads[id]; exists {
		return fmt.Errorf("lead with ID %s already exists", id)
	}

	lead := &LeadProfile{
		ID:               id,
		Name:             name,
		Company:          company,
		Email:            email,
		CurrentStage:     NewContact,
		PreviousStages:   []SalesStage{},
		EngagementLevel:  1,
		LastContactDate:  time.Now(),
		NextActionDate:   time.Now().AddDate(0, 0, 3), // Default follow-up in 3 days
		Notes:            []string{},
		InteractionCount: 0,
		EstimatedValue:   0,
		CloseProbability: 0.1, // Default low probability for new contacts
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Metadata:         make(map[string]interface{}),
	}

	spm.leads[id] = lead
	return nil
}

// GetLead retrieves a lead profile by ID
func (spm *SalesPipelineManager) GetLead(id string) (*LeadProfile, bool) {
	lead, exists := spm.leads[id]
	return lead, exists
}

// UpdateLeadStage updates the stage of a lead in the pipeline
func (spm *SalesPipelineManager) UpdateLeadStage(leadID string, newStage SalesStage, reason string) error {
	lead, exists := spm.leads[leadID]
	if !exists {
		return fmt.Errorf("lead with ID %s not found", leadID)
	}

	// Record previous stage
	lead.PreviousStages = append(lead.PreviousStages, lead.CurrentStage)
	lead.CurrentStage = newStage
	lead.UpdatedAt = time.Now()

	// Add note about stage change
	note := fmt.Sprintf("Moved to %s stage: %s", newStage, reason)
	lead.Notes = append(lead.Notes, note)
	lead.InteractionCount++

	// Update close probability based on stage
	lead.CloseProbability = calculateCloseProbability(newStage)

	return nil
}

// UpdateLeadInfo updates lead information
func (spm *SalesPipelineManager) UpdateLeadInfo(leadID string, updates map[string]interface{}) error {
	lead, exists := spm.leads[leadID]
	if !exists {
		return fmt.Errorf("lead with ID %s not found", leadID)
	}

	// Update fields based on provided updates
	if name, ok := updates["name"].(string); ok {
		lead.Name = name
	}
	if company, ok := updates["company"].(string); ok {
		lead.Company = company
	}
	if email, ok := updates["email"].(string); ok {
		lead.Email = email
	}
	if phone, ok := updates["phone"].(string); ok {
		lead.Phone = phone
	}
	if jobTitle, ok := updates["job_title"].(string); ok {
		lead.JobTitle = jobTitle
	}
	if industry, ok := updates["industry"].(string); ok {
		lead.Industry = industry
	}
	if companySize, ok := updates["company_size"].(string); ok {
		lead.CompanySize = companySize
	}
	if engagement, ok := updates["engagement_level"].(int); ok {
		lead.EngagementLevel = engagement
	}
	if estimatedValue, ok := updates["estimated_value"].(float64); ok {
		lead.EstimatedValue = estimatedValue
	}
	if metadata, ok := updates["metadata"].(map[string]interface{}); ok {
		for k, v := range metadata {
			lead.Metadata[k] = v
		}
	}

	lead.UpdatedAt = time.Now()
	return nil
}

// AddNote adds a note to a lead's profile
func (spm *SalesPipelineManager) AddNote(leadID, note string) error {
	lead, exists := spm.leads[leadID]
	if !exists {
		return fmt.Errorf("lead with ID %s not found", leadID)
	}

	lead.Notes = append(lead.Notes, note)
	lead.UpdatedAt = time.Now()
	return nil
}

// GetLeadsByStage returns all leads in a specific stage
func (spm *SalesPipelineManager) GetLeadsByStage(stage SalesStage) []*LeadProfile {
	var result []*LeadProfile
	for _, lead := range spm.leads {
		if lead.CurrentStage == stage {
			result = append(result, lead)
		}
	}
	return result
}

// GetLeadsByEngagement returns leads filtered by engagement level
func (spm *SalesPipelineManager) GetLeadsByEngagement(minLevel int) []*LeadProfile {
	var result []*LeadProfile
	for _, lead := range spm.leads {
		if lead.EngagementLevel >= minLevel {
			result = append(result, lead)
		}
	}
	return result
}

// GetQualifiedLeads returns leads that meet qualification criteria (BANT)
func (spm *SalesPipelineManager) GetQualifiedLeads() []*LeadProfile {
	var result []*LeadProfile
	for _, lead := range spm.leads {
		// Basic qualification: Engagement level 3+, stage beyond new contact, has contact info
		if lead.EngagementLevel >= 3 &&
		   lead.CurrentStage != NewContact &&
		   lead.Email != "" &&
		   lead.Company != "" {
			result = append(result, lead)
		}
	}
	return result
}

// GetPipelineReport generates a report of the sales pipeline
func (spm *SalesPipelineManager) GetPipelineReport() PipelineReport {
	report := PipelineReport{
		TotalLeads:    len(spm.leads),
		StageCounts:   make(map[SalesStage]int),
		AverageValue:  0,
		TotalValue:    0,
		QualifiedLeads: 0,
	}

	var totalValue float64
	for _, lead := range spm.leads {
		report.StageCounts[lead.CurrentStage]++

		if lead.EngagementLevel >= 3 &&
		   lead.CurrentStage != NewContact &&
		   lead.Email != "" &&
		   lead.Company != "" {
			report.QualifiedLeads++
		}

		totalValue += lead.EstimatedValue
	}

	if report.TotalLeads > 0 {
		report.AverageValue = totalValue / float64(report.TotalLeads)
	}
	report.TotalValue = totalValue

	return report
}

// PipelineReport represents a summary report of the sales pipeline
type PipelineReport struct {
	TotalLeads     int                    `json:"total_leads"`
	StageCounts    map[SalesStage]int     `json:"stage_counts"`
	AverageValue   float64                `json:"average_value"`
	TotalValue     float64                `json:"total_value"`
	QualifiedLeads int                    `json:"qualified_leads"`
}

// calculateCloseProbability estimates the close probability based on stage
func calculateCloseProbability(stage SalesStage) float64 {
	switch stage {
	case NewContact:
		return 0.1
	case Discovery:
		return 0.2
	case Presentation:
		return 0.4
	case Negotiation:
		return 0.7
	case Close:
		return 0.9
	case QualifiedLead:
		return 0.3
	default:
		return 0.1
	}
}

// AdvanceLead moves a lead to the next appropriate stage
func (spm *SalesPipelineManager) AdvanceLead(leadID, reason string) error {
	lead, exists := spm.leads[leadID]
	if !exists {
		return fmt.Errorf("lead with ID %s not found", leadID)
	}

	var nextStage SalesStage
	switch lead.CurrentStage {
	case NewContact:
		nextStage = Discovery
	case Discovery:
		nextStage = Presentation
	case Presentation:
		nextStage = Negotiation
	case Negotiation:
		nextStage = Close
	case QualifiedLead:
		nextStage = Presentation // From qualified lead, go to presentation
	default:
		// If already in Close stage, no advancement needed
		if lead.CurrentStage == Close {
			return fmt.Errorf("lead is already in closing stage")
		}
		// For other stages or unknown stages, move to next logical stage
		nextStage = Discovery
	}

	return spm.UpdateLeadStage(leadID, nextStage, reason)
}

// SetLeadValue sets the estimated value of a lead
func (spm *SalesPipelineManager) SetLeadValue(leadID string, value float64) error {
	lead, exists := spm.leads[leadID]
	if !exists {
		return fmt.Errorf("lead with ID %s not found", leadID)
	}

	lead.EstimatedValue = value
	lead.UpdatedAt = time.Now()
	return nil
}