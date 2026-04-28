package handlers

import (
	"net/http"

	"travel-planner-viewer/backend/internal/utils"
)

func AdminGuard(next http.HandlerFunc, adminMode bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if adminMode || utils.IsAdminOverride(r) {
			next(w, r)
			return
		}
		utils.WriteError(w, http.StatusForbidden, "write operation is forbidden when admin mode is disabled")
	}
}
