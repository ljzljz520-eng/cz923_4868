package catalog

import (
	"fmt"
	"sort"

	"pharmacycounter/domain"
)

func (c *Catalog) ValidateItems(items []domain.PrescriptionItem) error {
	for _, item := range items {
		medicine, ok := c.Get(item.MedicineCode)
		if !ok {
			return fmt.Errorf("药品 %s 不在目录中", item.MedicineCode)
		}
		if item.Quantity > medicine.MaximumQuantity {
			return fmt.Errorf("药品 %s 超过单次最大数量", item.MedicineCode)
		}
		if item.Specification != medicine.Specification {
			return fmt.Errorf("药品 %s 规格不一致", item.MedicineCode)
		}
		if item.Unit != medicine.Unit {
			return fmt.Errorf("药品 %s 单位不一致", item.MedicineCode)
		}
	}
	return nil
}

func (c *Catalog) Warnings(items []domain.PrescriptionItem) []domain.SafetyWarning {
	codes := make(map[string]bool)
	for _, item := range items {
		codes[item.MedicineCode] = true
	}
	warnings := make([]domain.SafetyWarning, 0)
	seen := make(map[string]bool)
	for _, item := range items {
		medicine, ok := c.Get(item.MedicineCode)
		if !ok {
			continue
		}
		if medicine.Controlled {
			code := "CONTROLLED-" + medicine.Code
			warnings = append(warnings, domain.SafetyWarning{Code: code, Message: medicine.Name + " 为重点核验药品", Severity: "high"})
		}
		for _, interacting := range medicine.InteractionCodes {
			if !codes[interacting] {
				continue
			}
			left, right := medicine.Code, interacting
			if left > right {
				left, right = right, left
			}
			code := "INTERACTION-" + left + "-" + right
			if seen[code] {
				continue
			}
			seen[code] = true
			warnings = append(warnings, domain.SafetyWarning{Code: code, Message: "药品组合需要药师复核", Severity: "high"})
		}
	}
	sort.SliceStable(warnings, func(left, right int) bool {
		return warnings[left].Code < warnings[right].Code
	})
	return warnings
}

func FillCatalogDetails(c *Catalog, items []domain.PrescriptionItem) []domain.PrescriptionItem {
	result := append([]domain.PrescriptionItem(nil), items...)
	for index := range result {
		medicine, ok := c.Get(result[index].MedicineCode)
		if !ok {
			continue
		}
		if result[index].Name == "" {
			result[index].Name = medicine.Name
		}
		if result[index].Specification == "" {
			result[index].Specification = medicine.Specification
		}
		if result[index].Unit == "" {
			result[index].Unit = medicine.Unit
		}
		if result[index].Location == "" {
			result[index].Location = medicine.Location
		}
	}
	return result
}
