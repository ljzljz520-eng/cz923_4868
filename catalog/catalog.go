package catalog

import (
	"errors"
	"sort"
	"strings"
)

type Medicine struct {
	Code             string   `json:"code"`
	Name             string   `json:"name"`
	Specification    string   `json:"specification"`
	Unit             string   `json:"unit"`
	Location         string   `json:"location"`
	MaximumQuantity  int      `json:"maximumQuantity"`
	Controlled       bool     `json:"controlled"`
	InteractionCodes []string `json:"interactionCodes"`
}

type Catalog struct {
	medicines map[string]Medicine
}

func New(medicines []Medicine) (*Catalog, error) {
	result := &Catalog{medicines: make(map[string]Medicine)}
	for _, medicine := range medicines {
		if err := ValidateMedicine(medicine); err != nil {
			return nil, err
		}
		medicine.Code = strings.ToUpper(strings.TrimSpace(medicine.Code))
		if _, exists := result.medicines[medicine.Code]; exists {
			return nil, errors.New("药品代码重复")
		}
		result.medicines[medicine.Code] = cloneMedicine(medicine)
	}
	return result, nil
}

func Default() *Catalog {
	result, _ := New([]Medicine{
		{Code: "MED001", Name: "阿莫西林胶囊", Specification: "0.25g*24粒", Unit: "盒", Location: "A-01", MaximumQuantity: 6, InteractionCodes: []string{"MED004"}},
		{Code: "MED002", Name: "布洛芬缓释胶囊", Specification: "0.3g*20粒", Unit: "盒", Location: "A-03", MaximumQuantity: 4},
		{Code: "MED003", Name: "硝苯地平控释片", Specification: "30mg*14片", Unit: "盒", Location: "B-02", MaximumQuantity: 8},
		{Code: "MED004", Name: "华法林钠片", Specification: "2.5mg*60片", Unit: "瓶", Location: "C-01", MaximumQuantity: 2, Controlled: true, InteractionCodes: []string{"MED001"}},
		{Code: "MED005", Name: "二甲双胍片", Specification: "0.5g*60片", Unit: "瓶", Location: "B-05", MaximumQuantity: 6},
	})
	return result
}

func ValidateMedicine(medicine Medicine) error {
	if strings.TrimSpace(medicine.Code) == "" {
		return errors.New("药品代码不能为空")
	}
	if strings.TrimSpace(medicine.Name) == "" {
		return errors.New("药品名称不能为空")
	}
	if strings.TrimSpace(medicine.Specification) == "" {
		return errors.New("药品规格不能为空")
	}
	if strings.TrimSpace(medicine.Unit) == "" {
		return errors.New("药品单位不能为空")
	}
	if medicine.MaximumQuantity < 1 {
		return errors.New("单次最大数量必须大于零")
	}
	return nil
}

func (c *Catalog) Get(code string) (Medicine, bool) {
	medicine, ok := c.medicines[strings.ToUpper(strings.TrimSpace(code))]
	return cloneMedicine(medicine), ok
}

func (c *Catalog) List() []Medicine {
	result := make([]Medicine, 0, len(c.medicines))
	for _, medicine := range c.medicines {
		result = append(result, cloneMedicine(medicine))
	}
	sort.SliceStable(result, func(left, right int) bool {
		return result[left].Code < result[right].Code
	})
	return result
}

func (c *Catalog) Search(value string) []Medicine {
	query := strings.ToLower(strings.TrimSpace(value))
	result := make([]Medicine, 0)
	for _, medicine := range c.List() {
		if query == "" || strings.Contains(strings.ToLower(medicine.Code), query) || strings.Contains(strings.ToLower(medicine.Name), query) {
			result = append(result, medicine)
		}
	}
	return result
}

func cloneMedicine(medicine Medicine) Medicine {
	medicine.InteractionCodes = append([]string(nil), medicine.InteractionCodes...)
	return medicine
}
