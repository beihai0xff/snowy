package mysql

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// jsonMap 用于 map[string]any 与 MySQL JSON 列之间的序列化/反序列化。
// 实现 driver.Valuer 和 sql.Scanner 接口。
type jsonMap map[string]any

func newJSONMap(v map[string]any) jsonMap {
	if v == nil {
		return nil
	}

	return jsonMap(v)
}

// Value 实现 driver.Valuer — 序列化为 JSON []byte。
func (m jsonMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte("null"), nil
	}

	return json.Marshal(m)
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

// jsonValue 用于任意 JSON 值与 MySQL JSON 列之间的序列化/反序列化。
type jsonValue struct {
	Data any
}

func newJSONValue(v any) jsonValue {
	return jsonValue{Data: v}
}

// Value 实现 driver.Valuer — 序列化任意 JSON 值。
func (j jsonValue) Value() (driver.Value, error) {
	if j.Data == nil {
		return []byte("null"), nil
	}

	return json.Marshal(j.Data)
}

// Scan 实现 sql.Scanner — 从 JSON []byte 反序列化为任意 Go 值。
func (j *jsonValue) Scan(src any) error {
	if src == nil {
		j.Data = nil

		return nil
	}

	var b []byte

	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("jsonValue: unsupported type %T", src)
	}

	var decoded any
	if err := json.Unmarshal(b, &decoded); err != nil {
		return err
	}

	j.Data = decoded

	return nil
}
