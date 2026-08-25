package service

import (
	"errors"
	"fmt"
	"strings"

	"pharmacycounter/audit"
	"pharmacycounter/domain"
	"pharmacycounter/queue"
	"pharmacycounter/store"
)

type Service struct {
	store  *store.Store
	policy queue.Policy
}

func New(storage *store.Store, policy queue.Policy) (*Service, error) {
	if storage == nil {
		return nil, errors.New("存储不能为空")
	}
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	return &Service{store: storage, policy: policy}, nil
}

func (s *Service) SeedCounters(counters []domain.Counter) error {
	return s.store.Update(func(tx *store.Transaction) error {
		return tx.SeedCounters(counters)
	})
}

func (s *Service) Register(command domain.CreateOrderCommand, operator string) (domain.PickupOrder, error) {
	if err := domain.Required("operator", operator); err != nil {
		return domain.PickupOrder{}, err
	}
	if err := domain.ValidateCreateOrder(command); err != nil {
		return domain.PickupOrder{}, err
	}
	order := domain.PickupOrder{
		ID:             command.ID,
		TicketNumber:   command.TicketNumber,
		PatientName:    command.PatientName,
		PatientCode:    command.PatientCode,
		PrescriptionID: command.PrescriptionID,
		Priority:       command.Priority,
		State:          domain.StateWaiting,
		CreatedAt:      command.CreatedAt,
		UpdatedAt:      command.CreatedAt,
		Version:        1,
		Items:          append([]domain.PrescriptionItem(nil), command.Items...),
	}
	for index := range order.Items {
		order.Items[index].PickupOrderID = order.ID
		if order.Items[index].ID == "" {
			order.Items[index].ID = fmt.Sprintf("%s-item-%02d", order.ID, index+1)
		}
	}
	order = domain.NormalizeOrder(order)
	entry, err := audit.NewEntry("audit-register-"+order.ID, audit.ActionRegistered, order.ID, operator, audit.DescribeRegistration(order), command.CreatedAt)
	if err != nil {
		return domain.PickupOrder{}, err
	}
	err = s.store.Update(func(tx *store.Transaction) error {
		if err := tx.CreateOrder(order); err != nil {
			return err
		}
		return tx.CreateAudit(entry)
	})
	return order, err
}

func (s *Service) AddWarning(orderID string, warning domain.SafetyWarning, operator string) (domain.PickupOrder, error) {
	if warning.Code == "" || warning.Message == "" {
		return domain.PickupOrder{}, domain.ErrValidation
	}
	if warning.Severity != "low" && warning.Severity != "medium" && warning.Severity != "high" {
		return domain.PickupOrder{}, domain.ValidationError{Field: "severity", Message: "级别无效"}
	}
	var updated domain.PickupOrder
	err := s.store.Update(func(tx *store.Transaction) error {
		order, err := tx.GetOrder(orderID)
		if err != nil {
			return err
		}
		for _, existing := range order.Warnings {
			if existing.Code == warning.Code {
				return domain.ErrDuplicate
			}
		}
		order.Warnings = append(order.Warnings, warning)
		order.Version++
		if err := tx.SaveOrder(order); err != nil {
			return err
		}
		updated = order
		return nil
	})
	return updated, err
}

func (s *Service) ResolveWarning(orderID, code, operator, auditID string, atTime domain.CallCommand) (domain.PickupOrder, error) {
	var updated domain.PickupOrder
	err := s.store.Update(func(tx *store.Transaction) error {
		order, err := tx.GetOrder(orderID)
		if err != nil {
			return err
		}
		order, err = domain.ResolveWarning(order, code)
		if err != nil {
			return err
		}
		if err := tx.SaveOrder(order); err != nil {
			return err
		}
		entry, err := audit.NewEntry(auditID, audit.ActionWarning, orderID, operator, "处理安全警告 "+code, atTime.CalledAt)
		if err != nil {
			return err
		}
		if err := tx.CreateAudit(entry); err != nil {
			return err
		}
		updated = order
		return nil
	})
	return updated, err
}

func normalizeOperator(value string) string {
	return strings.TrimSpace(value)
}
