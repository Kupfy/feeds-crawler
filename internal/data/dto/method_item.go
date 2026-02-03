package dto

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

type Method []MethodItem

func (m *Method) Scan(val interface{}) error {
	var err error
	switch v := val.(type) {
	case []byte:
		err = json.Unmarshal(v, &m)
	case string:
		err = json.Unmarshal([]byte(v), &m)
	default:
		err = fmt.Errorf("unsupported type: %v", v)
	}
	return err
}

type MethodItem struct {
	Content string
	Section *string
}

func (m *MethodItem) Scan(val interface{}) error {
	var err error
	switch v := val.(type) {
	case []byte:
		err = json.Unmarshal(v, &m)
	case string:
		err = json.Unmarshal([]byte(v), &m)
	default:
		err = fmt.Errorf("unsupported type: %v", v)
	}
	return err
}

func stringToMethod(str string) MethodItem {
	split := strings.Split(str, ":")
	method := MethodItem{
		Content: split[1],
	}
	if split[0] != "Method" {
		method.Section = &split[0]
	}
	return method
}

func (m MethodItem) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m MethodItem) String() string {
	return m.Content
}

func (m MethodItem) UnmarshalText(text []byte) error {
	return m.Scan(text)
}
