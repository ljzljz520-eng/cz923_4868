package domain

import "time"

type OrderState string

const (
	StateWaiting   OrderState = "waiting"
	StateCalled    OrderState = "called"
	StateCompleted OrderState = "completed"
	StateCancelled OrderState = "cancelled"
)

type Priority string

const (
	PriorityRoutine Priority = "routine"
	PrioritySenior  Priority = "senior"
	PriorityUrgent  Priority = "urgent"
)

type PickupOrder struct {
	ID             string             `json:"id"`
	TicketNumber   string             `json:"ticketNumber"`
	PatientName    string             `json:"patientName"`
	PatientCode    string             `json:"patientCode"`
	PrescriptionID string             `json:"prescriptionId"`
	Priority       Priority           `json:"priority"`
	State          OrderState         `json:"state"`
	CreatedAt      time.Time          `json:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt"`
	Version        int                `json:"version"`
	Items          []PrescriptionItem `json:"items"`
	Warnings       []SafetyWarning    `json:"warnings"`
}

type PrescriptionItem struct {
	ID            string `json:"id"`
	PickupOrderID string `json:"pickupOrderId"`
	MedicineCode  string `json:"medicineCode"`
	Name          string `json:"name"`
	Specification string `json:"specification"`
	Quantity      int    `json:"quantity"`
	Unit          string `json:"unit"`
	Lot           string `json:"lot"`
	Location      string `json:"location"`
	Verified      bool   `json:"verified"`
}

type CallRecord struct {
	ID            string    `json:"id"`
	PickupOrderID string    `json:"pickupOrderId"`
	TicketNumber  string    `json:"ticketNumber"`
	CounterCode   string    `json:"counterCode"`
	Operator      string    `json:"operator"`
	Sequence      int       `json:"sequence"`
	CalledAt      time.Time `json:"calledAt"`
	Recalled      bool      `json:"recalled"`
}

type DispenseRecord struct {
	ID            string              `json:"id"`
	PickupOrderID string              `json:"pickupOrderId"`
	Operator      string              `json:"operator"`
	CounterCode   string              `json:"counterCode"`
	CompletedAt   time.Time           `json:"completedAt"`
	Note          string              `json:"note"`
	Checks        []VerificationCheck `json:"checks"`
	ItemCount     int                 `json:"itemCount"`
}

type VerificationCheck struct {
	MedicineCode string `json:"medicineCode"`
	Lot          string `json:"lot"`
	Quantity     int    `json:"quantity"`
	Confirmed    bool   `json:"confirmed"`
}

type SafetyWarning struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Resolved bool   `json:"resolved"`
}

type AuditEntry struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	SubjectID string    `json:"subjectId"`
	Operator  string    `json:"operator"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"createdAt"`
}

type Counter struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
	QueueLimit int    `json:"queueLimit"`
}

type CreateOrderCommand struct {
	ID             string             `json:"id"`
	TicketNumber   string             `json:"ticketNumber"`
	PatientName    string             `json:"patientName"`
	PatientCode    string             `json:"patientCode"`
	PrescriptionID string             `json:"prescriptionId"`
	Priority       Priority           `json:"priority"`
	CreatedAt      time.Time          `json:"createdAt"`
	Items          []PrescriptionItem `json:"items"`
}

type CallCommand struct {
	OrderID     string    `json:"orderId"`
	CounterCode string    `json:"counterCode"`
	Operator    string    `json:"operator"`
	CalledAt    time.Time `json:"calledAt"`
}

type DispenseCommand struct {
	OrderID     string              `json:"orderId"`
	RecordID    string              `json:"recordId"`
	CounterCode string              `json:"counterCode"`
	Operator    string              `json:"operator"`
	CompletedAt time.Time           `json:"completedAt"`
	Note        string              `json:"note"`
	Checks      []VerificationCheck `json:"checks"`
}

type OrderFilter struct {
	States       []OrderState
	PatientQuery string
	CounterCode  string
	Priority     Priority
	CreatedFrom  time.Time
	CreatedTo    time.Time
	Limit        int
	Offset       int
}

type Dashboard struct {
	Waiting        []PickupOrder `json:"waiting"`
	Called         []PickupOrder `json:"called"`
	Completed      []PickupOrder `json:"completed"`
	WaitingCount   int           `json:"waitingCount"`
	CalledCount    int           `json:"calledCount"`
	CompletedCount int           `json:"completedCount"`
}
