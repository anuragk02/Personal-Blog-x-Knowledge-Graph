package models

import "time"

// Request models (used for creating entities - no ID field)
type NarrativeRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type SystemRequest struct {
	Name                string `json:"name"`
	BoundaryDescription string `json:"boundaryDescription,omitempty"`
}

type StockRequest struct {
	Name        string `json:"name"`
	// FormalConcept string `json:"formal_concept,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"` // "qualitative" or "quantitative"
}

type FlowRequest struct {
	Name        string `json:"name"`
	// FormalConcept string `json:"formal_concept,omitempty"`
	Description string `json:"description,omitempty"`
}

// Response models (full entities with IDs)
type Narrative struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

type System struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	BoundaryDescription string    `json:"boundaryDescription,omitempty"`
	CreatedAt           time.Time `json:"createdAt"`
}

type Stock struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	// FormalConcept string    `json:"formal_concept,omitempty"`
	Description string    `json:"description,omitempty"`
	Type        string    `json:"type"` // "qualitative" or "quantitative"
	CreatedAt   time.Time `json:"createdAt"`
}

type Flow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	// FormalConcept string    `json:"formal_concept,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}


// Relationships
type Describes struct {
	NarrativeID string `json:"narrativeId"` // From
	SystemID    string `json:"systemId"`    // To
}

type Constitutes struct {
	SystemID    string `json:"systemId"`    // To
	SubsystemID string `json:"subsystemId"` // From
}

type DescribesStatic struct {
	SystemID string `json:"systemId"` // To
	StockID  string `json:"stockId"`  // From
}

type DescribesDynamic struct {
	SystemID string `json:"systemId"` // To
	FlowID   string `json:"flowId"`   // From
}

type Changes struct {
	FlowID   string  `json:"flowId"`  // From
	StockID  string  `json:"stockId"` // To
	Polarity float32 `json:"polarity"` // +1 or -1
}

type CausalLink struct {
	FromID    string    `json:"fromId"`
	FromType  string    `json:"fromType"` // "Stock" or "Flow"
	ToID      string    `json:"toId"`
	ToType    string    `json:"toType"`  // "Stock" or "Flow"
	Question  string    `json:"question"` // The specific question linking them
	CuriosityScore float32   `json:"curiosityScore"`
}

type AnalyzeNarrativeRequest struct {
	NarrativeID string `json:"id"`
}

type LLMAction struct {
    FunctionName string                 `json:"function_name"`
    Parameters   map[string]interface{} `json:"parameters"`
}
type LLMResponse struct {
    Actions []LLMAction `json:"actions"`
}