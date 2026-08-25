package domain

import (
	"testing"
	"time"
)

func TestValidateCreateOrderAndTransition(t *testing.T) {
	at := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	command := CreateOrderCommand{ID: "o1", TicketNumber: "A001", PatientName: "张三", PatientCode: "P001", PrescriptionID: "RX001", Priority: PriorityRoutine, CreatedAt: at, Items: []PrescriptionItem{{MedicineCode: "MED001", Name: "阿莫西林", Specification: "0.25g", Quantity: 1, Unit: "盒", Lot: "L01"}}}
	if err := ValidateCreateOrder(command); err != nil {
		t.Fatal(err)
	}
	order := PickupOrder{ID: command.ID, State: StateWaiting, CreatedAt: at, UpdatedAt: at, Version: 1}
	called, err := Transition(order, StateCalled, at.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if called.State != StateCalled || called.Version != 2 {
		t.Fatalf("unexpected transition: %+v", called)
	}
	if _, err := Transition(called, StateWaiting, at.Add(-time.Minute)); err == nil {
		t.Fatal("expected invalid time")
	}
}

func TestFilterOrders(t *testing.T) {
	at := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	orders := []PickupOrder{{ID: "1", PatientName: "张三", State: StateWaiting, Priority: PriorityRoutine, CreatedAt: at}, {ID: "2", PatientName: "李四", State: StateCalled, Priority: PriorityUrgent, CreatedAt: at}}
	result := FilterOrders(orders, OrderFilter{States: []OrderState{StateCalled}, PatientQuery: "李", Limit: 10})
	if len(result) != 1 || result[0].ID != "2" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
