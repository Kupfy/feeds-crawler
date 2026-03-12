package dto

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Serving struct {
	Quantity int    `json:"quantity"`
	Course   string `json:"course"`
	Makes    string `json:"makes"`
}

func (s *Serving) Scan(val interface{}) error {
	var err error
	switch v := val.(type) {
	case []byte:
		err = json.Unmarshal(v, s)
	case string:
		err = json.Unmarshal([]byte(v), s)
	default:
		err = fmt.Errorf("unsupported type: %v", v)
	}
	return err
}

func (s Serving) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s Serving) String() string {
	switch {
	case s.Quantity > 0 && s.Course != "" && s.Makes != "":
		return fmt.Sprintf("Serves %d as a %s (%s)", s.Quantity, s.Course, s.Makes)
	case s.Quantity > 0 && s.Course != "":
		return fmt.Sprintf("Serves %d as a %s", s.Quantity, s.Course)
	case s.Quantity > 0 && s.Makes != "":
		return fmt.Sprintf("Serves %d (makes %s)", s.Quantity, s.Makes)
	case s.Makes != "":
		return fmt.Sprintf("Makes %s", s.Makes)
	default:
		return ""
	}
}
