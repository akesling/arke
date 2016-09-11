package codex

import (
	"encoding/json"
)

type JSONCodex struct{}

func (c JSONCodex) MIME() string {
	return "application/json"
}

func (c JSONCodex) Marshal(value interface{}) (data []byte, err error) {
	return json.Marshal(value)
}

func (c JSONCodex) Unmarshal(data []byte, value interface{}) (err error) {
	return json.Unmarshal(data, value)
}

func (c JSONCodex) Transmarshal(encoding Codex, input []byte) (result []byte, err error) {
	var temp interface{}

	if _, ok := encoding.(JSONCodex); ok {
		result = make([]byte, len(input))
		copy(input, result)
		return result, nil
	}

	err = encoding.Unmarshal(input, temp)
	if err != nil {
		return nil, err
	}

	return c.Marshal(temp)
}
