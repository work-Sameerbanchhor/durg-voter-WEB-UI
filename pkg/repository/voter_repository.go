package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"durg-voter-api/pkg/db"
	"durg-voter-api/pkg/models"
)

type VoterRepository interface {
	GetStats(ctx context.Context) (*models.StatsSummary, error)
	ListVoters(ctx context.Context, filter models.SearchFilter) ([]models.Voter, *models.Pagination, error)
	GetVoterByEPIC(ctx context.Context, epicNo string) (*models.Voter, error)
	ListPollingStations(ctx context.Context, filter models.SearchFilter) ([]models.PollingStation, *models.Pagination, error)
	GetPollingStation(ctx context.Context, assembly string, partNo int64) (*models.PollingStation, error)
	GetPartDetails(ctx context.Context, assembly string, partNo int64) (*models.PartDetails, error)
	ListConstituencies(ctx context.Context) ([]models.ConstituencySummary, error)
	ExecuteSQL(ctx context.Context, sqlQuery string) (*models.SQLResult, error)
	GroupBy(ctx context.Context, req models.GroupByRequest) (*models.GroupByResult, error)
	GetNearbyPollingStations(ctx context.Context, req models.GeoNearbyRequest) ([]models.GeoPollingStationResult, error)
	GetNearbyVoters(ctx context.Context, req models.GeoNearbyRequest) ([]models.GeoVoterResult, error)
	CalculateDistance(lat1, lng1, lat2, lng2 float64) models.GeoDistanceResult
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
		conditions = append(conditions, "(LOWER(voter_id) LIKE ? OR LOWER(voter_name_english) LIKE ? OR LOWER(voter_name_hindi) LIKE ? OR LOWER(relative_name_english) LIKE ? OR LOWER(relative_name_hindi) LIKE ? OR LOWER(house_number) LIKE ? OR LOWER(town_village) LIKE ? OR LOWER(assembly_constituency) LIKE ? OR LOWER(section_number_and_name) LIKE ?)")
		args = append(args, q, q, q, q, q, q, q, q, q)
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

	if filter.SectionNumberAndName != "" {
		conditions = append(conditions, "LOWER(section_number_and_name) LIKE ?")
		args = append(args, "%"+strings.ToLower(filter.SectionNumberAndName)+"%")
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
			COALESCE(section_number_and_name, ''),
			COALESCE(post_office, ''),
			COALESCE(police_station, ''),
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
			&v.SectionNumberAndName,
			&v.PostOffice,
			&v.PoliceStation,
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
			COALESCE(section_number_and_name, ''),
			COALESCE(post_office, ''),
			COALESCE(police_station, ''),
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
		&v.SectionNumberAndName,
		&v.PostOffice,
		&v.PoliceStation,
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

func (r *duckDBVoterRepository) GetPartDetails(ctx context.Context, assembly string, partNo int64) (*models.PartDetails, error) {
	asmPattern := "%" + strings.ReplaceAll(strings.ToLower(assembly), "-", "%") + "%"
	query := `
		SELECT 
			COALESCE(assembly_constituency, ''),
			COALESCE(part_number, 0),
			COALESCE(station_name_loc, ''),
			COALESCE(station_address_loc, ''),
			COALESCE(town_village, ''),
			COALESCE(tehsil, ''),
			COALESCE(district, ''),
			COALESCE(pin_code, ''),
			COALESCE(post_office, ''),
			COALESCE(police_station, ''),
			COUNT(*) AS total_voters
		FROM voters
		WHERE LOWER(assembly_constituency) LIKE ? AND part_number = ?
		GROUP BY 1, 2, 3, 4, 5, 6, 7, 8, 9, 10
		LIMIT 1;
	`
	var pd models.PartDetails
	err := r.duckDB.DB.QueryRowContext(ctx, query, asmPattern, partNo).Scan(
		&pd.AssemblyConstituency,
		&pd.PartNumber,
		&pd.PollingStationName,
		&pd.PollingStationAddress,
		&pd.TownVillage,
		&pd.Tehsil,
		&pd.District,
		&pd.PinCode,
		&pd.PostOffice,
		&pd.PoliceStation,
		&pd.TotalVoters,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query part details: %w", err)
	}

	secQuery := `
		SELECT DISTINCT section_number_and_name
		FROM voters
		WHERE LOWER(assembly_constituency) LIKE ? AND part_number = ?
		  AND section_number_and_name IS NOT NULL AND section_number_and_name != '';
	`
	rows, err := r.duckDB.DB.QueryContext(ctx, secQuery, asmPattern, partNo)
	if err == nil {
		defer rows.Close()
		secMap := make(map[string]bool)
		sections := make([]string, 0)
		var fullSecStr string
		for rows.Next() {
			var rawSec string
			if err := rows.Scan(&rawSec); err == nil {
				if fullSecStr == "" {
					fullSecStr = rawSec
				}
				parts := strings.Split(rawSec, ";")
				for _, p := range parts {
					cleanSec := strings.TrimSpace(p)
					cleanSec = strings.TrimSuffix(cleanSec, ":")
					cleanSec = strings.TrimSuffix(cleanSec, ";")
					cleanSec = strings.TrimSpace(cleanSec)
					if cleanSec != "" && cleanSec != "0" && !secMap[cleanSec] {
						secMap[cleanSec] = true
						sections = append(sections, cleanSec)
					}
				}
			}
		}
		pd.Sections = sections
		pd.SectionNumberAndName = fullSecStr
	}

	return &pd, nil
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

func (r *duckDBVoterRepository) ExecuteSQL(ctx context.Context, sqlQuery string) (*models.SQLResult, error) {
	sqlClean := strings.TrimSpace(sqlQuery)
	if sqlClean == "" {
		return nil, fmt.Errorf("SQL query cannot be empty")
	}

	// Security check: only allow read-only SELECT queries
	upperSQL := strings.ToUpper(sqlClean)
	if !strings.HasPrefix(upperSQL, "SELECT") && !strings.HasPrefix(upperSQL, "WITH") && !strings.HasPrefix(upperSQL, "EXPLAIN") && !strings.HasPrefix(upperSQL, "SHOW") && !strings.HasPrefix(upperSQL, "DESCRIBE") {
		return nil, fmt.Errorf("read-only access: custom SQL execution is restricted to SELECT/analytics queries")
	}

	t0 := time.Now()
	rows, err := r.duckDB.DB.QueryContext(ctx, sqlClean)
	if err != nil {
		return nil, fmt.Errorf("SQL execution error: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve column names: %w", err)
	}

	resultRows := make([][]interface{}, 0)
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, fmt.Errorf("failed scanning row: %w", err)
		}

		rowValues := make([]interface{}, len(cols))
		for i, val := range columns {
			if b, ok := val.([]byte); ok {
				rowValues[i] = string(b)
			} else {
				rowValues[i] = val
			}
		}
		resultRows = append(resultRows, rowValues)

		// Limit maximum rows returned in response to prevent memory overflow
		if len(resultRows) >= 1000 {
			break
		}
	}

	duration := float64(time.Since(t0).Microseconds()) / 1000.0

	return &models.SQLResult{
		Columns:         cols,
		Rows:            resultRows,
		RowCount:        len(resultRows),
		ExecutionTimeMS: duration,
	}, nil
}

func (r *duckDBVoterRepository) GroupBy(ctx context.Context, req models.GroupByRequest) (*models.GroupByResult, error) {
	fieldExpr := "voter_name_english"
	switch strings.ToLower(req.Field) {
	case "name", "full_name", "voter_name":
		fieldExpr = "COALESCE(NULLIF(TRIM(voter_name_english), ''), 'Unknown')"
	case "relative_name", "relative":
		fieldExpr = "COALESCE(NULLIF(TRIM(relative_name_english), ''), 'Unknown')"
	case "gender":
		fieldExpr = "COALESCE(NULLIF(TRIM(gender_english), ''), 'Unknown')"
	case "assembly", "assembly_constituency":
		fieldExpr = "COALESCE(NULLIF(TRIM(assembly_constituency), ''), 'Unknown')"
	case "town_village", "town", "village":
		fieldExpr = "COALESCE(NULLIF(TRIM(town_village), ''), 'Unknown')"
	case "age_group":
		fieldExpr = "CASE WHEN age < 25 THEN '18-24' WHEN age < 40 THEN '25-39' WHEN age < 60 THEN '40-59' ELSE '60+' END"
	}

	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	minCount := req.MinCount
	if minCount < 1 {
		minCount = 1
	}
	sortOrder := "DESC"
	if strings.EqualFold(req.Sort, "asc") {
		sortOrder = "ASC"
	}

	var totalVoters int64
	err := r.duckDB.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM voters;").Scan(&totalVoters)
	if err != nil || totalVoters == 0 {
		totalVoters = 1
	}

	query := fmt.Sprintf(`
		SELECT %s AS val, COUNT(*) AS cnt
		FROM voters
		GROUP BY val
		HAVING cnt >= ?
		ORDER BY cnt %s
		LIMIT ?;
	`, fieldExpr, sortOrder)

	rows, err := r.duckDB.DB.QueryContext(ctx, query, minCount, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GroupBy query: %w", err)
	}
	defer rows.Close()

	items := make([]models.GroupCountItem, 0)
	for rows.Next() {
		var item models.GroupCountItem
		if err := rows.Scan(&item.Value, &item.Count); err != nil {
			return nil, fmt.Errorf("failed scanning group by row: %w", err)
		}
		item.Percentage = math.Round((float64(item.Count)/float64(totalVoters))*10000.0) / 100.0
		items = append(items, item)
	}

	return &models.GroupByResult{
		Field:      req.Field,
		TotalItems: totalVoters,
		Groups:     items,
	}, nil
}

func (r *duckDBVoterRepository) GetNearbyPollingStations(ctx context.Context, req models.GeoNearbyRequest) ([]models.GeoPollingStationResult, error) {
	radius := req.RadiusKM
	if radius <= 0 {
		radius = 5.0 // default 5km
	}
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	query := `
		SELECT * FROM (
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
				COALESCE(total_female_voters, 0),
				(6371.0 * 2 * ASIN(SQRT(
					POWER(SIN(RADIANS(? - latitude) / 2), 2) +
					COS(RADIANS(?)) * COS(RADIANS(latitude)) *
					POWER(SIN(RADIANS(? - longitude) / 2), 2)
				))) AS distance_km
			FROM polling_stations
			WHERE latitude != 0 AND longitude != 0
		) sub
		WHERE sub.distance_km <= ?
		ORDER BY sub.distance_km ASC
		LIMIT ?;
	`

	rows, err := r.duckDB.DB.QueryContext(ctx, query, req.Latitude, req.Latitude, req.Longitude, radius, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query nearby polling stations: %w", err)
	}
	defer rows.Close()

	results := make([]models.GeoPollingStationResult, 0)
	for rows.Next() {
		var res models.GeoPollingStationResult
		err := rows.Scan(
			&res.PollingStation.AssemblyConstituency,
			&res.PollingStation.PartNumber,
			&res.PollingStation.PollingStationName,
			&res.PollingStation.PollingStationAddress,
			&res.PollingStation.TownVillage,
			&res.PollingStation.Tehsil,
			&res.PollingStation.District,
			&res.PollingStation.PinCode,
			&res.PollingStation.Latitude,
			&res.PollingStation.Longitude,
			&res.PollingStation.TotalVoters,
			&res.PollingStation.TotalMaleVoters,
			&res.PollingStation.TotalFemaleVoters,
			&res.DistanceKM,
		)
		if err != nil {
			return nil, fmt.Errorf("failed scanning nearby polling station: %w", err)
		}
		res.DistanceKM = math.Round(res.DistanceKM*100) / 100
		res.DistanceMeters = math.Round(res.DistanceKM * 1000)
		results = append(results, res)
	}

	return results, nil
}

func (r *duckDBVoterRepository) GetNearbyVoters(ctx context.Context, req models.GeoNearbyRequest) ([]models.GeoVoterResult, error) {
	radius := req.RadiusKM
	if radius <= 0 {
		radius = 2.0 // default 2km
	}
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	query := `
		SELECT * FROM (
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
				COALESCE(section_number_and_name, ''),
				COALESCE(post_office, ''),
				COALESCE(police_station, ''),
				COALESCE(latitude, 0.0),
				COALESCE(longitude, 0.0),
				(6371.0 * 2 * ASIN(SQRT(
					POWER(SIN(RADIANS(? - latitude) / 2), 2) +
					COS(RADIANS(?)) * COS(RADIANS(latitude)) *
					POWER(SIN(RADIANS(? - longitude) / 2), 2)
				))) AS distance_km
			FROM voters
			WHERE latitude != 0 AND longitude != 0
		) sub
		WHERE sub.distance_km <= ?
		ORDER BY sub.distance_km ASC
		LIMIT ?;
	`

	rows, err := r.duckDB.DB.QueryContext(ctx, query, req.Latitude, req.Latitude, req.Longitude, radius, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query nearby voters: %w", err)
	}
	defer rows.Close()

	results := make([]models.GeoVoterResult, 0)
	for rows.Next() {
		var res models.GeoVoterResult
		err := rows.Scan(
			&res.Voter.EPICNo,
			&res.Voter.FullName,
			&res.Voter.FullNameHindi,
			&res.Voter.RelativeName,
			&res.Voter.RelativeNameHindi,
			&res.Voter.Gender,
			&res.Voter.GenderHindi,
			&res.Voter.Age,
			&res.Voter.HouseNo,
			&res.Voter.PollingStationName,
			&res.Voter.PollingStationAddress,
			&res.Voter.PartNumber,
			&res.Voter.AssemblyConstituency,
			&res.Voter.AssemblyConstituencyFull,
			&res.Voter.ParliamentaryConstituency,
			&res.Voter.TownVillage,
			&res.Voter.Tehsil,
			&res.Voter.District,
			&res.Voter.PinCode,
			&res.Voter.SectionNumberAndName,
			&res.Voter.PostOffice,
			&res.Voter.PoliceStation,
			&res.Voter.Latitude,
			&res.Voter.Longitude,
			&res.DistanceKM,
		)
		if err != nil {
			return nil, fmt.Errorf("failed scanning nearby voter: %w", err)
		}
		res.DistanceKM = math.Round(res.DistanceKM*100) / 100
		res.DistanceMeters = math.Round(res.DistanceKM * 1000)
		results = append(results, res)
	}

	return results, nil
}

func (r *duckDBVoterRepository) CalculateDistance(lat1, lng1, lat2, lng2 float64) models.GeoDistanceResult {
	const earthRadiusKM = 6371.0

	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLng := (lng2 - lng1) * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180.0))*math.Cos(lat2*(math.Pi/180.0))*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distKM := earthRadiusKM * c
	distMiles := distKM * 0.621371
	distMeters := distKM * 1000.0

	return models.GeoDistanceResult{
		Lat1:           lat1,
		Lng1:           lng1,
		Lat2:           lat2,
		Lng2:           lng2,
		DistanceKM:     math.Round(distKM*100) / 100,
		DistanceMiles:  math.Round(distMiles*100) / 100,
		DistanceMeters: math.Round(distMeters),
	}
}
