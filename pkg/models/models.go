package models

import "time"

// Standard API Response wrapper
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Pagination `json:"meta,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Pagination metadata
type Pagination struct {
	CurrentPage int   `json:"current_page"`
	PageSize    int   `json:"page_size"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
}

// HealthStatus details
type HealthStatus struct {
	Status    string    `json:"status"`
	Uptime    string    `json:"uptime"`
	Timestamp time.Time `json:"timestamp"`
	GoVersion string    `json:"go_version"`
	AppName   string    `json:"app_name"`
	Version   string    `json:"version"`
}

// Voter structure representing electoral data
type Voter struct {
	EPICNo              string `json:"epic_no"`
	FullName            string `json:"full_name"`
	RelativeName        string `json:"relative_name"`
	RelationType        string `json:"relation_type"`
	Gender              string `json:"gender"`
	Age                 int    `json:"age"`
	HouseNo             string `json:"house_no"`
	PollingStationName  string `json:"polling_station_name"`
	PollingStationNo    int    `json:"polling_station_no"`
	AssemblyConstituency string `json:"assembly_constituency"`
	AssemblyNo          int    `json:"assembly_no"`
	District            string `json:"district"`
	State               string `json:"state"`
}

// SearchFilter payload for query requests
type SearchFilter struct {
	Query               string `json:"query"`
	AssemblyConstituency string `json:"assembly_constituency"`
	Gender              string `json:"gender"`
	MinAge              int    `json:"min_age"`
	MaxAge              int    `json:"max_age"`
	Page                int    `json:"page"`
	Limit               int    `json:"limit"`
}

// StatsSummary details overall electorate statistics
type StatsSummary struct {
	TotalVoters      int64          `json:"total_voters"`
	MaleVoters       int64          `json:"male_voters"`
	FemaleVoters     int64          `json:"female_voters"`
	OtherVoters      int64          `json:"other_voters"`
	TotalBooths      int            `json:"total_booths"`
	AssemblyBreakdown map[string]int `json:"assembly_breakdown"`
}
