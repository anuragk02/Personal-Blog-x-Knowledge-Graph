package models

import "time"

type Concept struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Summary      string    `json:"summary,omitempty"`
	MasteryLevel int       `json:"mastery_level,omitempty"` // 0-10
	LastReviewed time.Time `json:"last_reviewed,omitempty"`
}

type Claim struct {
	ID              string `json:"id"`
	Text            string `json:"text"`
	ConfidenceScore int    `json:"confidence_score,omitempty"` // 0-10
	IsVerified      bool   `json:"is_verified"`
}

type Source struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // Book, Article, Video, etc.
	Title     string    `json:"title"`
	Author    string    `json:"author,omitempty"`
	URL       string    `json:"url,omitempty"`
	DateAdded time.Time `json:"date_added"`
}

type Question struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Priority int    `json:"priority,omitempty"`
	Status   string `json:"status"` // open, answered
}

type Relationship struct {
	From string                 `json:"from"`
	To   string                 `json:"to"`
	Type string                 `json:"type"`           // DEFINES, INFLUENCES, SUPPORTS, CONTRADICTS, DERIVED_FROM, RAISES
	Data map[string]interface{} `json:"data,omitempty"` // polarity for INFLUENCES, etc.
}
