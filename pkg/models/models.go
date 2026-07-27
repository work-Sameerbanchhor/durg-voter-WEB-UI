package models

import "time"

// Standard API Response wrapper
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Meta      *Pagination `json:"meta,omitempty"`
	Error     string      `json:"error,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
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
	Status        string       `json:"status"`
	Database      string       `json:"database"`
	DatabasePath  string       `json:"database_path"`
	Uptime        string       `json:"uptime"`
	Timestamp     time.Time    `json:"timestamp"`
	GoVersion     string       `json:"go_version"`
	AppName       string       `json:"app_name"`
	Version       string       `json:"version"`
	MemoryAllocMB float64      `json:"memory_alloc_mb"`
	Goroutines    int          `json:"goroutines"`
}

// Voter structure representing electoral data from DuckDB
type Voter struct {
	EPICNo                   string  `json:"epic_no"`
	FullName                 string  `json:"full_name"`
	FullNameHindi            string  `json:"full_name_hindi,omitempty"`
	RelativeName             string  `json:"relative_name"`
	RelativeNameHindi        string  `json:"relative_name_hindi,omitempty"`
	Gender                   string  `json:"gender"`
	GenderHindi              string  `json:"gender_hindi,omitempty"`
	Age                      int     `json:"age"`
	HouseNo                  string  `json:"house_no"`
	PollingStationName       string  `json:"polling_station_name"`
	PollingStationAddress    string  `json:"polling_station_address,omitempty"`
	PartNumber               int64   `json:"part_number"`
	AssemblyConstituency     string  `json:"assembly_constituency"`
	AssemblyConstituencyFull string  `json:"assembly_constituency_full,omitempty"`
	ParliamentaryConstituency string `json:"parliamentary_constituency,omitempty"`
	TownVillage              string  `json:"town_village,omitempty"`
	Tehsil                   string  `json:"tehsil,omitempty"`
	District                 string  `json:"district"`
	PinCode                  string  `json:"pin_code,omitempty"`
	Latitude                 float64 `json:"latitude,omitempty"`
	Longitude                float64 `json:"longitude,omitempty"`
}

// PollingStation details
type PollingStation struct {
	AssemblyConstituency  string  `json:"assembly_constituency"`
	PartNumber            int64   `json:"part_number"`
	PollingStationName    string  `json:"polling_station_name"`
	PollingStationAddress string  `json:"polling_station_address"`
	TownVillage           string  `json:"town_village,omitempty"`
	Tehsil                string  `json:"tehsil,omitempty"`
	District              string  `json:"district"`
	PinCode               string  `json:"pin_code,omitempty"`
	Latitude              float64 `json:"latitude,omitempty"`
	Longitude             float64 `json:"longitude,omitempty"`
	TotalVoters           int64   `json:"total_voters"`
	TotalMaleVoters       int64   `json:"total_male_voters"`
	TotalFemaleVoters     int64   `json:"total_female_voters"`
}

// ConstituencySummary details for assembly constituencies
type ConstituencySummary struct {
	AssemblyConstituency string `json:"assembly_constituency"`
	TotalVoters          int64  `json:"total_voters"`
	MaleVoters           int64  `json:"male_voters"`
	FemaleVoters         int64  `json:"female_voters"`
	OtherVoters          int64  `json:"other_voters"`
	TotalBooths          int64  `json:"total_booths"`
}

// SearchFilter payload for query requests
type SearchFilter struct {
	Query                string `json:"query"`
	AssemblyConstituency string `json:"assembly_constituency"`
	Gender               string `json:"gender"`
	MinAge               int    `json:"min_age"`
	MaxAge               int    `json:"max_age"`
	TownVillage          string `json:"town_village"`
	PartNumber           int64  `json:"part_number"`
	SortBy               string `json:"sort_by"`
	SortOrder            string `json:"sort_order"`
	Page                 int    `json:"page"`
	Limit                int    `json:"limit"`
}

// StatsSummary details overall electorate statistics
type StatsSummary struct {
	TotalVoters       int64            `json:"total_voters"`
	MaleVoters        int64            `json:"male_voters"`
	FemaleVoters      int64            `json:"female_voters"`
	OtherVoters       int64            `json:"other_voters"`
	TotalBooths       int64            `json:"total_booths"`
	AssemblyBreakdown map[string]int64 `json:"assembly_breakdown"`
	Engine            string           `json:"engine"`
}
