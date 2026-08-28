package store

import "pharmacycounter/domain"

type Snapshot struct {
	Orders    []domain.PickupOrder    `json:"orders"`
	Calls     []domain.CallRecord     `json:"calls"`
	Dispenses []domain.DispenseRecord `json:"dispenses"`
	Audits    []domain.AuditEntry     `json:"audits"`
	Counters  []domain.Counter        `json:"counters"`
}

func (s *Store) Snapshot() (Snapshot, error) {
	var snapshot Snapshot
	err := s.View(func(tx *Transaction) error {
		var err error
		snapshot.Orders, err = tx.ListOrders()
		if err != nil {
			return err
		}
		snapshot.Calls, err = tx.ListCalls()
		if err != nil {
			return err
		}
		snapshot.Dispenses, err = tx.ListDispenses()
		if err != nil {
			return err
		}
		snapshot.Audits, err = tx.ListAudits()
		if err != nil {
			return err
		}
		snapshot.Counters, err = tx.ListCounters()
		return err
	})
	return snapshot, err
}

func (s *Store) Health() error {
	return s.View(func(tx *Transaction) error {
		_, err := tx.CountOrdersByState()
		return err
	})
}
