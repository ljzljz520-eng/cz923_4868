package query

import (
	"sort"
	"strings"

	"pharmacycounter/domain"
	"pharmacycounter/store"
)

type Service struct {
	store *store.Store
}

func New(storage *store.Store) *Service {
	return &Service{store: storage}
}

func (s *Service) Dashboard() (domain.Dashboard, error) {
	var dashboard domain.Dashboard
	err := s.store.View(func(tx *store.Transaction) error {
		orders, err := tx.ListOrders()
		if err != nil {
			return err
		}
		for _, order := range orders {
			switch order.State {
			case domain.StateWaiting:
				dashboard.Waiting = append(dashboard.Waiting, order)
			case domain.StateCalled:
				dashboard.Called = append(dashboard.Called, order)
			case domain.StateCompleted:
				dashboard.Completed = append(dashboard.Completed, order)
			}
		}
		dashboard.WaitingCount = len(dashboard.Waiting)
		dashboard.CalledCount = len(dashboard.Called)
		dashboard.CompletedCount = len(dashboard.Completed)
		return nil
	})
	return dashboard, err
}

func (s *Service) Orders(filter domain.OrderFilter) ([]domain.PickupOrder, error) {
	var orders []domain.PickupOrder
	err := s.store.View(func(tx *store.Transaction) error {
		var err error
		orders, err = tx.FilterOrders(filter)
		return err
	})
	return orders, err
}

func (s *Service) OrderDetail(id string) (domain.PickupOrder, []domain.AuditEntry, error) {
	var order domain.PickupOrder
	var entries []domain.AuditEntry
	err := s.store.View(func(tx *store.Transaction) error {
		var err error
		order, err = tx.GetOrder(id)
		if err != nil {
			return err
		}
		entries, err = tx.ListAuditsBySubject(id)
		return err
	})
	return order, entries, err
}

func SearchMedicines(orders []domain.PickupOrder, value string) []domain.PrescriptionItem {
	query := strings.ToLower(strings.TrimSpace(value))
	result := make([]domain.PrescriptionItem, 0)
	seen := make(map[string]bool)
	for _, order := range orders {
		for _, item := range order.Items {
			if query != "" && !strings.Contains(strings.ToLower(item.Name), query) && !strings.Contains(strings.ToLower(item.MedicineCode), query) {
				continue
			}
			key := item.MedicineCode + ":" + item.Lot
			if seen[key] {
				continue
			}
			seen[key] = true
			result = append(result, item)
		}
	}
	sort.SliceStable(result, func(left, right int) bool {
		if result[left].MedicineCode != result[right].MedicineCode {
			return result[left].MedicineCode < result[right].MedicineCode
		}
		return result[left].Lot < result[right].Lot
	})
	return result
}
