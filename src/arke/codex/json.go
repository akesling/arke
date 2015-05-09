package codex

import (
	"encoding/json"
)

type JSONCodex struct{}

func (c JSONCodex) MIME() string {
	return "application/json"
}

func (c JSONCodex) Encode(go_type interface{}) (encoded []byte, err error) {
	return json.Marshal(go_type)
}

func (c JSONCodex) Decode(encoded []byte) (go_type interface{}, err error) {
	// TODO(akesling): Implement Decode()
	return encoded, nil
}

func (c JSONCodex) Transcode(encoding Codex, source_encoded []byte) (target_encoded []byte, err error) {
	// Pass through identity transcodings
	if _, ok := encoding.(JSONCodex); ok {
		return source_encoded, nil
	}

	decoded, err := encoding.Decode(source_encoded)
	if err != nil {
		return nil, err
	}

	return c.Encode(decoded)
}
