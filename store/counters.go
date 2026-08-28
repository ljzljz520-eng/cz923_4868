package store

import (
	"sort"

	"pharmacycounter/domain"
)

func (tx *Transaction) SaveCounter(counter domain.Counter) error {
	if counter.Code == "" || counter.Name == "" {
		return domain.ErrValidation
	}
	if counter.QueueLimit < 1 {
		return domain.ValidationError{Field: "queueLimit", Message: "必须大于零"}
	}
	return put(tx.tx.Bucket(bucketCounters), counter.Code, counter)
}

func (tx *Transaction) GetCounter(code string) (domain.Counter, error) {
	var counter domain.Counter
	err := get(tx.tx.Bucket(bucketCounters), code, &counter)
	return counter, err
}

func (tx *Transaction) ListCounters() ([]domain.Counter, error) {
	counters, err := collect[domain.Counter](tx.tx.Bucket(bucketCounters))
	if err != nil {
		return nil, err
	}
	sort.SliceStable(counters, func(left, right int) bool {
		return counters[left].Code < counters[right].Code
	})
	return counters, nil
}

func (tx *Transaction) EnabledCounters() ([]domain.Counter, error) {
	counters, err := tx.ListCounters()
	if err != nil {
		return nil, err
	}
	result := make([]domain.Counter, 0)
	for _, counter := range counters {
		if counter.Enabled {
			result = append(result, counter)
		}
	}
	return result, nil
}

func (tx *Transaction) CounterLoad(code string) (int, error) {
	orders, err := tx.ListOrders()
	if err != nil {
		return 0, err
	}
	load := 0
	for _, order := range orders {
		if order.State != domain.StateCalled {
			continue
		}
		call, err := tx.GetActiveCall(order.ID)
		if err != nil {
			return 0, err
		}
		if call.CounterCode == code {
			load++
		}
	}
	return load, nil
}

func (tx *Transaction) SeedCounters(counters []domain.Counter) error {
	for _, counter := range counters {
		if _, err := tx.GetCounter(counter.Code); err == nil {
			continue
		} else if err != domain.ErrNotFound {
			return err
		}
		if err := tx.SaveCounter(counter); err != nil {
			return err
		}
	}
	return nil
}
