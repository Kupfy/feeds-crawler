package dto

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type JsonB map[string]interface{}

func (d *JsonB) Scan(val interface{}) error {
	switch v := val.(type) {
	case []byte:
		err := json.Unmarshal(v, &d)
		if err != nil {
			return err
		}
		return nil
	case string:
		err := json.Unmarshal([]byte(v), &d)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported type: %v", v)
	}
}

func (d JsonB) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func (d JsonB) String() string {
	b, _ := json.Marshal(d)
	return string(b)
}

func (d *JsonB) AddKeyValue(key string, value interface{}) {
	if *d == nil {
		*d = make(JsonB)
	}
	(*d)[key] = value
}

func (d *JsonB) ClearKey(key string) {
	delete(*d, key)
}
