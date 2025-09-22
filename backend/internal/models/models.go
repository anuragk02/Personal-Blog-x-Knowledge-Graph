package models

import "time"

// Nodes
type Narrative struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type System struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	BoundaryDescription  string    `json:"boundary_description,omitempty"`
	Type                 string    `json:"type,omitempty"` // Physical, Biological, Social, etc.
	CreatedAt            time.Time `json:"created_at"`	// Discovery
}

type Stock struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`	// Discovery
}

type Flow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   time.Time `json:"created_at"`	// Discovery
}

type QuestionData struct {
    ID          string    `json:"id"`
    Content     string    `json:"content"`
    Status      string    `json:"status"` // unresolved, resolved
    Type        string    `json:"type"` // behavioral, causal, mechanism
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at,omitempty"`
}


// Relationships
type Describes struct {
	NarrativeID string `json:"narrative_id"` // From
	SystemID    string `json:"system_id"`    // To
}

type Constitutes struct {
	SystemID    string `json:"system_id"`	// To
	SubsystemID string `json:"subsystem_id"` // From
}

type DescribesStatic struct {
	SystemID    string `json:"system_id"`	// To
	StockID     string `json:"stock_id"` // From
}

type DescribesDynamic struct {
	SystemID    string `json:"system_id"`	// To
	FlowID     string `json:"flow_id"` // From
}

type Changes struct {
	FlowID  string `json:"flow_id"`  // From
	StockID string `json:"stock_id"` // To
	Polarity float32    `json:"polarity"` // +1 or -1
}

type Raises struct {
	NarrativeID string `json:"narrative_id"` // From
	CausalLinkID string `json:"causal_link_id"` // To
	QuestionID  string `json:"question_id"`
}

type Resolves struct {
	QuestionID   string `json:"question_id"`
	NarrativeID  string `json:"narrative_id"` // From
	CausalLinkID string `json:"causal_link_id"` // To
}

type CausalLink struct {
    FromID      string     `json:"from_id"`
    FromType    string     `json:"from_type"`
    ToID        string     `json:"to_id"`
    ToType      string     `json:"to_type"`
    Polarity    float32    `json:"polarity"`
    Confidence  float32    `json:"confidence"`	// 0 to 1
    Questions   []QuestionData `json:"questions,omitempty"`
    Resolved    []QuestionData `json:"resolved_questions,omitempty"`
    StockCount  int        `json:"stock_count"`  // Number of intermediate stocks discovered
    FlowCount   int        `json:"flow_count"`   // Number of intermediate flows discovered
    CreatedAt   time.Time  `json:"created_at"`
}