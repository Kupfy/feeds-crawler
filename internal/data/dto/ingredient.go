package dto

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/Kupfy/feeds-crawler/internal/data/enum/unit"
)

type Ingredients []IngredientsItem

func (i *Ingredients) Scan(val interface{}) error {
	var err error
	switch v := val.(type) {
	case []byte:
		err = json.Unmarshal(v, i)
	case string:
		err = json.Unmarshal([]byte(v), i)
	default:
		err = fmt.Errorf("unsupported type: %v", v)
	}
	return err
}

func (i Ingredients) Value() (driver.Value, error) {
	return json.Marshal(i)
}

type IngredientsItem struct {
	Name        string    `json:"name"`
	Quantity    *float64  `json:"quantity"`
	QuantityMax *float64  `json:"quantityMax"`
	Range       bool      `json:"range"`
	Unit        unit.Unit `json:"unit"`
	AltQuantity *string   `json:"altQuantity"`
	Component   *string   `json:"component"`
	Instruction *string   `json:"instruction"`
}

func (i *IngredientsItem) Scan(val interface{}) error {
	var err error
	switch v := val.(type) {
	case []byte:
		err = json.Unmarshal(v, i)
	case string:
		err = json.Unmarshal([]byte(v), i)
	default:
		err = fmt.Errorf("unsupported type: %v", v)
	}
	return err
}

func (i IngredientsItem) Value() (driver.Value, error) {
	return json.Marshal(i)
}

func (i IngredientsItem) String() string {
	quantityRange := ""
	if i.Quantity != nil && i.QuantityMax != nil && *i.Quantity != *i.QuantityMax {
		quantityRange = fmt.Sprintf("-%v", fractionPrettyPrint(i.QuantityMax))
	}
	altQuantity := ""
	if i.AltQuantity != nil {
		altQuantity = fmt.Sprintf(" (%v)", *i.AltQuantity)
	}
	if i.Quantity == nil {
		return i.Name
	}
	return fmt.Sprintf("%s%s%s%s %s", fractionPrettyPrint(i.Quantity), quantityRange, i.Unit, altQuantity, i.Name)
}

var fractionUnicodes = map[float64]rune{
	0.25:      '¼',
	0.5:       '½',
	0.75:      '¾',
	1.0 / 7.0: '⅐',
	1.0 / 9.0: '⅑',
	0.1:       '⅒',
	1.0 / 3.0: '⅓',
	2.0 / 3.0: '⅔',
	0.2:       '⅕',
	0.4:       '⅖',
	0.6:       '⅗',
	0.8:       '⅘',
	1.0 / 6.0: '⅙',
	5.0 / 6.0: '⅚',
	0.125:     '⅛',
	0.375:     '⅜',
	0.625:     '⅝',
	0.875:     '⅞',
}

func fractionPrettyPrint(num *float64) string {
	if num == nil {
		return ""
	}
	wholeNum := int(*num)
	fraction := *num - float64(wholeNum)
	if fraction == 0 {
		return fmt.Sprintf("%d", wholeNum)
	}
	return fmt.Sprintf("%d%v", wholeNum, fractionUnicodes[fraction])
}
