package model

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/Im-oldxu/new-api/common"
)

// JSONValue 基于 json.RawMessage 实现，支持从数据库的 []byte 和 string 两种类型读取
type JSONValue json.RawMessage

// Value 实现 driver.Valuer 接口，用于数据库写入
// 必须返回 string 而非 []byte:PG simple protocol 下 []byte 按 bytea 编码,
// 写 json 列会触发 SQLSTATE 22P02。
func (j JSONValue) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return string(j), nil
}

// Scan 实现 sql.Scanner 接口，兼容不同驱动返回的类型
func (j *JSONValue) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*j = nil
		return nil
	case []byte:
		// 拷贝底层字节，避免保留底层缓冲区
		b := make([]byte, len(v))
		copy(b, v)
		*j = JSONValue(b)
		return nil
	case string:
		*j = JSONValue([]byte(v))
		return nil
	default:
		// 其他类型尝试序列化为 JSON
		b, err := common.Marshal(v)
		if err != nil {
			return err
		}
		*j = JSONValue(b)
		return nil
	}
}

// MarshalJSON 确保在对外编码时与 json.RawMessage 行为一致
func (j JSONValue) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON 确保在对外解码时与 json.RawMessage 行为一致
func (j *JSONValue) UnmarshalJSON(data []byte) error {
	if data == nil {
		*j = nil
		return nil
	}
	b := make([]byte, len(data))
	copy(b, data)
	*j = JSONValue(b)
	return nil
}
