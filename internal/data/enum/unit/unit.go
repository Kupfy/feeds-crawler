package unit

import (
	"database/sql/driver"
	"encoding/json"
)

type Unit struct {
	value string
}

func (u Unit) String() string {
	return u.value
}

var (
	Gram       = Unit{value: "g"}
	Millilitre = Unit{value: "mL"}
	Teaspoon   = Unit{value: "tsp"}
	Tablespoon = Unit{value: "tbsp"}
	Piece      = Unit{value: ""}
	Pinch      = Unit{value: "pinch"}
	Bunch      = Unit{value: "bunch"}
	Clove      = Unit{value: "clove"}
	ToTaste    = Unit{value: "to taste"}
)

func NewUnit(kind string) Unit {
	return map[string]Unit{
		"g":           Gram,
		"gram":        Gram,
		"grams":       Gram,
		"ml":          Millilitre,
		"mL":          Millilitre,
		"millilitre":  Millilitre,
		"millilitres": Millilitre,
		"tsp":         Teaspoon,
		"teaspoon":    Teaspoon,
		"teaspoons":   Teaspoon,
		"tbsp":        Tablespoon,
		"tablespoon":  Tablespoon,
		"tablespoons": Tablespoon,
		"pinch":       Pinch,
		"a pinch":     Pinch,
		"pinches":     Pinch,
		"bunch":       Bunch,
		"a bunch":     Bunch,
		"bunches":     Bunch,
		"to taste":    ToTaste,
		"clove":       Clove,
		"a clove":     Clove,
		"cloves":      Clove,
		"":            Piece,
	}[kind]
}

func (u Unit) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

func (u *Unit) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*u = NewUnit(s)
	return nil
}

func (u *Unit) UnmarshalText(text []byte) error {
	*u = NewUnit(string(text))
	return nil
}

func (u *Unit) Scan(value interface{}) error {
	if value == nil {
		*u = Unit{}
		return nil
	}
	*u = NewUnit(value.(string))
	return nil
}

func (u Unit) Value() (driver.Value, error) {
	i := NewUnit(u.value)
	if i.value == "" {
		return nil, nil
	}
	return i.value, nil
}
