package codex

type Codex interface {
	Encode(go_type interface{}) (encoded []byte, err error)
	Decode(encoded []byte) (go_type interface{}, err error)
	Transcode(encoding Codex, source_encoded []byte) (target_encoded []byte, err error)
	MIME() string
}
