package audit

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"pharmacycounter/domain"
)

const (
	ActionRegistered = "order.registered"
	ActionCalled     = "order.called"
	ActionRecalled   = "order.recalled"
	ActionCompleted  = "order.completed"
	ActionCancelled  = "order.cancelled"
	ActionWarning    = "warning.resolved"
)

var validActions = map[string]bool{
	ActionRegistered: true,
	ActionCalled:     true,
	ActionRecalled:   true,
	ActionCompleted:  true,
	ActionCancelled:  true,
	ActionWarning:    true,
}

func NewEntry(id, action, subjectID, operator, detail string, at time.Time) (domain.AuditEntry, error) {
	entry := domain.AuditEntry{
		ID:        strings.TrimSpace(id),
		Action:    strings.TrimSpace(action),
		SubjectID: strings.TrimSpace(subjectID),
		Operator:  strings.TrimSpace(operator),
		Detail:    strings.TrimSpace(detail),
		CreatedAt: at,
	}
	if err := ValidateEntry(entry); err != nil {
		return domain.AuditEntry{}, err
	}
	return entry, nil
}

func ValidateEntry(entry domain.AuditEntry) error {
	if entry.ID == "" {
		return errors.New("审计标识不能为空")
	}
	if !validActions[entry.Action] {
		return errors.New("审计动作无效")
	}
	if entry.SubjectID == "" {
		return errors.New("审计对象不能为空")
	}
	if entry.Operator == "" {
		return errors.New("操作员不能为空")
	}
	if entry.CreatedAt.IsZero() {
		return errors.New("审计时间不能为空")
	}
	return nil
}

func DescribeRegistration(order domain.PickupOrder) string {
	return fmt.Sprintf("登记取药单 %s，患者 %s，共 %d 项 %d 件药品", order.TicketNumber, order.PatientName, len(order.Items), domain.TotalQuantity(order.Items))
}

func DescribeCall(order domain.PickupOrder, record domain.CallRecord) string {
	return fmt.Sprintf("叫号 %s 到窗口 %s，序号 %d", order.TicketNumber, record.CounterCode, record.Sequence)
}

func DescribeDispense(order domain.PickupOrder, record domain.DispenseRecord) string {
	return fmt.Sprintf("完成取药单 %s，核验 %d 项，备注 %s", order.TicketNumber, record.ItemCount, record.Note)
}

func DescribeCancellation(order domain.PickupOrder, reason string) string {
	return fmt.Sprintf("取消取药单 %s，原因 %s", order.TicketNumber, strings.TrimSpace(reason))
}

func Timeline(entries []domain.AuditEntry) []domain.AuditEntry {
	result := append([]domain.AuditEntry(nil), entries...)
	sort.SliceStable(result, func(left, right int) bool {
		if !result[left].CreatedAt.Equal(result[right].CreatedAt) {
			return result[left].CreatedAt.Before(result[right].CreatedAt)
		}
		return result[left].ID < result[right].ID
	})
	return result
}

func ValidateTimeline(entries []domain.AuditEntry) error {
	ordered := Timeline(entries)
	seen := make(map[string]bool)
	states := make(map[string]domain.OrderState)
	for _, entry := range ordered {
		if err := ValidateEntry(entry); err != nil {
			return err
		}
		if seen[entry.ID] {
			return errors.New("审计标识重复")
		}
		seen[entry.ID] = true
		state := states[entry.SubjectID]
		switch entry.Action {
		case ActionRegistered:
			if state != "" {
				return errors.New("同一取药单重复登记")
			}
			states[entry.SubjectID] = domain.StateWaiting
		case ActionCalled:
			if state != domain.StateWaiting {
				return errors.New("叫号前状态无效")
			}
			states[entry.SubjectID] = domain.StateCalled
		case ActionRecalled:
			if state != domain.StateCalled {
				return errors.New("重叫前状态无效")
			}
		case ActionCompleted:
			if state != domain.StateCalled {
				return errors.New("发药前状态无效")
			}
			states[entry.SubjectID] = domain.StateCompleted
		case ActionCancelled:
			if state != domain.StateWaiting && state != domain.StateCalled {
				return errors.New("取消前状态无效")
			}
			states[entry.SubjectID] = domain.StateCancelled
		}
	}
	return nil
}

func ActionsByOperator(entries []domain.AuditEntry) map[string]int {
	result := make(map[string]int)
	for _, entry := range entries {
		result[entry.Operator]++
	}
	return result
}
