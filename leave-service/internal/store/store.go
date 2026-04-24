package store

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/asdlc-repos/prev-demo092/leave-service/internal/models"
)

type Store struct {
	mu       sync.RWMutex
	users    map[string]*models.User
	balances map[string]*models.LeaveBalance
	requests map[string]*models.LeaveRequest
	seq      uint64
}

func New() *Store {
	s := &Store{
		users:    make(map[string]*models.User),
		balances: make(map[string]*models.LeaveBalance),
		requests: make(map[string]*models.LeaveRequest),
	}
	s.seed()
	return s
}

func (s *Store) seed() {
	manager := &models.User{
		ID:    "manager1",
		Email: "manager1@example.com",
		Name:  "Morgan Manager",
		Role:  models.RoleManager,
	}
	employees := []*models.User{
		{ID: "employee1", Email: "employee1@example.com", Name: "Alice Employee", Role: models.RoleEmployee, ManagerID: "manager1"},
		{ID: "employee2", Email: "employee2@example.com", Name: "Bob Employee", Role: models.RoleEmployee, ManagerID: "manager1"},
		{ID: "employee3", Email: "employee3@example.com", Name: "Carol Employee", Role: models.RoleEmployee, ManagerID: "manager1"},
	}

	s.users[manager.ID] = manager
	s.balances[manager.ID] = &models.LeaveBalance{Annual: 20, Sick: 10, Personal: 5}
	for _, e := range employees {
		s.users[e.ID] = e
		s.balances[e.ID] = &models.LeaveBalance{Annual: 20, Sick: 10, Personal: 5}
	}

	sample := &models.LeaveRequest{
		ID:           s.nextID(),
		EmployeeID:   "employee1",
		EmployeeName: "Alice Employee",
		StartDate:    "2026-05-01",
		EndDate:      "2026-05-03",
		LeaveType:    models.LeaveAnnual,
		Reason:       "Family trip",
		Status:       models.StatusPending,
		CreatedAt:    time.Now().UTC(),
	}
	s.requests[sample.ID] = sample
}

func (s *Store) nextID() string {
	n := atomic.AddUint64(&s.seq, 1)
	return fmt.Sprintf("lr-%d", n)
}

func (s *Store) GetUser(id string) (*models.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return nil, false
	}
	cp := *u
	return &cp, true
}

func (s *Store) GetBalance(userID string) (*models.LeaveBalance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.balances[userID]
	if !ok {
		return nil, false
	}
	cp := *b
	return &cp, true
}

func (s *Store) DirectReportIDs(managerID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var ids []string
	for _, u := range s.users {
		if u.ManagerID == managerID {
			ids = append(ids, u.ID)
		}
	}
	return ids
}

func (s *Store) ListRequestsForEmployees(employeeIDs []string, status string) []*models.LeaveRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	set := make(map[string]struct{}, len(employeeIDs))
	for _, id := range employeeIDs {
		set[id] = struct{}{}
	}
	var out []*models.LeaveRequest
	for _, r := range s.requests {
		if _, ok := set[r.EmployeeID]; !ok {
			continue
		}
		if status != "" && string(r.Status) != status {
			continue
		}
		cp := *r
		out = append(out, &cp)
	}
	return out
}

func (s *Store) GetRequest(id string) (*models.LeaveRequest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.requests[id]
	if !ok {
		return nil, false
	}
	cp := *r
	return &cp, true
}

func (s *Store) CreateRequest(r *models.LeaveRequest) *models.LeaveRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	r.ID = s.nextID()
	r.CreatedAt = time.Now().UTC()
	r.Status = models.StatusPending
	s.requests[r.ID] = r
	cp := *r
	return &cp
}

func (s *Store) UpdateRequestStatus(id string, status models.LeaveStatus, reviewerID, rejectionReason string) (*models.LeaveRequest, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.requests[id]
	if !ok {
		return nil, false
	}
	now := time.Now().UTC()
	r.Status = status
	r.ReviewedAt = &now
	r.ReviewedBy = reviewerID
	if status == models.StatusRejected {
		r.RejectionReason = rejectionReason
	}
	cp := *r
	return &cp, true
}
