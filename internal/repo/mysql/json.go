package mysql

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// jsonMap 用于 map[string]any 与 MySQL JSON 列之间的序列化/反序列化。
// 实现 driver.Valuer 和 sql.Scanner 接口。
type jsonMap map[string]any

func newJSONMap(v map[string]any) *jsonMap {
	if v == nil {
		return nil
	}

	m := jsonMap(v)

	return &m
}

// Value 实现 driver.Valuer — 序列化为 JSON []byte。
func (m *jsonMap) Value() (driver.Value, error) {
	if m == nil || *m == nil {
		return []byte("null"), nil
	}

	return json.Marshal(*m)
}

// Scan 实现 sql.Scanner — 从 JSON []byte 反序列化。
func (m *jsonMap) Scan(src any) error {
	if src == nil {
		*m = nil

		return nil
	}

	var b []byte

	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("jsonMap: unsupported type %T", src)
	}

	return json.Unmarshal(b, m)
}

// jsonValueOf serializes any value to JSON []byte for storage in a JSON column.
func jsonValueOf(v any) driver.Value {
	if v == nil {
		return nil
	}

	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}

	return b
}
