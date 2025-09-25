package models

import "time"

// Request models (used for creating entities - no ID field)
type NarrativeRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type SystemRequest struct {
	Name                string `json:"name"`
	BoundaryDescription string `json:"boundary_description,omitempty"`
}

type StockRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"` // "qualitative" or "quantitative"
}

type FlowRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type CausalLinkRequest struct {
	FromType string `json:"from_type"` // "Stock" or "Flow"
	FromID   string `json:"from_id"`
	ToType   string `json:"to_type"` // "Stock" or "Flow"
	ToID     string `json:"to_id"`
	Question string `json:"question"` // The specific question linking them
}

// Response models (full entities with IDs)
type Narrative struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type System struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	BoundaryDescription string    `json:"boundary_description,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

type Stock struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Type        string    `json:"type"` // "qualitative" or "quantitative"
	CreatedAt   time.Time `json:"created_at"`
}

type Flow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CausalLink struct {
	FromID    string    `json:"from_id"`
	FromType  string    `json:"from_type"` // "Stock" or "Flow"
	ToID      string    `json:"to_id"`
	ToType    string    `json:"to_type"`  // "Stock" or "Flow"
	Question  string    `json:"question"` // The specific question linking them
	CreatedAt time.Time `json:"created_at"`
}

// Relationships
type Describes struct {
	NarrativeID string `json:"narrative_id"` // From
	SystemID    string `json:"system_id"`    // To
}

type Constitutes struct {
	SystemID    string `json:"system_id"`    // To
	SubsystemID string `json:"subsystem_id"` // From
}

type DescribesStatic struct {
	SystemID string `json:"system_id"` // To
	StockID  string `json:"stock_id"`  // From
}

type DescribesDynamic struct {
	SystemID string `json:"system_id"` // To
	FlowID   string `json:"flow_id"`   // From
}

type Changes struct {
	FlowID   string  `json:"flow_id"`  // From
	StockID  string  `json:"stock_id"` // To
	Polarity float32 `json:"polarity"` // +1 or -1
}

// Pseudo-Functions API Provided to LLM
// `linkNarrativeToSystem(narrative, system_name, boundary_description)`
// `createSystem(name, boundary_description)`
// `createStock(name, description, type)` where `type` is **`StockType.QUALITATIVE`** or **`StockType.QUANTITATIVE`**.
// `createFlow(name, description, )`
// `linkSystemAsSubsystem(parent_system_id, child_system_id)`
// `linkSystemToStock(system_id, stock_id)`
// `linkFlowToSystem(flow_id, system_id)`
// `linkFlowToStock(flow_id, stock_id, polarity)`
// `createCausalLink(from_type, from_id, to_type, to_id, question)`
