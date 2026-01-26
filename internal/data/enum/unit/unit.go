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
	Grams       = Unit{value: "g"}
	Millilitres = Unit{value: "mL"}
	Teaspoons   = Unit{value: "tsp"}
	Tablespoons = Unit{value: "tbsp"}
	Pieces      = Unit{value: ""}
	Pinch       = Unit{value: "a pinch"}
	ToTaste     = Unit{value: "to taste"}
)

func NewUnit(kind string) Unit {
	return map[string]Unit{
		Grams.value:       Grams,
		Millilitres.value: Millilitres,
		Teaspoons.value:   Teaspoons,
		Tablespoons.value: Tablespoons,
		Pieces.value:      Pieces,
		Pinch.value:       Pinch,
		ToTaste.value:     ToTaste,
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
