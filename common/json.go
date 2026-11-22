package common

import (
	"bytes"
	"errors"
	"io"

	"github.com/MoeForge/moego/json"
	"github.com/gin-gonic/gin/binding"
)

type RawMessage = json.RawMessage

func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// UnmarshalNoCopy borrows strings and raw JSON from data. Keep data and borrowed
// raw JSON immutable while decoded values are in use.
func UnmarshalNoCopy(data []byte, v any) error {
	return json.UnmarshalNoCopy(data, v)
}

func UnmarshalJsonStr(data string, v any) error {
	return Unmarshal([]byte(data), v)
}

func DecodeJson(reader io.Reader, v any) error {
	return json.NewDecoder(reader).Decode(v)
}

// DecodeJsonBody reads one JSON value and requires EOF after trailing whitespace.
// Seekable readers let the decoder process large values through a bounded window.
func DecodeJsonBody(reader io.Reader, v any) error {
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(v); err != nil {
		return err
	}
	var extra struct{}
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return err
		}
		return errors.New("multiple JSON values in request body")
	}
	return nil
}

// DecodeJsonWithValidation decodes JSON and applies Gin's configured binding-tag
// validator, including binding:"required" and any registered custom validators.
func DecodeJsonWithValidation(reader io.Reader, v any) error {
	if err := DecodeJson(reader, v); err != nil {
		return err
	}
	if binding.Validator == nil {
		return nil
	}
	return binding.Validator.ValidateStruct(v)
}

func Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func IndentJson(data []byte) ([]byte, error) {
	var buffer bytes.Buffer
	if err := json.Indent(&buffer, data, "", "  "); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func GetJsonType(data RawMessage) string {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return "unknown"
	}
	firstChar := trimmed[0]
	switch firstChar {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 't', 'f':
		return "boolean"
	case 'n':
		return "null"
	default:
		return "number"
	}
}

// JsonRawMessageToString returns JSON strings as their decoded value and other JSON values as raw text.
func JsonRawMessageToString(data RawMessage) string {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return ""
	}
	if trimmed[0] != '"' {
		return string(trimmed)
	}
	var value string
	if err := Unmarshal(trimmed, &value); err != nil {
		return string(trimmed)
	}
	return value
}
