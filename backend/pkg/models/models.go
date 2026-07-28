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
	CurrentPage         int    `json:"current_page"`
	PageSize            int    `json:"page_size"`
	TotalItems          int64  `json:"total_items"`
	TotalPages          int    `json:"total_pages"`
	TransliteratedQuery string `json:"transliterated_query,omitempty"`
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
	SectionNumberAndName     string  `json:"section_number_and_name,omitempty"`
	PostOffice               string  `json:"post_office,omitempty"`
	PoliceStation            string  `json:"police_station,omitempty"`
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

// PartDetails structure for complete booth & section metadata
type PartDetails struct {
	AssemblyConstituency  string   `json:"assembly_constituency"`
	PartNumber            int64    `json:"part_number"`
	PollingStationName    string   `json:"polling_station_name"`
	PollingStationAddress string   `json:"polling_station_address"`
	TownVillage           string   `json:"town_village,omitempty"`
	Tehsil                string   `json:"tehsil,omitempty"`
	District              string   `json:"district,omitempty"`
	PinCode               string   `json:"pin_code,omitempty"`
	PostOffice            string   `json:"post_office,omitempty"`
	PoliceStation         string   `json:"police_station,omitempty"`
	TotalVoters           int64    `json:"total_voters"`
	Sections              []string `json:"sections"`
	SectionNumberAndName  string   `json:"section_number_and_name,omitempty"`
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
	HindiQuery           string `json:"hindi_query,omitempty"`
	AssemblyConstituency string `json:"assembly_constituency"`
	Gender               string `json:"gender"`
	MinAge               int    `json:"min_age"`
	MaxAge               int    `json:"max_age"`
	TownVillage          string `json:"town_village"`
	PartNumber           int64  `json:"part_number"`
	SectionNumberAndName string `json:"section_number_and_name"`
	SortBy               string `json:"sort_by"`
	SortOrder            string `json:"sort_order"`
	Page                 int    `json:"page"`
	Limit                int    `json:"limit"`
	UseAI                *bool  `json:"use_ai,omitempty"`
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

// UserRole defines application roles
type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleGuest UserRole = "guest"
)

// User credentials and context
type User struct {
	Username    string   `json:"username"`
	Role        UserRole `json:"role"`
	Permissions []string `json:"permissions"`
}

// LoginRequest payload
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse payload
type LoginResponse struct {
	Token string   `json:"token"`
	User  User     `json:"user"`
	Role  UserRole `json:"role"`
}

// SQLRequest payload for raw query execution
type SQLRequest struct {
	SQL string `json:"sql"`
}

// SQLResult payload for query execution output
type SQLResult struct {
	Columns         []string        `json:"columns"`
	Rows            [][]interface{} `json:"rows"`
	RowCount        int             `json:"row_count"`
	ExecutionTimeMS float64         `json:"execution_time_ms"`
}

// GroupByRequest payload
type GroupByRequest struct {
	Field    string `json:"field"`     // "full_name", "relative_name", "gender", "assembly_constituency", "town_village", "age_group"
	Limit    int    `json:"limit"`     // default 20
	MinCount int    `json:"min_count"` // default 1
	Sort     string `json:"sort"`      // "desc" or "asc"
}

// GroupCountItem holds grouped count breakdown
type GroupCountItem struct {
	Value      string  `json:"value"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// GroupByResult payload
type GroupByResult struct {
	Field      string           `json:"field"`
	TotalItems int64            `json:"total_items"`
	Groups     []GroupCountItem `json:"groups"`
}

// GeoNearbyRequest payload for proximity queries
type GeoNearbyRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	RadiusKM  float64 `json:"radius_km"` // Search radius in kilometers
	Limit     int     `json:"limit"`
}

// GeoPollingStationResult extends PollingStation with calculated distance
type GeoPollingStationResult struct {
	PollingStation PollingStation `json:"polling_station"`
	DistanceKM     float64        `json:"distance_km"`
	DistanceMeters float64        `json:"distance_meters"`
}

// GeoVoterResult extends Voter with calculated distance
type GeoVoterResult struct {
	Voter          Voter   `json:"voter"`
	DistanceKM     float64 `json:"distance_km"`
	DistanceMeters float64 `json:"distance_meters"`
}

// GeoDistanceResult details distance between two coordinates
type GeoDistanceResult struct {
	Lat1           float64 `json:"lat1"`
	Lng1           float64 `json:"lng1"`
	Lat2           float64 `json:"lat2"`
	Lng2           float64 `json:"lng2"`
	DistanceKM     float64 `json:"distance_km"`
	DistanceMiles  float64 `json:"distance_miles"`
	DistanceMeters float64 `json:"distance_meters"`
}

// SearchLog represents a stored search query and its results metadata
type SearchLog struct {
	ID           string    `json:"id"`
	Query        string    `json:"query"`
	SearchType   string    `json:"search_type"` // "name", "epic_no", "house_no", "general"
	Filters      string    `json:"filters,omitempty"`
	TotalResults int64     `json:"total_results"`
	TopResults   string    `json:"top_results,omitempty"` // JSON representation of top matching voters
	IPAddress    string    `json:"ip_address,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// SaveSearchRequest payload for explicitly saving search queries
type SaveSearchRequest struct {
	Query        string `json:"query"`
	SearchType   string `json:"search_type,omitempty"`
	Filters      string `json:"filters,omitempty"`
	TotalResults int64  `json:"total_results,omitempty"`
	TopResults   string `json:"top_results,omitempty"`
}

