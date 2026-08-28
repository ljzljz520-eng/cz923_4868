package queue

import (
	"testing"
	"time"

	"pharmacycounter/domain"
)

func TestPolicyRanksUrgentThenSenior(t *testing.T) {
	at := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	orders := []domain.PickupOrder{{ID: "routine", State: domain.StateWaiting, Priority: domain.PriorityRoutine, CreatedAt: at}, {ID: "senior", State: domain.StateWaiting, Priority: domain.PrioritySenior, CreatedAt: at.Add(time.Minute)}, {ID: "urgent", State: domain.StateWaiting, Priority: domain.PriorityUrgent, CreatedAt: at.Add(2 * time.Minute)}}
	ranked, err := DefaultPolicy().Rank(orders)
	if err != nil {
		t.Fatal(err)
	}
	if ranked[0].ID != "urgent" || ranked[1].ID != "senior" || ranked[2].ID != "routine" {
		t.Fatalf("unexpected ranking: %+v", ranked)
	}
}

func TestChooseCounterHonorsCapacity(t *testing.T) {
	counters := []domain.Counter{{Code: "C01", Name: "一", Enabled: true, QueueLimit: 1}, {Code: "C02", Name: "二", Enabled: true, QueueLimit: 3}}
	selected, err := ChooseCounter(counters, map[string]int{"C01": 1, "C02": 0}, "")
	if err != nil {
		t.Fatal(err)
	}
	if selected.Code != "C02" {
		t.Fatalf("unexpected counter: %+v", selected)
	}
}
