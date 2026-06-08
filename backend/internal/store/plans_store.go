package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"travel-planner-viewer/backend/internal/models"
)

var (
	ErrPlanNotFound = errors.New("plan not found")
	ErrStopNotFound = errors.New("stop not found")
)

type AddStopRequest struct {
	DayNumber int              `json:"dayNumber"`
	Stop      models.StopPoint `json:"stop"`
}

type PlansStore struct {
	mu       sync.RWMutex
	dataDir  string
	index    models.PlansIndex
	indexLoc string
	plansDir string
}

func NewPlansStore(dataDir string) (*PlansStore, error) {
	s := &PlansStore{
		dataDir:  dataDir,
		indexLoc: filepath.Join(dataDir, "index.json"),
		plansDir: filepath.Join(dataDir, "plans"),
	}
	if err := s.loadOrInit(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *PlansStore) ListPlans() []models.PlanSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.PlanSummary, len(s.index.Plans))
	copy(out, s.index.Plans)
	return out
}

func (s *PlansStore) CreatePlan(plan models.PlanDetail) (models.PlanDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if plan.ID == "" {
		plan.ID = generatePlanID(plan.Name)
	}
	plan.ID = sanitizeID(plan.ID)
	if plan.CoverImage == "" {
		plan.CoverImage = "https://via.placeholder.com/640x360?text=Travel+Plan"
	}
	for _, p := range s.index.Plans {
		if p.ID == plan.ID {
			return models.PlanDetail{}, fmt.Errorf("plan id already exists: %s", plan.ID)
		}
	}
	if err := s.savePlanLocked(plan); err != nil {
		return models.PlanDetail{}, err
	}
	s.index.Plans = append(s.index.Plans, toSummary(plan))
	if err := s.saveIndexLocked(); err != nil {
		return models.PlanDetail{}, err
	}
	return plan, nil
}

func (s *PlansStore) GetPlan(id string) (models.PlanDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadPlanLocked(id)
}

func (s *PlansStore) UpdatePlan(id string, plan models.PlanDetail) (models.PlanDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.loadPlanLocked(id)
	if err != nil {
		return models.PlanDetail{}, err
	}
	plan.ID = id
	if plan.CoverImage == "" {
		plan.CoverImage = "https://via.placeholder.com/640x360?text=Travel+Plan"
	}
	if err := s.savePlanLocked(plan); err != nil {
		return models.PlanDetail{}, err
	}
	s.upsertSummaryLocked(toSummary(plan))
	if err := s.saveIndexLocked(); err != nil {
		return models.PlanDetail{}, err
	}
	return plan, nil
}

func (s *PlansStore) DeletePlan(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.loadPlanLocked(id); err != nil {
		return err
	}
	if err := os.Remove(s.planFile(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	filtered := make([]models.PlanSummary, 0, len(s.index.Plans))
	for _, p := range s.index.Plans {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}
	s.index.Plans = filtered
	return s.saveIndexLocked()
}

func (s *PlansStore) AddStop(planID string, req AddStopRequest) (models.StopPoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	plan, err := s.loadPlanLocked(planID)
	if err != nil {
		return models.StopPoint{}, err
	}
	stop := req.Stop
	if stop.ID == "" {
		stop.ID = fmt.Sprintf("day%d-%d", req.DayNumber, time.Now().UnixNano())
	}
	for i := range plan.DayPlans {
		if plan.DayPlans[i].DayNumber == req.DayNumber {
			plan.DayPlans[i].Stops = append(plan.DayPlans[i].Stops, stop)
			if err := s.savePlanLocked(plan); err != nil {
				return models.StopPoint{}, err
			}
			return stop, nil
		}
	}
	plan.DayPlans = append(plan.DayPlans, models.DayPlan{
		DayNumber: req.DayNumber,
		Title:     fmt.Sprintf("Day %d", req.DayNumber),
		Stops:     []models.StopPoint{stop},
	})
	if err := s.savePlanLocked(plan); err != nil {
		return models.StopPoint{}, err
	}
	return stop, nil
}

func (s *PlansStore) DeleteStop(planID, stopID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	plan, err := s.loadPlanLocked(planID)
	if err != nil {
		return err
	}
	found := false
	for i := range plan.DayPlans {
		stops := plan.DayPlans[i].Stops
		for j := range stops {
			if stops[j].ID == stopID {
				plan.DayPlans[i].Stops = append(stops[:j], stops[j+1:]...)
				found = true
				break
			}
		}
	}
	if !found {
		return ErrStopNotFound
	}
	return s.savePlanLocked(plan)
}

func (s *PlansStore) loadOrInit() error {
	if err := os.MkdirAll(s.plansDir, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(s.indexLoc); errors.Is(err, os.ErrNotExist) {
		return s.bootstrapDefaults()
	}
	return s.loadIndexLocked()
}

func (s *PlansStore) bootstrapDefaults() error {
	s.index.Plans = []models.PlanSummary{}
	return s.saveIndexLocked()
}

func (s *PlansStore) loadIndexLocked() error {
	f, err := os.Open(s.indexLoc)
	if err != nil {
		return err
	}
	defer f.Close()
	var idx models.PlansIndex
	if err := json.NewDecoder(f).Decode(&idx); err != nil {
		return err
	}
	s.index = idx
	return nil
}

func (s *PlansStore) saveIndexLocked() error {
	f, err := os.Create(s.indexLoc)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(s.index)
}

func (s *PlansStore) loadPlanLocked(id string) (models.PlanDetail, error) {
	f, err := os.Open(s.planFile(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return models.PlanDetail{}, ErrPlanNotFound
		}
		return models.PlanDetail{}, err
	}
	defer f.Close()
	var p models.PlanDetail
	if err := json.NewDecoder(f).Decode(&p); err != nil {
		return models.PlanDetail{}, err
	}
	return p, nil
}

func (s *PlansStore) savePlanLocked(plan models.PlanDetail) error {
	f, err := os.Create(s.planFile(plan.ID))
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(plan)
}

func (s *PlansStore) upsertSummaryLocked(summary models.PlanSummary) {
	for i := range s.index.Plans {
		if s.index.Plans[i].ID == summary.ID {
			s.index.Plans[i] = summary
			return
		}
	}
	s.index.Plans = append(s.index.Plans, summary)
}

func (s *PlansStore) planFile(id string) string {
	return filepath.Join(s.plansDir, sanitizeID(id)+".json")
}

func sanitizeID(id string) string {
	id = strings.TrimSpace(strings.ToLower(id))
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.ReplaceAll(id, "/", "-")
	return id
}

func generatePlanID(name string) string {
	base := sanitizeID(name)
	if base == "" {
		base = "plan"
	}
	return fmt.Sprintf("%s-%d", base, time.Now().Unix())
}

func toSummary(p models.PlanDetail) models.PlanSummary {
	return models.PlanSummary{
		ID:         p.ID,
		Name:       p.Name,
		StartDate:  p.StartDate,
		EndDate:    p.EndDate,
		CoverImage: p.CoverImage,
		IsSimple:   p.IsSimple,
	}
}
