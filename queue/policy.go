package queue

import (
	"errors"
	"sort"
	"strings"

	"pharmacycounter/domain"
)

type Policy struct {
	UrgentWeight  int
	SeniorWeight  int
	RoutineWeight int
	MaxRecall     int
}

func DefaultPolicy() Policy {
	return Policy{UrgentWeight: 300, SeniorWeight: 200, RoutineWeight: 100, MaxRecall: 2}
}

func (p Policy) Validate() error {
	if p.UrgentWeight <= p.SeniorWeight {
		return errors.New("紧急优先权重必须高于优先权重")
	}
	if p.SeniorWeight <= p.RoutineWeight {
		return errors.New("优先权重必须高于普通权重")
	}
	if p.RoutineWeight < 0 {
		return errors.New("普通权重不能为负数")
	}
	if p.MaxRecall < 0 {
		return errors.New("重叫次数不能为负数")
	}
	return nil
}

func (p Policy) Score(order domain.PickupOrder) int {
	switch order.Priority {
	case domain.PriorityUrgent:
		return p.UrgentWeight
	case domain.PrioritySenior:
		return p.SeniorWeight
	default:
		return p.RoutineWeight
	}
}

func (p Policy) Rank(orders []domain.PickupOrder) ([]domain.PickupOrder, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	result := make([]domain.PickupOrder, 0, len(orders))
	for _, order := range orders {
		if order.State == domain.StateWaiting {
			result = append(result, domain.CloneOrder(order))
		}
	}
	sort.SliceStable(result, func(left, right int) bool {
		leftScore := p.Score(result[left])
		rightScore := p.Score(result[right])
		if leftScore != rightScore {
			return leftScore > rightScore
		}
		if !result[left].CreatedAt.Equal(result[right].CreatedAt) {
			return result[left].CreatedAt.Before(result[right].CreatedAt)
		}
		return result[left].TicketNumber < result[right].TicketNumber
	})
	return result, nil
}

func (p Policy) Next(orders []domain.PickupOrder) (domain.PickupOrder, error) {
	ranked, err := p.Rank(orders)
	if err != nil {
		return domain.PickupOrder{}, err
	}
	if len(ranked) == 0 {
		return domain.PickupOrder{}, domain.ErrNotFound
	}
	return ranked[0], nil
}

func TicketPrefix(priority domain.Priority) string {
	switch priority {
	case domain.PriorityUrgent:
		return "E"
	case domain.PrioritySenior:
		return "P"
	default:
		return "A"
	}
}

func NormalizeCounterCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func CanRecall(record domain.CallRecord, existing []domain.CallRecord, maximum int) bool {
	count := 0
	for _, current := range existing {
		if current.PickupOrderID == record.PickupOrderID && current.Recalled {
			count++
		}
	}
	return count < maximum
}

func PendingForCounter(orders []domain.PickupOrder, calls []domain.CallRecord, code string) []domain.PickupOrder {
	callByOrder := make(map[string]domain.CallRecord)
	for _, call := range calls {
		callByOrder[call.PickupOrderID] = call
	}
	result := make([]domain.PickupOrder, 0)
	for _, order := range orders {
		if order.State != domain.StateCalled {
			continue
		}
		if call, ok := callByOrder[order.ID]; ok && call.CounterCode == code {
			result = append(result, domain.CloneOrder(order))
		}
	}
	return result
}
