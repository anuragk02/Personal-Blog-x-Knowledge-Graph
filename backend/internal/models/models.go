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
	Name string `json:"name"`
	// FormalConcept string `json:"formal_concept,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"` // "qualitative" or "quantitative"
}

type FlowRequest struct {
	Name string `json:"name"`
	// FormalConcept string `json:"formal_concept,omitempty"`
	Description string `json:"description,omitempty"`
}

// Response models (full entities with IDs)
type Narrative struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	Extrapolated bool      `json:"extrapolated"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
}

type System struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	BoundaryDescription string    `json:"boundaryDescription,omitempty"`
	Embedding           []float32 `json:"embedding,omitempty"`
	Embedded            bool      `json:"embedded"`                     // Tracks if embeddings are present
	Consolidated        bool      `json:"consolidated"`                 // For other consolidation process
	ConsolidationScore  int       `json:"consolidationScore"`           // Number of nodes consolidated into this one
	LastConsolidatedAt  time.Time `json:"lastConsolidatedAt,omitempty"` // When last consolidation happened
	CreatedAt           time.Time `json:"createdAt"`
}

type Stock struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description,omitempty"`
	Type               string    `json:"type"` // "qualitative" or "quantitative"
	Embedding          []float32 `json:"embedding,omitempty"`
	Embedded           bool      `json:"embedded"`                     // Tracks if embeddings are present
	Consolidated       bool      `json:"consolidated"`                 // For other consolidation process
	ConsolidationScore int       `json:"consolidationScore"`           // Number of nodes consolidated into this one
	LastConsolidatedAt time.Time `json:"lastConsolidatedAt,omitempty"` // When last consolidation happened
	CreatedAt          time.Time `json:"createdAt"`
}

type Flow struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description,omitempty"`
	Embedding          []float32 `json:"embedding,omitempty"`
	Embedded           bool      `json:"embedded"`                     // Tracks if embeddings are present
	Consolidated       bool      `json:"consolidated"`                 // For other consolidation process
	ConsolidationScore int       `json:"consolidationScore"`           // Number of nodes consolidated into this one
	LastConsolidatedAt time.Time `json:"lastConsolidatedAt,omitempty"` // When last consolidation happened
	CreatedAt          time.Time `json:"createdAt"`
}

// Relationships
type Describes struct {
	NarrativeID        string `json:"narrativeId"`        // From
	SystemID           string `json:"systemId"`           // To
	Consolidated       bool   `json:"consolidated"`       // For consolidation process
	ConsolidationScore int    `json:"consolidationScore"` // Number of relationships consolidated
}

type Constitutes struct {
	SystemID           string `json:"systemId"`           // To
	SubsystemID        string `json:"subsystemId"`        // From
	Consolidated       bool   `json:"consolidated"`       // For consolidation process
	ConsolidationScore int    `json:"consolidationScore"` // Number of relationships consolidated
}

type DescribesStatic struct {
	SystemID           string `json:"systemId"`           // To
	StockID            string `json:"stockId"`            // From
	Consolidated       bool   `json:"consolidated"`       // For consolidation process
	ConsolidationScore int    `json:"consolidationScore"` // Number of relationships consolidated
}

type DescribesDynamic struct {
	SystemID           string `json:"systemId"`           // To
	FlowID             string `json:"flowId"`             // From
	Consolidated       bool   `json:"consolidated"`       // For consolidation process
	ConsolidationScore int    `json:"consolidationScore"` // Number of relationships consolidated
}

type Changes struct {
	FlowID             string  `json:"flowId"`             // From
	StockID            string  `json:"stockId"`            // To
	Polarity           float32 `json:"polarity"`           // +1 or -1
	Consolidated       bool    `json:"consolidated"`       // For consolidation process
	ConsolidationScore int     `json:"consolidationScore"` // Number of relationships consolidated
}

type CausalLink struct {
	FromID             string  `json:"fromId"`
	FromType           string  `json:"fromType"` // "Stock" or "Flow"
	ToID               string  `json:"toId"`
	ToType             string  `json:"toType"`   // "Stock" or "Flow"
	Question           string  `json:"question"` // The specific question linking them
	CuriosityScore     float32 `json:"curiosityScore"`
	Consolidated       bool    `json:"consolidated"`       // For consolidation process
	ConsolidationScore int     `json:"consolidationScore"` // Number of relationships consolidated
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

// Consolidation workflow data structures
type NodeMatch struct {
	UnconsolidatedID string  `json:"unconsolidatedId"`
	ConsolidatedID   string  `json:"consolidatedId"`
	NodeType         string  `json:"nodeType"` // "system", "stock", "flow"
	SimilarityScore  float64 `json:"similarityScore"`
	NewName          string  `json:"newName,omitempty"`        // Synthesized name
	NewDescription   string  `json:"newDescription,omitempty"` // Synthesized description
}

type RelationshipConsolidation struct {
	RelationType     string                 `json:"relationType"` // "DESCRIBES", "CONSTITUTES", etc.
	FromID           string                 `json:"fromId"`
	ToID             string                 `json:"toId"`
	ConsolidatedFrom string                 `json:"consolidatedFrom"` // Mapped consolidated node ID
	ConsolidatedTo   string                 `json:"consolidatedTo"`   // Mapped consolidated node ID
	Properties       map[string]interface{} `json:"properties"`       // Additional relationship properties
}
