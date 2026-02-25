package dto

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/Kupfy/feeds-crawler/internal/data/enum/unit"
	"github.com/Kupfy/feeds-crawler/internal/util"
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
	if i.QuantityMax != nil {
		quantityRange = fmt.Sprintf("-%s", util.FractionPrettyPrint(i.QuantityMax))
	}
	altQuantity := ""
	if i.AltQuantity != nil {
		altQuantity = fmt.Sprintf(" (%s)", *i.AltQuantity)
	}
	suffix := ""
	if i.Instruction != nil {
		suffix = fmt.Sprintf(", %s", *i.Instruction)
	}
	return fmt.Sprintf("%s%s%s%s of %s%s", util.FractionPrettyPrint(i.Quantity), quantityRange, i.Unit, altQuantity, i.Name, suffix)
}
