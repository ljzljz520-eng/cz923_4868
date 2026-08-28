package service

import (
	"fmt"

	"pharmacycounter/audit"
	"pharmacycounter/domain"
	"pharmacycounter/queue"
	"pharmacycounter/store"
)

func (s *Service) CallNext(command domain.CallCommand) (domain.PickupOrder, domain.CallRecord, error) {
	if err := domain.ValidateCall(command); err != nil {
		return domain.PickupOrder{}, domain.CallRecord{}, err
	}
	var called domain.PickupOrder
	var record domain.CallRecord
	err := s.store.Update(func(tx *store.Transaction) error {
		orders, err := tx.ListOrders()
		if err != nil {
			return err
		}
		selected, err := s.policy.Next(orders)
		if err != nil {
			return err
		}
		if command.OrderID != "" && command.OrderID != selected.ID {
			selected, err = tx.GetOrder(command.OrderID)
			if err != nil {
				return err
			}
			if selected.State != domain.StateWaiting {
				return domain.ErrInvalidTransition
			}
		}
		counters, err := tx.EnabledCounters()
		if err != nil {
			return err
		}
		calls, err := tx.ListCalls()
		if err != nil {
			return err
		}
		loads := queue.BuildLoads(counters, orders, calls)
		counter, err := queue.ChooseCounter(counters, loads, command.CounterCode)
		if err != nil {
			return err
		}
		updated, err := domain.Transition(selected, domain.StateCalled, command.CalledAt)
		if err != nil {
			return err
		}
		sequence, err := tx.NextCallSequence()
		if err != nil {
			return err
		}
		record = domain.CallRecord{
			ID:            fmt.Sprintf("call-%06d", sequence),
			PickupOrderID: selected.ID,
			TicketNumber:  selected.TicketNumber,
			CounterCode:   counter.Code,
			Operator:      normalizeOperator(command.Operator),
			Sequence:      sequence,
			CalledAt:      command.CalledAt,
		}
		entry, err := audit.NewEntry("audit-"+record.ID, audit.ActionCalled, selected.ID, record.Operator, audit.DescribeCall(updated, record), command.CalledAt)
		if err != nil {
			return err
		}
		if err := tx.SaveOrder(updated); err != nil {
			return err
		}
		if err := tx.CreateCall(record); err != nil {
			return err
		}
		if err := tx.CreateAudit(entry); err != nil {
			return err
		}
		called = updated
		return nil
	})
	return called, record, err
}

func (s *Service) Recall(orderID, operator, auditID string, calledAt domain.CallCommand) (domain.CallRecord, error) {
	var recalled domain.CallRecord
	err := s.store.Update(func(tx *store.Transaction) error {
		order, err := tx.GetOrder(orderID)
		if err != nil {
			return err
		}
		if order.State != domain.StateCalled {
			return domain.ErrInvalidTransition
		}
		current, err := tx.GetActiveCall(orderID)
		if err != nil {
			return err
		}
		calls, err := tx.ListCalls()
		if err != nil {
			return err
		}
		if !queue.CanRecall(current, calls, s.policy.MaxRecall) {
			return domain.ErrConflict
		}
		sequence, err := tx.NextCallSequence()
		if err != nil {
			return err
		}
		recalled = current
		recalled.ID = fmt.Sprintf("call-%06d", sequence)
		recalled.Sequence = sequence
		recalled.Operator = operator
		recalled.CalledAt = calledAt.CalledAt
		recalled.Recalled = true
		entry, err := audit.NewEntry(auditID, audit.ActionRecalled, orderID, operator, audit.DescribeCall(order, recalled), calledAt.CalledAt)
		if err != nil {
			return err
		}
		if err := tx.CreateCall(recalled); err != nil {
			return err
		}
		return tx.CreateAudit(entry)
	})
	return recalled, err
}
