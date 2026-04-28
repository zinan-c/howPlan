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
	defaults := defaultPlans()
	s.index.Plans = make([]models.PlanSummary, 0, len(defaults))
	for _, p := range defaults {
		if err := s.savePlanLocked(p); err != nil {
			return err
		}
		s.index.Plans = append(s.index.Plans, toSummary(p))
	}
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

func defaultPlans() []models.PlanDetail {
	return []models.PlanDetail{
		{
			ID:         "yunnan-trip",
			Name:       "云南之旅",
			StartDate:  "2026-04-10",
			EndDate:    "2026-04-16",
			CoverImage: "https://via.placeholder.com/640x360?text=%E4%BA%91%E5%8D%97%E4%B9%8B%E6%97%85",
			DayPlans: []models.DayPlan{
				{DayNumber: 1, Title: "昆明集合", Stops: []models.StopPoint{{ID: "yn-d1-km", Name: "昆明翠湖", Latitude: 25.0486, Longitude: 102.7046, ActivityDescription: "抵达昆明，市区休整。", ImageURLs: []string{"https://via.placeholder.com/640x360?text=Kunming"}}}},
				{DayNumber: 2, Title: "大理古城", Stops: []models.StopPoint{{ID: "yn-d2-dl", Name: "大理古城", Latitude: 25.6927, Longitude: 100.1648, ActivityDescription: "古城漫步与洱海骑行。", ImageURLs: []string{"https://via.placeholder.com/640x360?text=Dali"}}}},
			},
		},
		{
			ID:         "japan-sakura",
			Name:       "日本赏樱",
			StartDate:  "2026-03-25",
			EndDate:    "2026-04-02",
			CoverImage: "https://via.placeholder.com/640x360?text=%E6%97%A5%E6%9C%AC%E8%B5%8F%E6%A8%B1",
			DayPlans: []models.DayPlan{
				{DayNumber: 1, Title: "东京初见", Stops: []models.StopPoint{{ID: "jp-d1-tyo", Name: "上野公园", Latitude: 35.7156, Longitude: 139.7745, ActivityDescription: "赏樱与博物馆参观。", ImageURLs: []string{"https://via.placeholder.com/640x360?text=Ueno+Sakura"}}}},
				{DayNumber: 2, Title: "京都夜樱", Stops: []models.StopPoint{{ID: "jp-d2-kyo", Name: "哲学之道", Latitude: 35.0269, Longitude: 135.7983, ActivityDescription: "步行赏樱，夜间拍摄。", ImageURLs: []string{"https://via.placeholder.com/640x360?text=Kyoto+Sakura"}}}},
			},
		},
		{
			ID:         "xinjiang-roadtrip",
			Name:       "新疆自驾",
			StartDate:  "2026-07-08",
			EndDate:    "2026-07-20",
			CoverImage: "https://via.placeholder.com/640x360?text=%E6%96%B0%E7%96%86%E8%87%AA%E9%A9%BE",
			DayPlans: []models.DayPlan{
				{DayNumber: 1, Title: "乌鲁木齐出发", Stops: []models.StopPoint{{ID: "xj-d1-wlmq", Name: "红山公园", Latitude: 43.8305, Longitude: 87.6168, ActivityDescription: "车辆整备，城市补给。", ImageURLs: []string{"https://via.placeholder.com/640x360?text=Urumqi"}}}},
				{DayNumber: 2, Title: "赛里木湖", Stops: []models.StopPoint{{ID: "xj-d2-slm", Name: "赛里木湖", Latitude: 44.6018, Longitude: 81.1663, ActivityDescription: "环湖公路自驾，日落拍摄。", ImageURLs: []string{"https://via.placeholder.com/640x360?text=Sailimu+Lake"}}}},
			},
		},
	}
}
