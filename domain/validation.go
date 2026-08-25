package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound          = errors.New("记录不存在")
	ErrConflict          = errors.New("记录版本冲突")
	ErrInvalidTransition = errors.New("不允许的状态变更")
	ErrDuplicate         = errors.New("记录已存在")
	ErrValidation        = errors.New("业务校验失败")
	ErrUnresolvedWarning = errors.New("存在未处理的安全警告")
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

func Required(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return ValidationError{Field: field, Message: "不能为空"}
	}
	return nil
}

func ValidatePriority(priority Priority) error {
	switch priority {
	case PriorityRoutine, PrioritySenior, PriorityUrgent:
		return nil
	default:
		return ValidationError{Field: "priority", Message: "优先级无效"}
	}
}

func ValidateState(state OrderState) error {
	switch state {
	case StateWaiting, StateCalled, StateCompleted, StateCancelled:
		return nil
	default:
		return ValidationError{Field: "state", Message: "状态无效"}
	}
}

func ValidateCreateOrder(command CreateOrderCommand) error {
	checks := []struct {
		field string
		value string
	}{
		{"id", command.ID},
		{"ticketNumber", command.TicketNumber},
		{"patientName", command.PatientName},
		{"patientCode", command.PatientCode},
		{"prescriptionId", command.PrescriptionID},
	}
	for _, check := range checks {
		if err := Required(check.field, check.value); err != nil {
			return err
		}
	}
	if err := ValidatePriority(command.Priority); err != nil {
		return err
	}
	if command.CreatedAt.IsZero() {
		return ValidationError{Field: "createdAt", Message: "必须提供业务时间"}
	}
	if len(command.Items) == 0 {
		return ValidationError{Field: "items", Message: "至少需要一项药品"}
	}
	seen := make(map[string]bool)
	for index, item := range command.Items {
		if err := ValidatePrescriptionItem(item); err != nil {
			return fmt.Errorf("items[%d]: %w", index, err)
		}
		if seen[item.MedicineCode] {
			return ValidationError{Field: "items", Message: "同一药品不能重复"}
		}
		seen[item.MedicineCode] = true
	}
	return nil
}

func ValidatePrescriptionItem(item PrescriptionItem) error {
	if err := Required("medicineCode", item.MedicineCode); err != nil {
		return err
	}
	if err := Required("name", item.Name); err != nil {
		return err
	}
	if err := Required("specification", item.Specification); err != nil {
		return err
	}
	if err := Required("unit", item.Unit); err != nil {
		return err
	}
	if err := Required("lot", item.Lot); err != nil {
		return err
	}
	if item.Quantity <= 0 {
		return ValidationError{Field: "quantity", Message: "必须大于零"}
	}
	return nil
}

func ValidateCall(command CallCommand) error {
	if err := Required("orderId", command.OrderID); err != nil {
		return err
	}
	if err := Required("counterCode", command.CounterCode); err != nil {
		return err
	}
	if err := Required("operator", command.Operator); err != nil {
		return err
	}
	if command.CalledAt.IsZero() {
		return ValidationError{Field: "calledAt", Message: "必须提供业务时间"}
	}
	return nil
}

func ValidateDispense(command DispenseCommand, order PickupOrder) error {
	if err := Required("orderId", command.OrderID); err != nil {
		return err
	}
	if err := Required("recordId", command.RecordID); err != nil {
		return err
	}
	if err := Required("counterCode", command.CounterCode); err != nil {
		return err
	}
	if err := Required("operator", command.Operator); err != nil {
		return err
	}
	if command.CompletedAt.IsZero() {
		return ValidationError{Field: "completedAt", Message: "必须提供业务时间"}
	}
	if order.State != StateCalled {
		return ErrInvalidTransition
	}
	if len(command.Checks) != len(order.Items) {
		return ValidationError{Field: "checks", Message: "核验项数量与处方不一致"}
	}
	checks := make(map[string]VerificationCheck)
	for _, check := range command.Checks {
		if check.MedicineCode == "" {
			return ValidationError{Field: "medicineCode", Message: "不能为空"}
		}
		if !check.Confirmed {
			return ValidationError{Field: "confirmed", Message: "所有药品必须确认"}
		}
		checks[check.MedicineCode] = check
	}
	for _, item := range order.Items {
		check, ok := checks[item.MedicineCode]
		if !ok {
			return ValidationError{Field: "checks", Message: "缺少药品核验"}
		}
		if check.Lot != item.Lot {
			return ValidationError{Field: "lot", Message: "批号与处方不一致"}
		}
		if check.Quantity != item.Quantity {
			return ValidationError{Field: "quantity", Message: "数量与处方不一致"}
		}
	}
	for _, warning := range order.Warnings {
		if warning.Severity == "high" && !warning.Resolved {
			return ErrUnresolvedWarning
		}
	}
	return nil
}

func CanTransition(from, to OrderState) bool {
	switch from {
	case StateWaiting:
		return to == StateCalled || to == StateCancelled
	case StateCalled:
		return to == StateCompleted || to == StateWaiting || to == StateCancelled
	case StateCompleted, StateCancelled:
		return false
	default:
		return false
	}
}

func Transition(order PickupOrder, to OrderState, at time.Time) (PickupOrder, error) {
	if !CanTransition(order.State, to) {
		return order, ErrInvalidTransition
	}
	if at.IsZero() || at.Before(order.CreatedAt) {
		return order, ValidationError{Field: "updatedAt", Message: "业务时间无效"}
	}
	order.State = to
	order.UpdatedAt = at
	order.Version++
	return order, nil
}

func ResolveWarning(order PickupOrder, code string) (PickupOrder, error) {
	found := false
	for index := range order.Warnings {
		if order.Warnings[index].Code == code {
			order.Warnings[index].Resolved = true
			found = true
		}
	}
	if !found {
		return order, ErrNotFound
	}
	order.Version++
	return order, nil
}

func NormalizeOrder(order PickupOrder) PickupOrder {
	order.ID = strings.TrimSpace(order.ID)
	order.TicketNumber = strings.ToUpper(strings.TrimSpace(order.TicketNumber))
	order.PatientName = strings.TrimSpace(order.PatientName)
	order.PatientCode = strings.ToUpper(strings.TrimSpace(order.PatientCode))
	order.PrescriptionID = strings.ToUpper(strings.TrimSpace(order.PrescriptionID))
	for index := range order.Items {
		order.Items[index].MedicineCode = strings.ToUpper(strings.TrimSpace(order.Items[index].MedicineCode))
		order.Items[index].Name = strings.TrimSpace(order.Items[index].Name)
		order.Items[index].Lot = strings.ToUpper(strings.TrimSpace(order.Items[index].Lot))
		order.Items[index].Unit = strings.TrimSpace(order.Items[index].Unit)
	}
	return order
}
