package dto

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Kupfy/feeds-crawler/internal/data/enum/unit"
)

type Ingredients []IngredientsItem

func (i *Ingredients) Scan(val interface{}) error {
	var err error
	switch v := val.(type) {
	case []byte:
		err = json.Unmarshal(v, &i)
	case string:
		err = json.Unmarshal([]byte(v), &i)
	default:
		err = fmt.Errorf("unsupported type: %v", v)
	}
	return err
}

type IngredientsItem struct {
	Name        string
	Quantity    float64
	Unit        unit.Unit
	Component   *string
	Instruction *string
}

func (i *IngredientsItem) Scan(val interface{}) error {
	var err error
	switch v := val.(type) {
	case []byte:
		err = json.Unmarshal(v, &i)
	case string:
		err = json.Unmarshal([]byte(v), &i)
	default:
		err = fmt.Errorf("unsupported type: %v", v)
	}
	return err
}

func dbStringToIngredient(str string) IngredientsItem {
	split := strings.Split(str, ":")
	var u unit.Unit
	_ = u.UnmarshalText([]byte(split[2]))
	ingredient := IngredientsItem{
		Name: split[1],
		Unit: u,
	}
	if split[0] != "Dish" {
		ingredient.Component = &split[0]
	}
	return ingredient
}

func (i IngredientsItem) Value() (driver.Value, error) {
	return json.Marshal(i)
}

func (i IngredientsItem) String() string {
	suffix := ""
	if i.Instruction != nil {
		suffix = fmt.Sprintf(", %s", *i.Instruction)
	}
	return fmt.Sprintf("%s%s of %s%s", floatToPrint(i.Quantity), i.Unit, i.Name, suffix)
}

func floatToPrint(float65 float64) string {
	wholePart := int(float65)
	decimalPart := float65 - float64(wholePart)
	replacements := map[float64]string{
		0.5:   "½",
		0.25:  "¼",
		0.75:  "¾",
		0.125: "⅛",
	}

	return strconv.Itoa(wholePart) + replacements[decimalPart]
}
