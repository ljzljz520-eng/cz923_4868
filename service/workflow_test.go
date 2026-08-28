package service

import (
	"path/filepath"
	"testing"
	"time"

	"pharmacycounter/domain"
	"pharmacycounter/queue"
	"pharmacycounter/store"
)

func workflowService(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	storage, err := store.Open(filepath.Join(t.TempDir(), "workflow.db"))
	if err != nil {
		t.Fatal(err)
	}
	business, err := New(storage, queue.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if err := business.SeedCounters([]domain.Counter{{Code: "C01", Name: "窗口一", Enabled: true, QueueLimit: 3}}); err != nil {
		t.Fatal(err)
	}
	return business, storage
}

func TestWorkflowRegisterAndList(t *testing.T) {
	business, storage := workflowService(t)
	defer storage.Close()
	at := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	command := domain.CreateOrderCommand{ID: "order-register", TicketNumber: "A001", PatientName: "张三", PatientCode: "P001", PrescriptionID: "RX001", Priority: domain.PriorityRoutine, CreatedAt: at, Items: []domain.PrescriptionItem{{MedicineCode: "MED001", Name: "阿莫西林", Specification: "0.25g", Quantity: 1, Unit: "盒", Lot: "L001"}}}
	order, err := business.Register(command, "pharmacist-a")
	if err != nil {
		t.Fatal(err)
	}
	if order.State != domain.StateWaiting {
		t.Fatalf("unexpected state: %s", order.State)
	}
	if err := storage.View(func(tx *store.Transaction) error {
		orders, err := tx.ListOrders()
		if err != nil {
			return err
		}
		if len(orders) != 1 || orders[0].ID != command.ID {
			t.Fatalf("unexpected orders: %+v", orders)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowCallAndQueue(t *testing.T) {
	business, storage := workflowService(t)
	defer storage.Close()
	at := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	for _, command := range []domain.CreateOrderCommand{{ID: "routine", TicketNumber: "A001", PatientName: "张三", PatientCode: "P001", PrescriptionID: "RX001", Priority: domain.PriorityRoutine, CreatedAt: at, Items: []domain.PrescriptionItem{{MedicineCode: "MED001", Name: "药品", Specification: "规格", Quantity: 1, Unit: "盒", Lot: "L1"}}}, {ID: "urgent", TicketNumber: "E001", PatientName: "李四", PatientCode: "P002", PrescriptionID: "RX002", Priority: domain.PriorityUrgent, CreatedAt: at.Add(time.Minute), Items: []domain.PrescriptionItem{{MedicineCode: "MED002", Name: "药品", Specification: "规格", Quantity: 1, Unit: "盒", Lot: "L2"}}}} {
		if _, err := business.Register(command, "pharmacist-a"); err != nil {
			t.Fatal(err)
		}
	}
	called, record, err := business.CallNext(domain.CallCommand{OrderID: "urgent", CounterCode: "C01", Operator: "pharmacist-b", CalledAt: at.Add(2 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if called.ID != "urgent" || called.State != domain.StateCalled || record.CounterCode != "C01" {
		t.Fatalf("unexpected call result: %+v %+v", called, record)
	}
}
