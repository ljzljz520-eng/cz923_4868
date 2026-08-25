package store

import (
	"path/filepath"
	"testing"
	"time"

	"pharmacycounter/domain"
)

func TestPersistenceSurvivesReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pharmacy.db")
	storage, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	order := domain.PickupOrder{ID: "order-reopen", TicketNumber: "A018", PatientName: "王芳", PatientCode: "P018", PrescriptionID: "RX018", Priority: domain.PriorityRoutine, State: domain.StateWaiting, CreatedAt: at, UpdatedAt: at, Version: 1, Items: []domain.PrescriptionItem{{ID: "item-1", PickupOrderID: "order-reopen", MedicineCode: "MED001", Name: "阿莫西林", Specification: "0.25g", Quantity: 1, Unit: "盒", Lot: "L018"}}}
	if err := storage.Update(func(tx *Transaction) error { return tx.CreateOrder(order) }); err != nil {
		t.Fatal(err)
	}
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}
	storage, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	if err := storage.View(func(tx *Transaction) error {
		loaded, err := tx.GetOrder(order.ID)
		if err != nil {
			return err
		}
		if loaded.PrescriptionID != order.PrescriptionID || len(loaded.Items) != 1 {
			t.Fatalf("unexpected persisted order: %+v", loaded)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestStoreSnapshotIncludesEntities(t *testing.T) {
	storage, err := Open(filepath.Join(t.TempDir(), "snapshot.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	if err := storage.Update(func(tx *Transaction) error {
		return tx.SaveCounter(domain.Counter{Code: "C01", Name: "窗口一", Enabled: true, QueueLimit: 3})
	}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := storage.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Counters) != 1 {
		t.Fatalf("unexpected counters: %+v", snapshot.Counters)
	}
}
