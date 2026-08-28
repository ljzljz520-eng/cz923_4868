package query

import (
	"sort"
	"time"

	"pharmacycounter/domain"
)

type DailySummary struct {
	Day               string        `json:"day"`
	Registered        int           `json:"registered"`
	Completed         int           `json:"completed"`
	Cancelled         int           `json:"cancelled"`
	MedicineUnits     int           `json:"medicineUnits"`
	AverageTurnaround time.Duration `json:"averageTurnaround"`
}

type MedicineUsage struct {
	MedicineCode string `json:"medicineCode"`
	Name         string `json:"name"`
	Quantity     int    `json:"quantity"`
	Orders       int    `json:"orders"`
}

func BuildDailySummaries(orders []domain.PickupOrder) []DailySummary {
	type accumulator struct {
		summary       DailySummary
		duration      time.Duration
		durationCount int
	}
	byDay := make(map[string]*accumulator)
	for _, order := range orders {
		day := domain.DayKey(order.CreatedAt)
		entry := byDay[day]
		if entry == nil {
			entry = &accumulator{summary: DailySummary{Day: day}}
			byDay[day] = entry
		}
		entry.summary.Registered++
		entry.summary.MedicineUnits += domain.TotalQuantity(order.Items)
		switch order.State {
		case domain.StateCompleted:
			entry.summary.Completed++
			duration := domain.OrderDuration(order)
			if duration > 0 {
				entry.duration += duration
				entry.durationCount++
			}
		case domain.StateCancelled:
			entry.summary.Cancelled++
		}
	}
	result := make([]DailySummary, 0, len(byDay))
	for _, entry := range byDay {
		if entry.durationCount > 0 {
			entry.summary.AverageTurnaround = entry.duration / time.Duration(entry.durationCount)
		}
		result = append(result, entry.summary)
	}
	sort.SliceStable(result, func(left, right int) bool {
		return result[left].Day < result[right].Day
	})
	return result
}

func BuildMedicineUsage(orders []domain.PickupOrder) []MedicineUsage {
	usage := make(map[string]*MedicineUsage)
	for _, order := range orders {
		if order.State != domain.StateCompleted {
			continue
		}
		seen := make(map[string]bool)
		for _, item := range order.Items {
			entry := usage[item.MedicineCode]
			if entry == nil {
				entry = &MedicineUsage{MedicineCode: item.MedicineCode, Name: item.Name}
				usage[item.MedicineCode] = entry
			}
			entry.Quantity += item.Quantity
			if !seen[item.MedicineCode] {
				entry.Orders++
				seen[item.MedicineCode] = true
			}
		}
	}
	result := make([]MedicineUsage, 0, len(usage))
	for _, entry := range usage {
		result = append(result, *entry)
	}
	sort.SliceStable(result, func(left, right int) bool {
		if result[left].Quantity != result[right].Quantity {
			return result[left].Quantity > result[right].Quantity
		}
		return result[left].MedicineCode < result[right].MedicineCode
	})
	return result
}

func CompletionRate(summary DailySummary) float64 {
	if summary.Registered == 0 {
		return 0
	}
	return float64(summary.Completed) / float64(summary.Registered)
}

func PeakDay(summaries []DailySummary) (DailySummary, bool) {
	if len(summaries) == 0 {
		return DailySummary{}, false
	}
	peak := summaries[0]
	for _, summary := range summaries[1:] {
		if summary.Registered > peak.Registered {
			peak = summary
		}
	}
	return peak, true
}
