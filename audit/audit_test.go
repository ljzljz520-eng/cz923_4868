package audit

import (
	"testing"
	"time"

	"pharmacycounter/domain"
)

func TestAuditCSVAndTimeline(t *testing.T) {
	at := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	entries := []domain.AuditEntry{{ID: "a2", Action: ActionCalled, SubjectID: "o1", Operator: "pharmacist", Detail: "called", CreatedAt: at.Add(time.Minute)}, {ID: "a1", Action: ActionRegistered, SubjectID: "o1", Operator: "pharmacist", Detail: "registered", CreatedAt: at}}
	if err := ValidateTimeline(entries); err != nil {
		t.Fatal(err)
	}
	data, err := ExportCSV(entries)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := ImportCSV(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 || loaded[0].ID != "a1" {
		t.Fatalf("unexpected entries: %+v", loaded)
	}
}
