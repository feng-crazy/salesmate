package sales_agent

// SalesStage represents the stage in the sales pipeline
type SalesStage string

const (
	NewContact    SalesStage = "new_contact"    // 新联系 - Contact just made
	Discovery     SalesStage = "discovery"      // 发现需求 - Understanding customer needs
	Presentation  SalesStage = "presentation"   // 提案 - Presenting solution
	Negotiation   SalesStage = "negotiation"    // 谈判 - Handling objections and terms
	Close         SalesStage = "close"          // 成交 - Finalizing the deal
	Lost          SalesStage = "lost"           // 丢失 - Deal lost
	QualifiedLead SalesStage = "qualified_lead" // 合格线索 - BANT qualified lead
)