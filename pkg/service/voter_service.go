package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"durg-voter-api/pkg/models"
	"durg-voter-api/pkg/repository"
)

type VoterService interface {
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
	AuthenticateUser(username, password string) (*models.LoginResponse, error)
}

type voterService struct {
	repo      repository.VoterRepository
	geminiSvc GeminiService

	mu                   sync.RWMutex
	cachedStats          *models.StatsSummary
	statsCacheTime       time.Time
	cachedConstituencies []models.ConstituencySummary
	constCacheTime       time.Time
	cacheTTL             time.Duration
}

func NewVoterService(repo repository.VoterRepository, geminiSvc GeminiService) VoterService {
	return &voterService{
		repo:      repo,
		geminiSvc: geminiSvc,
		cacheTTL:  10 * time.Minute,
	}
}

func (s *voterService) GetStats(ctx context.Context) (*models.StatsSummary, error) {
	s.mu.RLock()
	if s.cachedStats != nil && time.Since(s.statsCacheTime) < s.cacheTTL {
		defer s.mu.RUnlock()
		return s.cachedStats, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock
	if s.cachedStats != nil && time.Since(s.statsCacheTime) < s.cacheTTL {
		return s.cachedStats, nil
	}

	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("service failed to get stats: %w", err)
	}

	s.cachedStats = stats
	s.statsCacheTime = time.Now()
	return stats, nil
}

func (s *voterService) ListVoters(ctx context.Context, filter models.SearchFilter) ([]models.Voter, *models.Pagination, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	if s.geminiSvc != nil && filter.Query != "" && filter.HindiQuery == "" {
		if hindiQuery, err := s.geminiSvc.TransliterateEnglishToHindi(ctx, filter.Query); err == nil && hindiQuery != "" {
			filter.HindiQuery = hindiQuery
		}
	}

	voters, meta, err := s.repo.ListVoters(ctx, filter)
	if err == nil && meta != nil && filter.HindiQuery != "" {
		meta.TransliteratedQuery = filter.HindiQuery
	}
	return voters, meta, err
}

func (s *voterService) GetVoterByEPIC(ctx context.Context, epicNo string) (*models.Voter, error) {
	if epicNo == "" {
		return nil, fmt.Errorf("epic number cannot be empty")
	}
	return s.repo.GetVoterByEPIC(ctx, epicNo)
}

func (s *voterService) ListPollingStations(ctx context.Context, filter models.SearchFilter) ([]models.PollingStation, *models.Pagination, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 2000 {
		filter.Limit = 2000
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	return s.repo.ListPollingStations(ctx, filter)
}

func (s *voterService) GetPollingStation(ctx context.Context, assembly string, partNo int64) (*models.PollingStation, error) {
	if assembly == "" || partNo <= 0 {
		return nil, fmt.Errorf("assembly constituency and valid part number are required")
	}
	return s.repo.GetPollingStation(ctx, assembly, partNo)
}

func (s *voterService) ListConstituencies(ctx context.Context) ([]models.ConstituencySummary, error) {
	s.mu.RLock()
	if s.cachedConstituencies != nil && time.Since(s.constCacheTime) < s.cacheTTL {
		defer s.mu.RUnlock()
		return s.cachedConstituencies, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cachedConstituencies != nil && time.Since(s.constCacheTime) < s.cacheTTL {
		return s.cachedConstituencies, nil
	}

	constituencies, err := s.repo.ListConstituencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("service failed to list constituencies: %w", err)
	}

	s.cachedConstituencies = constituencies
	s.constCacheTime = time.Now()
	return constituencies, nil
}

func (s *voterService) ExecuteSQL(ctx context.Context, sqlQuery string) (*models.SQLResult, error) {
	if sqlQuery == "" {
		return nil, fmt.Errorf("SQL query string is required")
	}
	return s.repo.ExecuteSQL(ctx, sqlQuery)
}

func (s *voterService) GroupBy(ctx context.Context, req models.GroupByRequest) (*models.GroupByResult, error) {
	if req.Field == "" {
		req.Field = "full_name"
	}
	return s.repo.GroupBy(ctx, req)
}

func (s *voterService) GetNearbyPollingStations(ctx context.Context, req models.GeoNearbyRequest) ([]models.GeoPollingStationResult, error) {
	if req.Latitude == 0 || req.Longitude == 0 {
		return nil, fmt.Errorf("valid latitude and longitude coordinates are required")
	}
	return s.repo.GetNearbyPollingStations(ctx, req)
}

func (s *voterService) GetNearbyVoters(ctx context.Context, req models.GeoNearbyRequest) ([]models.GeoVoterResult, error) {
	if req.Latitude == 0 || req.Longitude == 0 {
		return nil, fmt.Errorf("valid latitude and longitude coordinates are required")
	}
	return s.repo.GetNearbyVoters(ctx, req)
}

func (s *voterService) CalculateDistance(lat1, lng1, lat2, lng2 float64) models.GeoDistanceResult {
	return s.repo.CalculateDistance(lat1, lng1, lat2, lng2)
}

func (s *voterService) AuthenticateUser(username, password string) (*models.LoginResponse, error) {
	switch username {
	case "admin":
		if password == "adminpass" || password == "admin" {
			return &models.LoginResponse{
				Token: "admin-token-secret-key-12345",
				Role:  models.RoleAdmin,
				User: models.User{
					Username:    "admin",
					Role:        models.RoleAdmin,
					Permissions: []string{"read", "search", "filter", "group_by", "geo", "execute_sql"},
				},
			}, nil
		}
	case "guest":
		if password == "guestpass" || password == "guest" || password == "" {
			return &models.LoginResponse{
				Token: "guest-token-secret-key-67890",
				Role:  models.RoleGuest,
				User: models.User{
					Username:    "guest",
					Role:        models.RoleGuest,
					Permissions: []string{"read", "search", "filter", "group_by", "geo"},
				},
			}, nil
		}
	}
	return nil, fmt.Errorf("invalid username or password credentials")
}

func (s *voterService) GetPartDetails(ctx context.Context, assembly string, partNo int64) (*models.PartDetails, error) {
	return s.repo.GetPartDetails(ctx, assembly, partNo)
}
