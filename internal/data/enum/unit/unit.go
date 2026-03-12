package unit

import (
	"database/sql/driver"
	"encoding/json"
	"strings"
)

type Unit struct {
	value     string
	qualifier string
	rank      int
}

func (u Unit) String() string {
	return u.value
}

var (
	Gram       = Unit{value: "g", rank: 1}
	Kilogram   = Unit{value: "Kg", rank: 2}
	Ounce      = Unit{value: "oz", rank: 3}
	Pound      = Unit{value: "lb", rank: 4}
	Millilitre = Unit{value: "mL", rank: 5}
	Litre      = Unit{value: "L", rank: 6}
	FluidOunce = Unit{value: "fl oz", rank: 7}
	Teaspoon   = Unit{value: "tsp", rank: 8}
	Tablespoon = Unit{value: "tbsp", rank: 9}
	Cups       = Unit{value: "cup", rank: 10}
	Cloves     = Unit{value: "clove", rank: 11}
	Pinch      = Unit{value: "pinch", rank: 12}
	Bunch      = Unit{value: "bunch", rank: 13}
	ToTaste    = Unit{value: "to taste", rank: 14}
	Piece      = Unit{value: "", rank: 15}
)

func NewUnit(kind string) Unit {
	tokens := tokenize(strings.ToLower(kind))

	var u Unit
	for i, token := range tokens {
		u = map[string]Unit{
			"g":            Gram,
			"kg":           Kilogram,
			"gram":         Gram,
			"grams":        Gram,
			"kilogram":     Kilogram,
			"kilograms":    Kilogram,
			"oz":           Ounce,
			"ounce":        Ounce,
			"ounces":       Ounce,
			"lb":           Pound,
			"pound":        Pound,
			"pounds":       Pound,
			"ml":           Millilitre,
			"mL":           Millilitre,
			"milliliter":   Millilitre,
			"millilitre":   Millilitre,
			"millilitres":  Millilitre,
			"milliliters":  Millilitre,
			"l":            Litre,
			"litre":        Litre,
			"litres":       Litre,
			"liter":        Litre,
			"liters":       Litre,
			"fl oz":        FluidOunce,
			"fl ounce":     FluidOunce,
			"fl ounces":    FluidOunce,
			"fluid ounce":  FluidOunce,
			"fluid ounces": FluidOunce,
			"tsp":          Teaspoon,
			"teaspoon":     Teaspoon,
			"teaspoons":    Teaspoon,
			"tbsp":         Tablespoon,
			"tablespoon":   Tablespoon,
			"tablespoons":  Tablespoon,
			"cup":          Cups,
			"cups":         Cups,
			"clove":        Cloves,
			"cloves":       Cloves,
			"a clove":      Cloves,
			"pinch":        Pinch,
			"a pinch":      Pinch,
			"pinches":      Pinch,
			"bunch":        Bunch,
			"a bunch":      Bunch,
			"bunches":      Bunch,
			"to taste":     ToTaste,
			"":             Piece,
		}[token]

		if u.value != "" {
			qualifier := append(tokens[:i], tokens[i+1:]...)
			u.qualifier = strings.Join(qualifier, " ")
		}
	}
	return u
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

func (u Unit) Rank() int {
	return u.rank
}

func tokenize(s string) []string {
	s = strings.ReplaceAll(s, ",", " , ")
	s = strings.ReplaceAll(s, "(", " ( ")
	s = strings.ReplaceAll(s, ")", " ) ")
	return strings.Fields(s)
}
