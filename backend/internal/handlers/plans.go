package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"travel-planner-viewer/backend/internal/models"
	"travel-planner-viewer/backend/internal/store"
	"travel-planner-viewer/backend/internal/utils"
)

type PlansHandler struct {
	store     *store.PlansStore
	adminMode bool
}

func NewPlansHandler(s *store.PlansStore, adminMode bool) *PlansHandler {
	return &PlansHandler{store: s, adminMode: adminMode}
}

func (h *PlansHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/admin/status", h.handleAdminStatus)
	mux.HandleFunc("/api/plans", h.handlePlans)
	mux.HandleFunc("/api/plans/", h.handlePlanRoutes)
}

func (h *PlansHandler) handleAdminStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	utils.WriteJSON(w, http.StatusOK, map[string]bool{
		"isAdmin": h.adminMode || utils.IsAdminOverride(r),
	})
}

func (h *PlansHandler) handlePlans(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		utils.WriteJSON(w, http.StatusOK, map[string][]models.PlanSummary{"plans": h.store.ListPlans()})
	case http.MethodPost:
		AdminGuard(h.createPlan, h.adminMode)(w, r)
	default:
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *PlansHandler) createPlan(w http.ResponseWriter, r *http.Request) {
	var payload models.PlanDetail
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		utils.WriteError(w, http.StatusBadRequest, "plan name is required")
		return
	}
	created, err := h.store.CreatePlan(payload)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusCreated, created)
}

func (h *PlansHandler) handlePlanRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/plans/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		utils.WriteError(w, http.StatusBadRequest, "missing plan id")
		return
	}
	planID := parts[0]

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			h.getPlan(w, r, planID)
		case http.MethodPut:
			AdminGuard(func(w http.ResponseWriter, r *http.Request) {
				h.updatePlan(w, r, planID)
			}, h.adminMode)(w, r)
		case http.MethodDelete:
			AdminGuard(func(w http.ResponseWriter, r *http.Request) {
				h.deletePlan(w, r, planID)
			}, h.adminMode)(w, r)
		default:
			utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	if len(parts) == 2 && parts[1] == "stops" && r.Method == http.MethodPost {
		AdminGuard(func(w http.ResponseWriter, r *http.Request) {
			h.addStop(w, r, planID)
		}, h.adminMode)(w, r)
		return
	}

	if len(parts) == 3 && parts[1] == "stops" && r.Method == http.MethodDelete {
		stopID := parts[2]
		AdminGuard(func(w http.ResponseWriter, r *http.Request) {
			h.deleteStop(w, r, planID, stopID)
		}, h.adminMode)(w, r)
		return
	}

	utils.WriteError(w, http.StatusNotFound, "route not found")
}

func (h *PlansHandler) getPlan(w http.ResponseWriter, r *http.Request, planID string) {
	plan, err := h.store.GetPlan(planID)
	if err != nil {
		if errors.Is(err, store.ErrPlanNotFound) {
			utils.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, plan)
}

func (h *PlansHandler) updatePlan(w http.ResponseWriter, r *http.Request, planID string) {
	var payload models.PlanDetail
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		utils.WriteError(w, http.StatusBadRequest, "plan name is required")
		return
	}
	updated, err := h.store.UpdatePlan(planID, payload)
	if err != nil {
		if errors.Is(err, store.ErrPlanNotFound) {
			utils.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, updated)
}

func (h *PlansHandler) deletePlan(w http.ResponseWriter, r *http.Request, planID string) {
	if err := h.store.DeletePlan(planID); err != nil {
		if errors.Is(err, store.ErrPlanNotFound) {
			utils.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PlansHandler) addStop(w http.ResponseWriter, r *http.Request, planID string) {
	var req store.AddStopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.DayNumber < 1 || strings.TrimSpace(req.Stop.Name) == "" {
		utils.WriteError(w, http.StatusBadRequest, "dayNumber and stop.name are required")
		return
	}
	stop, err := h.store.AddStop(planID, req)
	if err != nil {
		if errors.Is(err, store.ErrPlanNotFound) {
			utils.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusCreated, stop)
}

func (h *PlansHandler) deleteStop(w http.ResponseWriter, r *http.Request, planID, stopID string) {
	if stopID == "" {
		utils.WriteError(w, http.StatusBadRequest, "missing stopId")
		return
	}
	if err := h.store.DeleteStop(planID, stopID); err != nil {
		if errors.Is(err, store.ErrPlanNotFound) || errors.Is(err, store.ErrStopNotFound) {
			utils.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
