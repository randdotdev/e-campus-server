package settings

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

type SettingsRepository interface {
	Get(ctx context.Context) (*SettingsRow, error)
	Update(ctx context.Context, settings json.RawMessage, updatedBy uuid.UUID) error
}

type Service struct {
	settingsRepo SettingsRepository

	mu       sync.RWMutex
	cached   *UniversitySettings
	cachedAt time.Time
}

func NewService(settingsRepo SettingsRepository) *Service {
	return &Service{settingsRepo: settingsRepo}
}

func (s *Service) Get(ctx context.Context) (*UniversitySettings, error) {
	s.mu.RLock()
	if s.cached != nil {
		defer s.mu.RUnlock()
		return s.cached, nil
	}
	s.mu.RUnlock()

	return s.refresh(ctx)
}

func (s *Service) Update(ctx context.Context, settings *UniversitySettings, updatedBy uuid.UUID) error {
	if err := ValidateSettings(settings); err != nil {
		return err
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	if err := s.settingsRepo.Update(ctx, data, updatedBy); err != nil {
		return err
	}

	s.mu.Lock()
	s.cached = settings
	s.cachedAt = time.Now()
	s.mu.Unlock()

	return nil
}

func (s *Service) UpdatePartial(ctx context.Context, updates SettingsUpdates, updatedBy uuid.UUID) (*UniversitySettings, error) {
	current, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}

	merged := ApplyUpdates(current, updates)

	if err := s.Update(ctx, merged, updatedBy); err != nil {
		return nil, err
	}

	return merged, nil
}

func (s *Service) refresh(ctx context.Context) (*UniversitySettings, error) {
	row, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrSettingsNotFound
	}

	var settings UniversitySettings
	if err := json.Unmarshal(row.Settings, &settings); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cached = &settings
	s.cachedAt = time.Now()
	s.mu.Unlock()

	return &settings, nil
}

func (s *Service) Refresh(ctx context.Context) error {
	_, err := s.refresh(ctx)
	return err
}

func (s *Service) GetFeatures(ctx context.Context) (Features, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return Features{}, err
	}
	return settings.Features, nil
}

func (s *Service) IsFeatureEnabled(ctx context.Context, feature string) (bool, error) {
	f, err := s.GetFeatures(ctx)
	if err != nil {
		return false, err
	}
	return GetFeatureByName(f, feature), nil
}

func (s *Service) GetInstitution(ctx context.Context) (Institution, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return Institution{}, err
	}
	return settings.Institution, nil
}

func (s *Service) GetGradingConfig(ctx context.Context) (GradingConfig, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return GradingConfig{}, err
	}
	return settings.Grading, nil
}

func (s *Service) GetAcademicConfig(ctx context.Context) (AcademicConfig, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return AcademicConfig{}, err
	}
	return settings.Academic, nil
}

type SettingsUpdates struct {
	Institution  *Institution
	DegreeLabels map[string]DegreeLabel
	Grading      *GradingConfig
	Features     *Features
	Academic     *AcademicConfig
}
