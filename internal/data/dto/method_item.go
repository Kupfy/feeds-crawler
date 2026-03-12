package dto

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Method []MethodItem

func (m *Method) Scan(val interface{}) error {
	var err error
	switch v := val.(type) {
	case []byte:
		err = json.Unmarshal(v, m)
	case string:
		err = json.Unmarshal([]byte(v), m)
	default:
		err = fmt.Errorf("unsupported type: %v", v)
	}
	return err
}

func (m Method) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type MethodItem struct {
	Content string  `json:"content"`
	Section *string `json:"section"`
}

func NewMethodItem(content string, section *string) MethodItem {
	return MethodItem{content, section}
}

func (m *MethodItem) Scan(val interface{}) error {
	var err error
	switch v := val.(type) {
	case []byte:
		err = json.Unmarshal(v, m)
	case string:
		err = json.Unmarshal([]byte(v), m)
	default:
		err = fmt.Errorf("unsupported type: %v", v)
	}
	return err
}

func (m MethodItem) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m MethodItem) String() string {
	return m.Content
}
