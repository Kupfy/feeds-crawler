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
	Millilitre = Unit{value: "mL", rank: 2}
	Teaspoon   = Unit{value: "tsp", rank: 3}
	Tablespoon = Unit{value: "tbsp", rank: 4}
	Cups       = Unit{value: "cup", rank: 5}
	Cloves     = Unit{value: "clove", rank: 6}
	Pinch      = Unit{value: "pinch", rank: 7}
	Bunch      = Unit{value: "bunch", rank: 8}
	ToTaste    = Unit{value: "to taste", rank: 9}
	Piece      = Unit{value: "", rank: 10}
)

func NewUnit(kind string) Unit {
	tokens := tokenize(kind)

	var u Unit
	for i, token := range tokens {
		u = map[string]Unit{
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
			"cup":         Cups,
			"cups":        Cups,
			"clove":       Cloves,
			"cloves":      Cloves,
			"a clove":     Cloves,
			"pinch":       Pinch,
			"a pinch":     Pinch,
			"pinches":     Pinch,
			"bunch":       Bunch,
			"a bunch":     Bunch,
			"bunches":     Bunch,
			"to taste":    ToTaste,
			"":            Piece,
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
