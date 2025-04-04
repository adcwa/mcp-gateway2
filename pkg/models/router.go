package models

import (
	"time"
)

// Router represents a routing configuration
type Router struct {
	ID          string    `json:"id"`
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description"`
	Rules       []Rule    `json:"rules"`
	Version     int       `json:"version"`
	Status      string    `json:"status" binding:"oneof=active inactive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Rule represents a routing rule
type Rule struct {
	ID         string      `json:"id"`
	Path       string      `json:"path" binding:"required"` // Path pattern (e.g., /mcp-server/{name})
	TargetType string      `json:"targetType" binding:"required,oneof=mcp-server http-backend"`
	TargetID   string      `json:"targetId" binding:"required"` // ID of the MCP Server or HTTP backend
	Priority   int         `json:"priority"`                    // Higher priority rules are evaluated first
	Conditions []Condition `json:"conditions,omitempty"`
}

// Condition represents a condition for a routing rule
type Condition struct {
	Type     string `json:"type" binding:"required,oneof=header query path method"`  // Type of condition
	Name     string `json:"name" binding:"required"`                                 // Name of the header, query param, or path param
	Operator string `json:"operator" binding:"required,oneof=eq neq contains regex"` // Operator for comparison
	Value    string `json:"value" binding:"required"`                                // Value to compare against
}
