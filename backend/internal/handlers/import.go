package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"travel-planner-viewer/backend/internal/models"
	"travel-planner-viewer/backend/internal/store"
	"travel-planner-viewer/backend/internal/utils"
)

type ImportHandler struct {
	store     *store.PlansStore
	adminMode bool
}

func NewImportHandler(s *store.PlansStore, adminMode bool) *ImportHandler {
	return &ImportHandler{store: s, adminMode: adminMode}
}

func (h *ImportHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/plans/import/template", h.DownloadTemplate)
	mux.HandleFunc("/api/plans/import", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			AdminGuard(h.ImportPlan, h.adminMode)(w, r)
			return
		}
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	})
}

func (h *ImportHandler) DownloadTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	data, err := utils.BuildImportTemplate()
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="travel_plan_template.xlsx"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *ImportHandler) ImportPlan(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "missing file")
		return
	}
	defer file.Close()

	rows, err := utils.ParseImportExcel(file)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	planName := strings.TrimSpace(r.FormValue("planName"))
	if planName == "" {
		planName = "Imported Plan " + time.Now().Format("20060102_150405")
	}

	type groupedRow struct {
		date string
		row  utils.ExcelImportRow
	}
	groups := map[string][]utils.ExcelImportRow{}
	warnings := make([]string, 0)

	for _, row := range rows {
		if h.isDuplicateStop("", row.Date, row.Location) {
			warnings = append(warnings, fmt.Sprintf("Skipped duplicate stop: %s %s", row.Date, row.Location))
			continue
		}
		groups[row.Date] = append(groups[row.Date], row)
	}

	dates := make([]string, 0, len(groups))
	for d := range groups {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	if len(dates) == 0 {
		utils.WriteError(w, http.StatusBadRequest, "no valid rows after dedup")
		return
	}

	dayPlans := make([]models.DayPlan, 0, len(dates))
	for idx, d := range dates {
		stops := make([]models.StopPoint, 0, len(groups[d]))
		for _, row := range groups[d] {
			lat := row.Latitude
			lng := row.Longitude
			if !(row.HasLatitude && row.HasLongitude) {
				parts := make([]string, 0, 3)
				if v := strings.TrimSpace(row.Location); v != "" {
					parts = append(parts, v)
				}
				if v := strings.TrimSpace(row.CityRegion); v != "" {
					parts = append(parts, v)
				}
				if v := strings.TrimSpace(row.Country); v != "" {
					parts = append(parts, v)
				}
				geocodeQuery := strings.Join(parts, ", ")

				resolvedLat, resolvedLng, geoErr := utils.GeocodeLocation(geocodeQuery)
				if geoErr != nil {
					warnings = append(warnings, fmt.Sprintf("Could not geocode location: %s (query: %s, %v)", row.Location, geocodeQuery, geoErr))
				} else {
					lat = resolvedLat
					lng = resolvedLng
				}
			}

			stops = append(stops, models.StopPoint{
				ID:                  "",
				Name:                row.Location,
				Latitude:            lat,
				Longitude:           lng,
				ActivityDescription: row.Description,
				ImageURLs:           []string{},
			})
		}
		dayPlans = append(dayPlans, models.DayPlan{
			DayNumber: idx + 1,
			Title:     d,
			Stops:     stops,
		})
	}

	plan := models.PlanDetail{
		Name:       planName,
		StartDate:  dates[0],
		EndDate:    dates[len(dates)-1],
		CoverImage: "https://via.placeholder.com/640x360?text=Excel+Import",
		IsSimple:   true,
		DayPlans:   dayPlans,
	}

	created, err := h.store.CreatePlan(plan)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, models.ImportPlanResponse{
		PlanID:   created.ID,
		Name:     created.Name,
		Warnings: warnings,
	})
}

// isDuplicateStop scans all existing plans and checks duplicate by (date + locationName).
func (h *ImportHandler) isDuplicateStop(_ string, date, locationName string) bool {
	normDate := strings.TrimSpace(date)
	normName := strings.ToLower(strings.TrimSpace(locationName))
	if normDate == "" || normName == "" {
		return false
	}

	for _, summary := range h.store.ListPlans() {
		plan, err := h.store.GetPlan(summary.ID)
		if err != nil {
			continue
		}
		start, err := time.Parse("2006-01-02", plan.StartDate)
		if err != nil {
			continue
		}
		for _, day := range plan.DayPlans {
			currentDate := start.AddDate(0, 0, day.DayNumber-1).Format("2006-01-02")
			if currentDate != normDate {
				continue
			}
			for _, stop := range day.Stops {
				if strings.ToLower(strings.TrimSpace(stop.Name)) == normName {
					return true
				}
			}
		}
	}
	return false
}
