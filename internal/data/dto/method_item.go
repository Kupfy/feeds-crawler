package dto

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

type MethodItem struct {
	Content string
	Section *string
}

func (m *MethodItem) Scan(val interface{}) error {
	switch v := val.(type) {
	case []byte:
		*m = stringToMethod(string(v))
		return nil
	case string:
		*m = stringToMethod(v)
		return nil
	default:
		return fmt.Errorf("unsupported type: %v", v)
	}
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
	section := "Method"
	if m.Section != nil {
		section = *m.Section
	}

	return fmt.Sprintf("%s:%s", section, m.Content), nil
}

func (m MethodItem) String() string {
	return m.Content
}

func (m MethodItem) UnmarshalText(text []byte) error {
	return m.Scan(text)
}
