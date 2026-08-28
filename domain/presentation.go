package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func PriorityRank(priority Priority) int {
	switch priority {
	case PriorityUrgent:
		return 0
	case PrioritySenior:
		return 1
	default:
		return 2
	}
}

func StateLabel(state OrderState) string {
	switch state {
	case StateWaiting:
		return "待取药"
	case StateCalled:
		return "已叫号"
	case StateCompleted:
		return "已完成"
	case StateCancelled:
		return "已取消"
	default:
		return "未知"
	}
}

func PriorityLabel(priority Priority) string {
	switch priority {
	case PriorityUrgent:
		return "紧急"
	case PrioritySenior:
		return "优先"
	default:
		return "普通"
	}
}

func ItemSummary(items []PrescriptionItem) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s %d%s", item.Name, item.Quantity, item.Unit))
	}
	return strings.Join(parts, "、")
}

func TotalQuantity(items []PrescriptionItem) int {
	total := 0
	for _, item := range items {
		total += item.Quantity
	}
	return total
}

func HasUnresolvedWarning(order PickupOrder) bool {
	for _, warning := range order.Warnings {
		if !warning.Resolved {
			return true
		}
	}
	return false
}

func CloneOrder(order PickupOrder) PickupOrder {
	copyOrder := order
	copyOrder.Items = append([]PrescriptionItem(nil), order.Items...)
	copyOrder.Warnings = append([]SafetyWarning(nil), order.Warnings...)
	return copyOrder
}

func SortOrders(orders []PickupOrder) []PickupOrder {
	result := make([]PickupOrder, len(orders))
	for index, order := range orders {
		result[index] = CloneOrder(order)
	}
	sort.SliceStable(result, func(left, right int) bool {
		leftRank := PriorityRank(result[left].Priority)
		rightRank := PriorityRank(result[right].Priority)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if !result[left].CreatedAt.Equal(result[right].CreatedAt) {
			return result[left].CreatedAt.Before(result[right].CreatedAt)
		}
		return result[left].TicketNumber < result[right].TicketNumber
	})
	return result
}

func MatchesFilter(order PickupOrder, filter OrderFilter) bool {
	if len(filter.States) > 0 {
		matched := false
		for _, state := range filter.States {
			if order.State == state {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	query := strings.ToLower(strings.TrimSpace(filter.PatientQuery))
	if query != "" && !strings.Contains(strings.ToLower(order.PatientName), query) && !strings.Contains(strings.ToLower(order.PatientCode), query) {
		return false
	}
	if filter.Priority != "" && order.Priority != filter.Priority {
		return false
	}
	if !filter.CreatedFrom.IsZero() && order.CreatedAt.Before(filter.CreatedFrom) {
		return false
	}
	if !filter.CreatedTo.IsZero() && order.CreatedAt.After(filter.CreatedTo) {
		return false
	}
	return true
}

func FilterOrders(orders []PickupOrder, filter OrderFilter) []PickupOrder {
	result := make([]PickupOrder, 0)
	for _, order := range orders {
		if MatchesFilter(order, filter) {
			result = append(result, CloneOrder(order))
		}
	}
	start := filter.Offset
	if start < 0 {
		start = 0
	}
	if start >= len(result) {
		return []PickupOrder{}
	}
	end := len(result)
	if filter.Limit > 0 && start+filter.Limit < end {
		end = start + filter.Limit
	}
	return result[start:end]
}

func DayKey(value time.Time) string {
	return value.Format("2006-01-02")
}

func OrderDuration(order PickupOrder) time.Duration {
	if order.UpdatedAt.IsZero() || order.UpdatedAt.Before(order.CreatedAt) {
		return 0
	}
	return order.UpdatedAt.Sub(order.CreatedAt)
}
