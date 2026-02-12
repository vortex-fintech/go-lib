package schemaregistry

import (
	"errors"

	"github.com/twmb/franz-go/pkg/sr"
)

var (
	ErrDataTooShort          = errors.New("schema registry payload is too short")
	ErrInvalidMagicByte      = errors.New("schema registry payload has invalid magic byte")
	ErrInvalidMessageIndexes = errors.New("schema registry payload has invalid protobuf message indexes")
)

type ProtoDeserializer struct{}

func NewProtoDeserializer(_ *Client) *ProtoDeserializer {
	return &ProtoDeserializer{}
}

// Deserialize parses Confluent wire format and returns protobuf payload + schema ID.
func (d *ProtoDeserializer) Deserialize(data []byte) ([]byte, int, error) {
	payload, schemaID, _, err := d.DeserializeWithIndexes(data)
	if err != nil {
		return nil, 0, err
	}
	return payload, schemaID, nil
}

// DeserializeWithIndexes parses payload, schema ID and protobuf message-index path.
func (d *ProtoDeserializer) DeserializeWithIndexes(data []byte) ([]byte, int, []int, error) {
	if len(data) < 6 {
		return nil, 0, nil, ErrDataTooShort
	}

	if data[0] != 0 {
		return nil, 0, nil, ErrInvalidMagicByte
	}

	schemaID, body, err := confluentHeader.DecodeID(data)
	if err != nil {
		if errors.Is(err, sr.ErrBadHeader) {
			return nil, 0, nil, ErrInvalidMagicByte
		}
		return nil, 0, nil, ErrDataTooShort
	}

	indexes, payload, err := confluentHeader.DecodeIndex(body, 0)
	if err != nil {
		return nil, 0, nil, ErrInvalidMessageIndexes
	}

	return payload, schemaID, indexes, nil
}
