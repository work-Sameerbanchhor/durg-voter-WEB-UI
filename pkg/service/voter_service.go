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
	ListConstituencies(ctx context.Context) ([]models.ConstituencySummary, error)
}

type voterService struct {
	repo repository.VoterRepository

	mu                 sync.RWMutex
	cachedStats        *models.StatsSummary
	statsCacheTime     time.Time
	cachedConstituencies []models.ConstituencySummary
	constCacheTime     time.Time
	cacheTTL           time.Duration
}

func NewVoterService(repo repository.VoterRepository) VoterService {
	return &voterService{
		repo:     repo,
		cacheTTL: 10 * time.Minute,
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

	return s.repo.ListVoters(ctx, filter)
}

func (s *voterService) GetVoterByEPIC(ctx context.Context, epicNo string) (*models.Voter, error) {
	if epicNo == "" {
		return nil, fmt.Errorf("epic number cannot be empty")
	}
	return s.repo.GetVoterByEPIC(ctx, epicNo)
}

func (s *voterService) ListPollingStations(ctx context.Context, filter models.SearchFilter) ([]models.PollingStation, *models.Pagination, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
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
