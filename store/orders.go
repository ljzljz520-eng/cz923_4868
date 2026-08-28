package store

import (
	"errors"
	"fmt"

	"pharmacycounter/domain"
)

func (tx *Transaction) CreateOrder(order domain.PickupOrder) error {
	bucket := tx.tx.Bucket(bucketOrders)
	if bucket.Get([]byte(order.ID)) != nil {
		return domain.ErrDuplicate
	}
	if existing := tx.findOrderByPrescription(order.PrescriptionID); existing != "" {
		return fmt.Errorf("处方 %s: %w", order.PrescriptionID, domain.ErrDuplicate)
	}
	if err := put(bucket, order.ID, order); err != nil {
		return err
	}
	return tx.putIndex("prescription:"+order.PrescriptionID, order.ID)
}

func (tx *Transaction) SaveOrder(order domain.PickupOrder) error {
	bucket := tx.tx.Bucket(bucketOrders)
	var current domain.PickupOrder
	if err := get(bucket, order.ID, &current); err != nil {
		return err
	}
	if order.Version != current.Version+1 {
		return domain.ErrConflict
	}
	if current.PrescriptionID != order.PrescriptionID {
		return errors.New("处方标识不可修改")
	}
	return put(bucket, order.ID, order)
}

func (tx *Transaction) ForceSaveOrder(order domain.PickupOrder) error {
	bucket := tx.tx.Bucket(bucketOrders)
	if bucket.Get([]byte(order.ID)) == nil {
		return domain.ErrNotFound
	}
	return put(bucket, order.ID, order)
}

func (tx *Transaction) GetOrder(id string) (domain.PickupOrder, error) {
	var order domain.PickupOrder
	err := get(tx.tx.Bucket(bucketOrders), id, &order)
	if err != nil {
		return domain.PickupOrder{}, err
	}
	return domain.CloneOrder(order), nil
}

func (tx *Transaction) GetOrderByPrescription(prescriptionID string) (domain.PickupOrder, error) {
	id := tx.findOrderByPrescription(prescriptionID)
	if id == "" {
		return domain.PickupOrder{}, domain.ErrNotFound
	}
	return tx.GetOrder(id)
}

func (tx *Transaction) findOrderByPrescription(prescriptionID string) string {
	value := tx.tx.Bucket(bucketIndexes).Get([]byte("prescription:" + prescriptionID))
	return string(value)
}

func (tx *Transaction) ListOrders() ([]domain.PickupOrder, error) {
	orders, err := collect[domain.PickupOrder](tx.tx.Bucket(bucketOrders))
	if err != nil {
		return nil, err
	}
	return domain.SortOrders(orders), nil
}

func (tx *Transaction) FilterOrders(filter domain.OrderFilter) ([]domain.PickupOrder, error) {
	orders, err := tx.ListOrders()
	if err != nil {
		return nil, err
	}
	return domain.FilterOrders(orders, filter), nil
}

func (tx *Transaction) DeleteOrder(id string) error {
	order, err := tx.GetOrder(id)
	if err != nil {
		return err
	}
	if order.State != domain.StateCancelled {
		return errors.New("只能删除已取消取药单")
	}
	if err := tx.tx.Bucket(bucketOrders).Delete([]byte(id)); err != nil {
		return err
	}
	return tx.tx.Bucket(bucketIndexes).Delete([]byte("prescription:" + order.PrescriptionID))
}

func (tx *Transaction) CountOrdersByState() (map[domain.OrderState]int, error) {
	orders, err := tx.ListOrders()
	if err != nil {
		return nil, err
	}
	counts := map[domain.OrderState]int{
		domain.StateWaiting:   0,
		domain.StateCalled:    0,
		domain.StateCompleted: 0,
		domain.StateCancelled: 0,
	}
	for _, order := range orders {
		counts[order.State]++
	}
	return counts, nil
}

func (tx *Transaction) putIndex(key, value string) error {
	bucket := tx.tx.Bucket(bucketIndexes)
	if current := bucket.Get([]byte(key)); current != nil && string(current) != value {
		return domain.ErrDuplicate
	}
	return bucket.Put([]byte(key), []byte(value))
}
