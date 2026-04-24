package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/asdlc-repos/prev-demo092/leave-service/internal/models"
	"github.com/asdlc-repos/prev-demo092/leave-service/internal/store"
)

const defaultUserID = "employee1"

type Handler struct {
	store *store.Store
}

func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/api/v1/users/me", h.handleMe)
	mux.HandleFunc("/api/v1/leave-balance", h.handleLeaveBalance)
	mux.HandleFunc("/api/v1/leave-requests", h.handleLeaveRequests)
	mux.HandleFunc("/api/v1/leave-requests/", h.handleLeaveRequestByID)
	return mux
}

func (h *Handler) currentUser(r *http.Request) (*models.User, bool) {
	id := r.Header.Get("X-User-Id")
	if id == "" {
		id = defaultUserID
	}
	return h.store.GetUser(id)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	user, ok := h.currentUser(r)
	if !ok {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) handleLeaveBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	user, ok := h.currentUser(r)
	if !ok {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	balance, ok := h.store.GetBalance(user.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "balance not found")
		return
	}
	writeJSON(w, http.StatusOK, balance)
}

func (h *Handler) handleLeaveRequests(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(r)
	if !ok {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.listLeaveRequests(w, r, user)
	case http.MethodPost:
		h.createLeaveRequest(w, r, user)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) listLeaveRequests(w http.ResponseWriter, r *http.Request, user *models.User) {
	status := r.URL.Query().Get("status")
	if status != "" && !models.IsValidStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid status filter")
		return
	}

	var ids []string
	if user.Role == models.RoleManager {
		ids = h.store.DirectReportIDs(user.ID)
	} else {
		ids = []string{user.ID}
	}

	results := h.store.ListRequestsForEmployees(ids, status)
	if results == nil {
		results = []*models.LeaveRequest{}
	}
	writeJSON(w, http.StatusOK, results)
}

type createRequestPayload struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	LeaveType string `json:"leaveType"`
	Reason    string `json:"reason"`
}

func (h *Handler) createLeaveRequest(w http.ResponseWriter, r *http.Request, user *models.User) {
	var payload createRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if payload.StartDate == "" || payload.EndDate == "" || payload.LeaveType == "" {
		writeError(w, http.StatusBadRequest, "startDate, endDate, and leaveType are required")
		return
	}
	if !models.IsValidLeaveType(payload.LeaveType) {
		writeError(w, http.StatusBadRequest, "invalid leaveType")
		return
	}

	start, err := time.Parse("2006-01-02", payload.StartDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid startDate format (expected YYYY-MM-DD)")
		return
	}
	end, err := time.Parse("2006-01-02", payload.EndDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid endDate format (expected YYYY-MM-DD)")
		return
	}
	if end.Before(start) {
		writeError(w, http.StatusBadRequest, "endDate must be on or after startDate")
		return
	}

	days := float64(end.Sub(start).Hours()/24) + 1
	balance, ok := h.store.GetBalance(user.ID)
	if !ok {
		writeError(w, http.StatusBadRequest, "no leave balance for user")
		return
	}
	switch models.LeaveType(payload.LeaveType) {
	case models.LeaveAnnual:
		if balance.Annual < days {
			writeError(w, http.StatusBadRequest, "insufficient annual leave balance")
			return
		}
	case models.LeaveSick:
		if balance.Sick < days {
			writeError(w, http.StatusBadRequest, "insufficient sick leave balance")
			return
		}
	case models.LeavePersonal:
		if balance.Personal < days {
			writeError(w, http.StatusBadRequest, "insufficient personal leave balance")
			return
		}
	}

	req := &models.LeaveRequest{
		EmployeeID:   user.ID,
		EmployeeName: user.Name,
		StartDate:    payload.StartDate,
		EndDate:      payload.EndDate,
		LeaveType:    models.LeaveType(payload.LeaveType),
		Reason:       payload.Reason,
	}
	created := h.store.CreateRequest(req)
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) handleLeaveRequestByID(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(r)
	if !ok {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/leave-requests/")
	if path == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	parts := strings.Split(path, "/")

	id := parts[0]
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.getLeaveRequest(w, user, id)
		return
	}

	if len(parts) != 2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	action := parts[1]
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	switch action {
	case "approve":
		h.decideLeaveRequest(w, r, user, id, models.StatusApproved)
	case "reject":
		h.decideLeaveRequest(w, r, user, id, models.StatusRejected)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *Handler) getLeaveRequest(w http.ResponseWriter, user *models.User, id string) {
	req, ok := h.store.GetRequest(id)
	if !ok {
		writeError(w, http.StatusNotFound, "leave request not found")
		return
	}
	if !h.canView(user, req) {
		writeError(w, http.StatusForbidden, "not authorized to view this request")
		return
	}
	writeJSON(w, http.StatusOK, req)
}

type rejectPayload struct {
	Reason string `json:"reason"`
}

func (h *Handler) decideLeaveRequest(w http.ResponseWriter, r *http.Request, user *models.User, id string, status models.LeaveStatus) {
	if user.Role != models.RoleManager {
		writeError(w, http.StatusForbidden, "only managers can approve or reject requests")
		return
	}
	req, ok := h.store.GetRequest(id)
	if !ok {
		writeError(w, http.StatusNotFound, "leave request not found")
		return
	}
	employee, ok := h.store.GetUser(req.EmployeeID)
	if !ok || employee.ManagerID != user.ID {
		writeError(w, http.StatusForbidden, "request does not belong to your direct report")
		return
	}
	if req.Status != models.StatusPending {
		writeError(w, http.StatusBadRequest, "request already reviewed")
		return
	}

	reason := ""
	if status == models.StatusRejected && r.Body != nil && r.ContentLength != 0 {
		var payload rejectPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
			reason = payload.Reason
		}
	}

	updated, ok := h.store.UpdateRequestStatus(id, status, user.ID, reason)
	if !ok {
		writeError(w, http.StatusNotFound, "leave request not found")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) canView(user *models.User, req *models.LeaveRequest) bool {
	if req.EmployeeID == user.ID {
		return true
	}
	if user.Role != models.RoleManager {
		return false
	}
	employee, ok := h.store.GetUser(req.EmployeeID)
	if !ok {
		return false
	}
	return employee.ManagerID == user.ID
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s %s", r.RemoteAddr, r.Method, r.URL.Path, time.Since(start))
	})
}
