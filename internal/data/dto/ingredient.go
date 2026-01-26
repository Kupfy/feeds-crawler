package dto

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/Kupfy/feeds-crawler/internal/data/enum/unit"
)

type Ingredient struct {
	Name      string
	Unit      unit.Unit
	Component *string
}

func (i *Ingredient) Scan(val interface{}) error {
	switch v := val.(type) {
	case []byte:
		*i = stringToIngredient(string(v))
		return nil
	case string:
		*i = stringToIngredient(v)
		return nil
	default:
		return fmt.Errorf("unsupported type: %v", v)
	}
}

func stringToIngredient(str string) Ingredient {
	split := strings.Split(str, ":")
	var u unit.Unit
	_ = u.UnmarshalText([]byte(split[2]))
	ingredient := Ingredient{
		Name: split[1],
		Unit: u,
	}
	if split[0] != "Dish" {
		ingredient.Component = &split[0]
	}
	return ingredient
}

func (i Ingredient) Value() (driver.Value, error) {
	component := "Dish"
	if i.Component != nil {
		component = *i.Component
	}

	return fmt.Sprintf("%s:%s:%s", component, i.Name, i.Unit), nil
}
