package codex

type Marshaller interface {
	Marshal(value interface{}) (data []byte, err error)
}

type Unmarshaller interface {
	Unmarshal(data []byte, value interface{}) (err error)
}

type Transmarshaller interface {
	Transmarshal(encoding Codex, input []byte) (result []byte, err error)
}

type Codex interface {
	MIME() string

	Marshaller
	Unmarshaller
	Transmarshaller
}
