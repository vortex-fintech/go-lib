package schemaregistry

import (
	"errors"
	"reflect"
	"testing"
)

func TestProtoDeserializer_Deserialize(t *testing.T) {
	d := NewProtoDeserializer(nil)
	payload := []byte{10, 3, 'f', 'o', 'o'}
	wire := createWireFormat(payload, 101, []int{0})

	gotPayload, gotID, err := d.Deserialize(wire)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotID != 101 {
		t.Fatalf("expected schema ID 101, got %d", gotID)
	}
	if !reflect.DeepEqual(gotPayload, payload) {
		t.Fatalf("unexpected payload: got %v want %v", gotPayload, payload)
	}
}

func TestProtoDeserializer_DeserializeWithIndexes(t *testing.T) {
	d := NewProtoDeserializer(nil)
	payload := []byte{1, 2, 3}
	wire := createWireFormat(payload, 7, []int{2, 1})

	gotPayload, gotID, gotIndexes, err := d.DeserializeWithIndexes(wire)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotID != 7 {
		t.Fatalf("expected schema ID 7, got %d", gotID)
	}
	if !reflect.DeepEqual(gotIndexes, []int{2, 1}) {
		t.Fatalf("unexpected indexes: got %v", gotIndexes)
	}
	if !reflect.DeepEqual(gotPayload, payload) {
		t.Fatalf("unexpected payload: got %v want %v", gotPayload, payload)
	}
}

func TestProtoDeserializer_InvalidMagicByte(t *testing.T) {
	d := NewProtoDeserializer(nil)
	_, _, err := d.Deserialize([]byte{1, 0, 0, 0, 1, 0})
	if !errors.Is(err, ErrInvalidMagicByte) {
		t.Fatalf("expected ErrInvalidMagicByte, got %v", err)
	}
}

func TestProtoDeserializer_InvalidMessageIndexes(t *testing.T) {
	d := NewProtoDeserializer(nil)
	_, _, err := d.Deserialize([]byte{0, 0, 0, 0, 1, 1})
	if !errors.Is(err, ErrInvalidMessageIndexes) {
		t.Fatalf("expected ErrInvalidMessageIndexes, got %v", err)
	}
}
