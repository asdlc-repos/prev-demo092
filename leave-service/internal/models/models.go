package models

import "time"

type Role string

const (
	RoleEmployee Role = "employee"
	RoleManager  Role = "manager"
)

type LeaveType string

const (
	LeaveAnnual   LeaveType = "annual"
	LeaveSick     LeaveType = "sick"
	LeavePersonal LeaveType = "personal"
)

type LeaveStatus string

const (
	StatusPending  LeaveStatus = "pending"
	StatusApproved LeaveStatus = "approved"
	StatusRejected LeaveStatus = "rejected"
)

type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      Role   `json:"role"`
	ManagerID string `json:"managerId,omitempty"`
}

type LeaveBalance struct {
	Annual   float64 `json:"annual"`
	Sick     float64 `json:"sick"`
	Personal float64 `json:"personal"`
}

type LeaveRequest struct {
	ID              string      `json:"id"`
	EmployeeID      string      `json:"employeeId"`
	EmployeeName    string      `json:"employeeName"`
	StartDate       string      `json:"startDate"`
	EndDate         string      `json:"endDate"`
	LeaveType       LeaveType   `json:"leaveType"`
	Reason          string      `json:"reason,omitempty"`
	Status          LeaveStatus `json:"status"`
	CreatedAt       time.Time   `json:"createdAt"`
	ReviewedAt      *time.Time  `json:"reviewedAt,omitempty"`
	ReviewedBy      string      `json:"reviewedBy,omitempty"`
	RejectionReason string      `json:"rejectionReason,omitempty"`
}

func IsValidLeaveType(t string) bool {
	switch LeaveType(t) {
	case LeaveAnnual, LeaveSick, LeavePersonal:
		return true
	}
	return false
}

func IsValidStatus(s string) bool {
	switch LeaveStatus(s) {
	case StatusPending, StatusApproved, StatusRejected:
		return true
	}
	return false
}
