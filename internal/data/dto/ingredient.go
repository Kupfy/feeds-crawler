package dto

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/Kupfy/feeds-crawler/internal/data/enum/unit"
)

type IngredientsItem struct {
	Name        string
	Quantity    float64
	Unit        unit.Unit
	Component   *string
	Instruction *string
}

func (i *IngredientsItem) Scan(val interface{}) error {
	switch v := val.(type) {
	case []byte:
		*i = dbStringToIngredient(string(v))
		return nil
	case string:
		*i = dbStringToIngredient(v)
		return nil
	default:
		return fmt.Errorf("unsupported type: %v", v)
	}
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
	component := "Dish"
	if i.Component != nil {
		component = *i.Component
	}

	return fmt.Sprintf("%s:%s:%s", component, i.Name, i.Unit), nil
}

func (i IngredientsItem) String() string {

	return fmt.Sprintf("%s %s of %s", i.Quantity, i.Unit, i.Name)
}
