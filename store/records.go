package store

import (
	"fmt"
	"sort"

	"pharmacycounter/domain"
)

func (tx *Transaction) CreateCall(record domain.CallRecord) error {
	bucket := tx.tx.Bucket(bucketCalls)
	if bucket.Get([]byte(record.ID)) != nil {
		return domain.ErrDuplicate
	}
	if err := put(bucket, record.ID, record); err != nil {
		return err
	}
	return tx.putIndex("active-call:"+record.PickupOrderID, record.ID)
}

func (tx *Transaction) SaveCall(record domain.CallRecord) error {
	bucket := tx.tx.Bucket(bucketCalls)
	if bucket.Get([]byte(record.ID)) == nil {
		return domain.ErrNotFound
	}
	return put(bucket, record.ID, record)
}

func (tx *Transaction) GetCall(id string) (domain.CallRecord, error) {
	var record domain.CallRecord
	err := get(tx.tx.Bucket(bucketCalls), id, &record)
	return record, err
}

func (tx *Transaction) GetActiveCall(orderID string) (domain.CallRecord, error) {
	id := string(tx.tx.Bucket(bucketIndexes).Get([]byte("active-call:" + orderID)))
	if id == "" {
		return domain.CallRecord{}, domain.ErrNotFound
	}
	return tx.GetCall(id)
}

func (tx *Transaction) ListCalls() ([]domain.CallRecord, error) {
	records, err := collect[domain.CallRecord](tx.tx.Bucket(bucketCalls))
	if err != nil {
		return nil, err
	}
	sort.SliceStable(records, func(left, right int) bool {
		if records[left].Sequence != records[right].Sequence {
			return records[left].Sequence < records[right].Sequence
		}
		return records[left].ID < records[right].ID
	})
	return records, nil
}

func (tx *Transaction) NextCallSequence() (int, error) {
	records, err := tx.ListCalls()
	if err != nil {
		return 0, err
	}
	maximum := 0
	for _, record := range records {
		if record.Sequence > maximum {
			maximum = record.Sequence
		}
	}
	return maximum + 1, nil
}

func (tx *Transaction) CreateDispense(record domain.DispenseRecord) error {
	bucket := tx.tx.Bucket(bucketDispenses)
	if bucket.Get([]byte(record.ID)) != nil {
		return domain.ErrDuplicate
	}
	if existing := tx.tx.Bucket(bucketIndexes).Get([]byte("dispense:" + record.PickupOrderID)); existing != nil {
		return fmt.Errorf("取药单已发药: %w", domain.ErrDuplicate)
	}
	if err := put(bucket, record.ID, record); err != nil {
		return err
	}
	return tx.putIndex("dispense:"+record.PickupOrderID, record.ID)
}

func (tx *Transaction) GetDispense(id string) (domain.DispenseRecord, error) {
	var record domain.DispenseRecord
	err := get(tx.tx.Bucket(bucketDispenses), id, &record)
	return record, err
}

func (tx *Transaction) GetDispenseByOrder(orderID string) (domain.DispenseRecord, error) {
	id := string(tx.tx.Bucket(bucketIndexes).Get([]byte("dispense:" + orderID)))
	if id == "" {
		return domain.DispenseRecord{}, domain.ErrNotFound
	}
	return tx.GetDispense(id)
}

func (tx *Transaction) ListDispenses() ([]domain.DispenseRecord, error) {
	records, err := collect[domain.DispenseRecord](tx.tx.Bucket(bucketDispenses))
	if err != nil {
		return nil, err
	}
	sort.SliceStable(records, func(left, right int) bool {
		if !records[left].CompletedAt.Equal(records[right].CompletedAt) {
			return records[left].CompletedAt.Before(records[right].CompletedAt)
		}
		return records[left].ID < records[right].ID
	})
	return records, nil
}

func (tx *Transaction) CreateAudit(entry domain.AuditEntry) error {
	bucket := tx.tx.Bucket(bucketAudits)
	if bucket.Get([]byte(entry.ID)) != nil {
		return domain.ErrDuplicate
	}
	return put(bucket, entry.ID, entry)
}

func (tx *Transaction) ListAudits() ([]domain.AuditEntry, error) {
	entries, err := collect[domain.AuditEntry](tx.tx.Bucket(bucketAudits))
	if err != nil {
		return nil, err
	}
	sort.SliceStable(entries, func(left, right int) bool {
		if !entries[left].CreatedAt.Equal(entries[right].CreatedAt) {
			return entries[left].CreatedAt.Before(entries[right].CreatedAt)
		}
		return entries[left].ID < entries[right].ID
	})
	return entries, nil
}

func (tx *Transaction) ListAuditsBySubject(subjectID string) ([]domain.AuditEntry, error) {
	entries, err := tx.ListAudits()
	if err != nil {
		return nil, err
	}
	result := make([]domain.AuditEntry, 0)
	for _, entry := range entries {
		if entry.SubjectID == subjectID {
			result = append(result, entry)
		}
	}
	return result, nil
}
