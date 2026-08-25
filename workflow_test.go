package main

import (
	"path/filepath"
	"testing"
	"time"

	"pharmacycounter/domain"
	"pharmacycounter/queue"
	"pharmacycounter/service"
	"pharmacycounter/store"
)

func TestBusinessChain05(t *testing.T) {
	storage, err := store.Open(filepath.Join(t.TempDir(), "chain.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	business, err := service.New(storage, queue.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if err := business.SeedCounters([]domain.Counter{{Code: "C01", Name: "窗口一", Enabled: true, QueueLimit: 3}}); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	command := domain.CreateOrderCommand{ID: "order-chain-05", TicketNumber: "A005", PatientName: "赵敏", PatientCode: "P005", PrescriptionID: "RX005", Priority: domain.PriorityRoutine, CreatedAt: at, Items: []domain.PrescriptionItem{{MedicineCode: "MED001", Name: "阿莫西林", Specification: "0.25g", Quantity: 2, Unit: "盒", Lot: "LOT-05"}}}
	if _, err := business.Register(command, "pharmacist-a"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := business.CallNext(domain.CallCommand{OrderID: command.ID, CounterCode: "C01", Operator: "pharmacist-b", CalledAt: at.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	incomplete := domain.DispenseCommand{OrderID: command.ID, RecordID: "dispense-chain-05", CounterCode: "C01", Operator: "pharmacist-b", CompletedAt: at.Add(2 * time.Minute), Checks: []domain.VerificationCheck{{MedicineCode: "MED001", Lot: "WRONG-LOT", Quantity: 1, Confirmed: false}}}
	if _, _, err := business.CompleteDispense(incomplete); err == nil {
		t.Fatal("incomplete verification must fail")
	}
	if err := storage.View(func(tx *store.Transaction) error {
		order, err := tx.GetOrder(command.ID)
		if err != nil {
			return err
		}
		if order.State != domain.StateCalled {
			t.Fatalf("state changed after rejected dispense: %s", order.State)
		}
		if _, err := tx.GetDispenseByOrder(command.ID); err != domain.ErrNotFound {
			t.Fatalf("unexpected dispense record: %v", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
