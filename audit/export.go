package audit

import (
	"bytes"
	"encoding/csv"
	"errors"
	"strings"
	"time"

	"pharmacycounter/domain"
)

func ExportCSV(entries []domain.AuditEntry) ([]byte, error) {
	buffer := &bytes.Buffer{}
	buffer.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(buffer)
	if err := writer.Write([]string{"审计标识", "动作", "对象", "操作员", "详情", "时间"}); err != nil {
		return nil, err
	}
	for _, entry := range Timeline(entries) {
		if err := writer.Write([]string{entry.ID, entry.Action, entry.SubjectID, entry.Operator, entry.Detail, entry.CreatedAt.Format(time.RFC3339)}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func ImportCSV(data []byte) ([]domain.AuditEntry, error) {
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\xEF\xBB\xBF")))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, errors.New("审计文件为空")
	}
	if len(records[0]) != 6 || records[0][0] != "审计标识" {
		return nil, errors.New("审计表头无效")
	}
	entries := make([]domain.AuditEntry, 0, len(records)-1)
	for index, record := range records[1:] {
		if len(record) != 6 {
			return nil, errors.New("审计行列数无效")
		}
		createdAt, err := time.Parse(time.RFC3339, record[5])
		if err != nil {
			return nil, err
		}
		entry := domain.AuditEntry{ID: record[0], Action: record[1], SubjectID: record[2], Operator: record[3], Detail: record[4], CreatedAt: createdAt}
		if err := ValidateEntry(entry); err != nil {
			return nil, errors.New("第 " + string(rune(index+2)) + " 行无效")
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
