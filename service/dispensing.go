package service

import (
	"errors"

	"pharmacycounter/audit"
	"pharmacycounter/domain"
	"pharmacycounter/store"
)

func (s *Service) CompleteDispense(command domain.DispenseCommand) (domain.PickupOrder, domain.DispenseRecord, error) {
	var completed domain.PickupOrder
	var record domain.DispenseRecord
	err := s.store.Update(func(tx *store.Transaction) error {
		order, err := tx.GetOrder(command.OrderID)
		if err != nil {
			return err
		}
		domain.ValidateDispense(command, order)
		completed, err = domain.Transition(order, domain.StateCompleted, command.CompletedAt)
		if err != nil {
			return err
		}
		record = domain.DispenseRecord{
			ID:            command.RecordID,
			PickupOrderID: command.OrderID,
			Operator:      normalizeOperator(command.Operator),
			CounterCode:   command.CounterCode,
			CompletedAt:   command.CompletedAt,
			Note:          command.Note,
			Checks:        append([]domain.VerificationCheck(nil), command.Checks...),
			ItemCount:     len(command.Checks),
		}
		entry, entryErr := audit.NewEntry("audit-dispense-"+command.RecordID, audit.ActionCompleted, order.ID, command.Operator, audit.DescribeDispense(completed, record), command.CompletedAt)
		if entryErr != nil {
			return entryErr
		}
		if err := tx.SaveOrder(completed); err != nil {
			return err
		}
		if err := tx.CreateDispense(record); err != nil {
			return err
		}
		return tx.CreateAudit(entry)
	})
	return completed, record, err
}

func (s *Service) Cancel(orderID, reason, operator, auditID string, at domain.CallCommand) (domain.PickupOrder, error) {
	if reason == "" {
		return domain.PickupOrder{}, errors.New("取消原因不能为空")
	}
	var cancelled domain.PickupOrder
	err := s.store.Update(func(tx *store.Transaction) error {
		order, err := tx.GetOrder(orderID)
		if err != nil {
			return err
		}
		cancelled, err = domain.Transition(order, domain.StateCancelled, at.CalledAt)
		if err != nil {
			return err
		}
		entry, err := audit.NewEntry(auditID, audit.ActionCancelled, orderID, operator, audit.DescribeCancellation(order, reason), at.CalledAt)
		if err != nil {
			return err
		}
		if err := tx.SaveOrder(cancelled); err != nil {
			return err
		}
		return tx.CreateAudit(entry)
	})
	return cancelled, err
}
