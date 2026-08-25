package query

import (
	"testing"
	"time"

	"pharmacycounter/domain"
)

func TestBuildDailySummaries(t *testing.T) {
	at := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	orders := []domain.PickupOrder{{ID: "1", State: domain.StateCompleted, CreatedAt: at, UpdatedAt: at.Add(10 * time.Minute), Items: []domain.PrescriptionItem{{Quantity: 2}}}, {ID: "2", State: domain.StateCancelled, CreatedAt: at, UpdatedAt: at.Add(time.Minute), Items: []domain.PrescriptionItem{{Quantity: 1}}}}
	summaries := BuildDailySummaries(orders)
	if len(summaries) != 1 || summaries[0].Registered != 2 || summaries[0].Completed != 1 || summaries[0].MedicineUnits != 3 {
		t.Fatalf("unexpected summaries: %+v", summaries)
	}
	if summaries[0].AverageTurnaround != 10*time.Minute {
		t.Fatalf("unexpected average: %s", summaries[0].AverageTurnaround)
	}
}

func TestBuildMedicineUsage(t *testing.T) {
	orders := []domain.PickupOrder{{State: domain.StateCompleted, Items: []domain.PrescriptionItem{{MedicineCode: "MED001", Name: "药品一", Quantity: 2}}}, {State: domain.StateWaiting, Items: []domain.PrescriptionItem{{MedicineCode: "MED001", Name: "药品一", Quantity: 4}}}}
	usage := BuildMedicineUsage(orders)
	if len(usage) != 1 || usage[0].Quantity != 2 || usage[0].Orders != 1 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}
