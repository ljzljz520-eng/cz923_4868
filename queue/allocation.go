package queue

import (
	"errors"
	"sort"

	"pharmacycounter/domain"
)

type CounterLoad struct {
	Counter domain.Counter
	Active  int
}

func AvailableCounters(counters []domain.Counter, loads map[string]int) []CounterLoad {
	result := make([]CounterLoad, 0)
	for _, counter := range counters {
		if !counter.Enabled {
			continue
		}
		active := loads[counter.Code]
		if active >= counter.QueueLimit {
			continue
		}
		result = append(result, CounterLoad{Counter: counter, Active: active})
	}
	sort.SliceStable(result, func(left, right int) bool {
		if result[left].Active != result[right].Active {
			return result[left].Active < result[right].Active
		}
		return result[left].Counter.Code < result[right].Counter.Code
	})
	return result
}

func ChooseCounter(counters []domain.Counter, loads map[string]int, requested string) (domain.Counter, error) {
	available := AvailableCounters(counters, loads)
	if requested != "" {
		requested = NormalizeCounterCode(requested)
		for _, candidate := range available {
			if candidate.Counter.Code == requested {
				return candidate.Counter, nil
			}
		}
		return domain.Counter{}, errors.New("指定窗口不可用")
	}
	if len(available) == 0 {
		return domain.Counter{}, errors.New("没有可用发药窗口")
	}
	return available[0].Counter, nil
}

func BuildLoads(counters []domain.Counter, orders []domain.PickupOrder, calls []domain.CallRecord) map[string]int {
	loads := make(map[string]int)
	for _, counter := range counters {
		loads[counter.Code] = 0
	}
	active := make(map[string]bool)
	for _, order := range orders {
		if order.State == domain.StateCalled {
			active[order.ID] = true
		}
	}
	for _, call := range calls {
		if active[call.PickupOrderID] {
			loads[call.CounterCode]++
		}
	}
	return loads
}

func GroupWaitingByPriority(orders []domain.PickupOrder) map[domain.Priority][]domain.PickupOrder {
	groups := map[domain.Priority][]domain.PickupOrder{
		domain.PriorityUrgent:  {},
		domain.PrioritySenior:  {},
		domain.PriorityRoutine: {},
	}
	for _, order := range orders {
		if order.State != domain.StateWaiting {
			continue
		}
		groups[order.Priority] = append(groups[order.Priority], domain.CloneOrder(order))
	}
	return groups
}

func QueuePosition(orderID string, ranked []domain.PickupOrder) int {
	for index, order := range ranked {
		if order.ID == orderID {
			return index + 1
		}
	}
	return 0
}
