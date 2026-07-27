package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"

	"durg-voter-api/pkg/db"
	"durg-voter-api/pkg/models"
)

type VoterRepository interface {
	GetStats(ctx context.Context) (*models.StatsSummary, error)
	ListVoters(ctx context.Context, filter models.SearchFilter) ([]models.Voter, *models.Pagination, error)
	GetVoterByEPIC(ctx context.Context, epicNo string) (*models.Voter, error)
	ListPollingStations(ctx context.Context, filter models.SearchFilter) ([]models.PollingStation, *models.Pagination, error)
	GetPollingStation(ctx context.Context, assembly string, partNo int64) (*models.PollingStation, error)
	ListConstituencies(ctx context.Context) ([]models.ConstituencySummary, error)
}

type duckDBVoterRepository struct {
	duckDB *db.DuckDB
}

func NewVoterRepository(duckDB *db.DuckDB) VoterRepository {
	return &duckDBVoterRepository{duckDB: duckDB}
}

func (r *duckDBVoterRepository) GetStats(ctx context.Context) (*models.StatsSummary, error) {
	queryStats := `
		SELECT 
			COUNT(*) AS total_voters,
			COUNT(CASE WHEN LOWER(gender_english) = 'male' THEN 1 END) AS male_voters,
			COUNT(CASE WHEN LOWER(gender_english) = 'female' THEN 1 END) AS female_voters,
			COUNT(CASE WHEN LOWER(gender_english) NOT IN ('male', 'female') THEN 1 END) AS other_voters
		FROM voters;
	`

	var stats models.StatsSummary
	err := r.duckDB.DB.QueryRowContext(ctx, queryStats).Scan(
		&stats.TotalVoters,
		&stats.MaleVoters,
		&stats.FemaleVoters,
		&stats.OtherVoters,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch voter stats: %w", err)
	}

	var totalBooths int64
	err = r.duckDB.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM polling_stations;").Scan(&totalBooths)
	if err != nil {
		stats.TotalBooths = 0
	} else {
		stats.TotalBooths = totalBooths
	}

	queryAssembly := `
		SELECT assembly_constituency, COUNT(*) 
		FROM voters 
		WHERE assembly_constituency IS NOT NULL AND assembly_constituency != ''
		GROUP BY assembly_constituency;
	`
	rows, err := r.duckDB.DB.QueryContext(ctx, queryAssembly)
	if err == nil {
		defer rows.Close()
		breakdown := make(map[string]int64)
		for rows.Next() {
			var ac string
			var cnt int64
			if err := rows.Scan(&ac, &cnt); err == nil {
				breakdown[ac] = cnt
			}
		}
		stats.AssemblyBreakdown = breakdown
	}

	stats.Engine = "DuckDB 1.5 Vectorized Execution Engine"
	return &stats, nil
}

func (r *duckDBVoterRepository) ListVoters(ctx context.Context, filter models.SearchFilter) ([]models.Voter, *models.Pagination, error) {
	var conditions []string
	var args []interface{}

	if filter.Query != "" {
		q := "%" + strings.ToLower(filter.Query) + "%"
		conditions = append(conditions, "(LOWER(voter_id) LIKE ? OR LOWER(voter_name_english) LIKE ? OR LOWER(relative_name_english) LIKE ? OR LOWER(house_number) LIKE ?)")
		args = append(args, q, q, q, q)
	}

	if filter.AssemblyConstituency != "" {
		conditions = append(conditions, "LOWER(assembly_constituency) LIKE ?")
		args = append(args, "%"+strings.ToLower(filter.AssemblyConstituency)+"%")
	}

	if filter.Gender != "" {
		conditions = append(conditions, "LOWER(gender_english) = ?")
		args = append(args, strings.ToLower(filter.Gender))
	}

	if filter.MinAge > 0 {
		conditions = append(conditions, "age >= ?")
		args = append(args, filter.MinAge)
	}

	if filter.MaxAge > 0 {
		conditions = append(conditions, "age <= ?")
		args = append(args, filter.MaxAge)
	}

	if filter.TownVillage != "" {
		conditions = append(conditions, "LOWER(town_village) LIKE ?")
		args = append(args, "%"+strings.ToLower(filter.TownVillage)+"%")
	}

	if filter.PartNumber > 0 {
		conditions = append(conditions, "part_number = ?")
		args = append(args, filter.PartNumber)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM voters" + whereClause
	var totalItems int64
	err := r.duckDB.DB.QueryRowContext(ctx, countQuery, args...).Scan(&totalItems)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count voters: %w", err)
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))
	offset := (page - 1) * limit

	orderBy := "voter_name_english ASC"
	if filter.SortBy != "" {
		col := "voter_name_english"
		switch strings.ToLower(filter.SortBy) {
		case "epic", "voter_id", "epic_no":
			col = "voter_id"
		case "age":
			col = "age"
		case "assembly":
			col = "assembly_constituency"
		case "part_number":
			col = "part_number"
		}
		order := "ASC"
		if strings.EqualFold(filter.SortOrder, "desc") {
			order = "DESC"
		}
		orderBy = fmt.Sprintf("%s %s", col, order)
	}

	selectQuery := fmt.Sprintf(`
		SELECT 
			COALESCE(voter_id, ''),
			COALESCE(voter_name_english, ''),
			COALESCE(voter_name_hindi, ''),
			COALESCE(relative_name_english, ''),
			COALESCE(relative_name_hindi, ''),
			COALESCE(gender_english, ''),
			COALESCE(gender_hindi, ''),
			COALESCE(age, 0),
			COALESCE(house_number, ''),
			COALESCE(station_name_loc, ''),
			COALESCE(station_address_loc, ''),
			COALESCE(part_number, 0),
			COALESCE(assembly_constituency, ''),
			COALESCE(assembly_constituency_full, ''),
			COALESCE(parliamentary_constituency, ''),
			COALESCE(town_village, ''),
			COALESCE(tehsil, ''),
			COALESCE(district, ''),
			COALESCE(pin_code, ''),
			COALESCE(latitude, 0.0),
			COALESCE(longitude, 0.0)
		FROM voters %s
		ORDER BY %s
		LIMIT ? OFFSET ?;
	`, whereClause, orderBy)

	queryArgs := append(args, limit, offset)
	rows, err := r.duckDB.DB.QueryContext(ctx, selectQuery, queryArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list voters: %w", err)
	}
	defer rows.Close()

	voters := make([]models.Voter, 0)
	for rows.Next() {
		var v models.Voter
		err := rows.Scan(
			&v.EPICNo,
			&v.FullName,
			&v.FullNameHindi,
			&v.RelativeName,
			&v.RelativeNameHindi,
			&v.Gender,
			&v.GenderHindi,
			&v.Age,
			&v.HouseNo,
			&v.PollingStationName,
			&v.PollingStationAddress,
			&v.PartNumber,
			&v.AssemblyConstituency,
			&v.AssemblyConstituencyFull,
			&v.ParliamentaryConstituency,
			&v.TownVillage,
			&v.Tehsil,
			&v.District,
			&v.PinCode,
			&v.Latitude,
			&v.Longitude,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan voter row: %w", err)
		}
		voters = append(voters, v)
	}

	pagination := &models.Pagination{
		CurrentPage: page,
		PageSize:    limit,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}

	return voters, pagination, nil
}

func (r *duckDBVoterRepository) GetVoterByEPIC(ctx context.Context, epicNo string) (*models.Voter, error) {
	query := `
		SELECT 
			COALESCE(voter_id, ''),
			COALESCE(voter_name_english, ''),
			COALESCE(voter_name_hindi, ''),
			COALESCE(relative_name_english, ''),
			COALESCE(relative_name_hindi, ''),
			COALESCE(gender_english, ''),
			COALESCE(gender_hindi, ''),
			COALESCE(age, 0),
			COALESCE(house_number, ''),
			COALESCE(station_name_loc, ''),
			COALESCE(station_address_loc, ''),
			COALESCE(part_number, 0),
			COALESCE(assembly_constituency, ''),
			COALESCE(assembly_constituency_full, ''),
			COALESCE(parliamentary_constituency, ''),
			COALESCE(town_village, ''),
			COALESCE(tehsil, ''),
			COALESCE(district, ''),
			COALESCE(pin_code, ''),
			COALESCE(latitude, 0.0),
			COALESCE(longitude, 0.0)
		FROM voters
		WHERE LOWER(voter_id) = LOWER(?)
		LIMIT 1;
	`

	var v models.Voter
	err := r.duckDB.DB.QueryRowContext(ctx, query, epicNo).Scan(
		&v.EPICNo,
		&v.FullName,
		&v.FullNameHindi,
		&v.RelativeName,
		&v.RelativeNameHindi,
		&v.Gender,
		&v.GenderHindi,
		&v.Age,
		&v.HouseNo,
		&v.PollingStationName,
		&v.PollingStationAddress,
		&v.PartNumber,
		&v.AssemblyConstituency,
		&v.AssemblyConstituencyFull,
		&v.ParliamentaryConstituency,
		&v.TownVillage,
		&v.Tehsil,
		&v.District,
		&v.PinCode,
		&v.Latitude,
		&v.Longitude,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query voter by EPIC: %w", err)
	}

	return &v, nil
}

func (r *duckDBVoterRepository) ListPollingStations(ctx context.Context, filter models.SearchFilter) ([]models.PollingStation, *models.Pagination, error) {
	var conditions []string
	var args []interface{}

	if filter.Query != "" {
		q := "%" + strings.ToLower(filter.Query) + "%"
		conditions = append(conditions, "(LOWER(polling_station_name) LIKE ? OR LOWER(polling_station_address) LIKE ? OR LOWER(town_village) LIKE ?)")
		args = append(args, q, q, q)
	}

	if filter.AssemblyConstituency != "" {
		conditions = append(conditions, "LOWER(assembly_constituency) LIKE ?")
		args = append(args, "%"+strings.ToLower(filter.AssemblyConstituency)+"%")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	var totalItems int64
	err := r.duckDB.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM polling_stations"+whereClause, args...).Scan(&totalItems)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count polling stations: %w", err)
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))
	offset := (page - 1) * limit

	selectQuery := fmt.Sprintf(`
		SELECT 
			COALESCE(assembly_constituency, ''),
			COALESCE(part_number, 0),
			COALESCE(polling_station_name, ''),
			COALESCE(polling_station_address, ''),
			COALESCE(town_village, ''),
			COALESCE(tehsil, ''),
			COALESCE(district, ''),
			COALESCE(pin_code, ''),
			COALESCE(latitude, 0.0),
			COALESCE(longitude, 0.0),
			COALESCE(total_voters, 0),
			COALESCE(total_male_voters, 0),
			COALESCE(total_female_voters, 0)
		FROM polling_stations %s
		ORDER BY assembly_constituency ASC, part_number ASC
		LIMIT ? OFFSET ?;
	`, whereClause)

	queryArgs := append(args, limit, offset)
	rows, err := r.duckDB.DB.QueryContext(ctx, selectQuery, queryArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list polling stations: %w", err)
	}
	defer rows.Close()

	stations := make([]models.PollingStation, 0)
	for rows.Next() {
		var ps models.PollingStation
		err := rows.Scan(
			&ps.AssemblyConstituency,
			&ps.PartNumber,
			&ps.PollingStationName,
			&ps.PollingStationAddress,
			&ps.TownVillage,
			&ps.Tehsil,
			&ps.District,
			&ps.PinCode,
			&ps.Latitude,
			&ps.Longitude,
			&ps.TotalVoters,
			&ps.TotalMaleVoters,
			&ps.TotalFemaleVoters,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan polling station row: %w", err)
		}
		stations = append(stations, ps)
	}

	pagination := &models.Pagination{
		CurrentPage: page,
		PageSize:    limit,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}

	return stations, pagination, nil
}

func (r *duckDBVoterRepository) GetPollingStation(ctx context.Context, assembly string, partNo int64) (*models.PollingStation, error) {
	query := `
		SELECT 
			COALESCE(assembly_constituency, ''),
			COALESCE(part_number, 0),
			COALESCE(polling_station_name, ''),
			COALESCE(polling_station_address, ''),
			COALESCE(town_village, ''),
			COALESCE(tehsil, ''),
			COALESCE(district, ''),
			COALESCE(pin_code, ''),
			COALESCE(latitude, 0.0),
			COALESCE(longitude, 0.0),
			COALESCE(total_voters, 0),
			COALESCE(total_male_voters, 0),
			COALESCE(total_female_voters, 0)
		FROM polling_stations
		WHERE LOWER(assembly_constituency) = LOWER(?) AND part_number = ?
		LIMIT 1;
	`

	var ps models.PollingStation
	err := r.duckDB.DB.QueryRowContext(ctx, query, assembly, partNo).Scan(
		&ps.AssemblyConstituency,
		&ps.PartNumber,
		&ps.PollingStationName,
		&ps.PollingStationAddress,
		&ps.TownVillage,
		&ps.Tehsil,
		&ps.District,
		&ps.PinCode,
		&ps.Latitude,
		&ps.Longitude,
		&ps.TotalVoters,
		&ps.TotalMaleVoters,
		&ps.TotalFemaleVoters,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query polling station: %w", err)
	}

	return &ps, nil
}

func (r *duckDBVoterRepository) ListConstituencies(ctx context.Context) ([]models.ConstituencySummary, error) {
	query := `
		SELECT 
			assembly_constituency,
			COUNT(*) AS total_voters,
			COUNT(CASE WHEN LOWER(gender_english) = 'male' THEN 1 END) AS male_voters,
			COUNT(CASE WHEN LOWER(gender_english) = 'female' THEN 1 END) AS female_voters,
			COUNT(CASE WHEN LOWER(gender_english) NOT IN ('male', 'female') THEN 1 END) AS other_voters,
			COUNT(DISTINCT part_number) AS total_booths
		FROM voters
		WHERE assembly_constituency IS NOT NULL AND assembly_constituency != ''
		GROUP BY assembly_constituency
		ORDER BY total_voters DESC;
	`

	rows, err := r.duckDB.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query constituencies: %w", err)
	}
	defer rows.Close()

	summaries := make([]models.ConstituencySummary, 0)
	for rows.Next() {
		var cs models.ConstituencySummary
		if err := rows.Scan(&cs.AssemblyConstituency, &cs.TotalVoters, &cs.MaleVoters, &cs.FemaleVoters, &cs.OtherVoters, &cs.TotalBooths); err != nil {
			return nil, fmt.Errorf("failed to scan constituency summary: %w", err)
		}
		summaries = append(summaries, cs)
	}

	return summaries, nil
}
