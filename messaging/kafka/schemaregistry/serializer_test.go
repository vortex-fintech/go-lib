package schemaregistry

import (
	"errors"
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestCreateWireFormat(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	schemaID := 42

	result := createWireFormat(data, schemaID, []int{0})

	if result[0] != 0 {
		t.Fatalf("expected magic byte 0, got %d", result[0])
	}

	if len(result) != len(data)+6 {
		t.Fatalf("expected length %d, got %d", len(data)+6, len(result))
	}

	gotID := int(result[1])<<24 | int(result[2])<<16 | int(result[3])<<8 | int(result[4])
	if gotID != schemaID {
		t.Fatalf("expected schema ID %d, got %d", schemaID, gotID)
	}

	if result[5] != 0 {
		t.Fatalf("expected message-index shortcut byte 0, got %d", result[5])
	}
}

func TestProtoSerializer_Caching(t *testing.T) {
	registry := &mockRegistry{
		schemas: map[string]string{
			"test-value": `syntax = "proto3"; message Int32Value { int32 value = 1; }`,
		},
		ids: map[string]int{
			"test-value": 123,
		},
	}
	serializer := NewProtoSerializer(registry)

	msg := &wrapperspb.Int32Value{Value: 42}

	encoded, firstID, err := serializer.SerializeWithSchema(
		"test-value",
		`syntax = "proto3"; message Int32Value { int32 value = 1; }`,
		msg,
	)
	if err != nil {
		t.Fatalf("first serialize failed: %v", err)
	}

	if firstID != 123 {
		t.Fatalf("expected ID 123, got %d", firstID)
	}

	if len(encoded) < 6 || encoded[5] != 0 {
		t.Fatalf("expected protobuf message-index shortcut in header")
	}

	_, secondID, err := serializer.Serialize("test-value", msg)
	if err != nil {
		t.Fatalf("second serialize failed: %v", err)
	}

	if secondID != 123 {
		t.Fatalf("expected cached ID 123, got %d", secondID)
	}

	if len(registry.registerCalls) != 0 {
		t.Fatalf("expected 0 RegisterSchema calls, got %d", len(registry.registerCalls))
	}
	if len(registry.registerWithRefsCalls) != 1 {
		t.Fatalf("expected 1 RegisterSchemaWithRefs call, got %d", len(registry.registerWithRefsCalls))
	}
}

func TestProtoSerializer_MultipleSubjects(t *testing.T) {
	registry := &mockRegistry{
		schemas: map[string]string{},
		ids:     map[string]int{},
	}
	serializer := NewProtoSerializer(registry)

	msg1 := &wrapperspb.Int32Value{Value: 42}
	msg2 := &wrapperspb.Int32Value{Value: 99}

	_, id1, err := serializer.SerializeWithSchema(
		"test-value-1",
		`syntax = "proto3"; message Int32Value { int32 value = 1; }`,
		msg1,
	)
	if err != nil {
		t.Fatalf("first serialize failed: %v", err)
	}

	if id1 != 1 {
		t.Fatalf("expected ID 1, got %d", id1)
	}

	_, id2, err := serializer.SerializeWithSchema(
		"test-value-2",
		`syntax = "proto3"; message Int32Value { int32 value = 1; }`,
		msg2,
	)
	if err != nil {
		t.Fatalf("second serialize failed: %v", err)
	}

	if id2 != 2 {
		t.Fatalf("expected new ID 2, got %d", id2)
	}

	_, id3, err := serializer.Serialize("test-value-1", msg1)
	if err != nil {
		t.Fatalf("third serialize failed: %v", err)
	}

	if id3 != 1 {
		t.Fatalf("expected cached ID 1, got %d", id3)
	}

	if len(registry.registerWithRefsCalls) != 2 {
		t.Fatalf("expected 2 RegisterSchemaWithRefs calls, got %d", len(registry.registerWithRefsCalls))
	}
}

func TestProtoSerializer_RequiresSchemaOnFirstSerialize(t *testing.T) {
	registry := &mockRegistry{schemas: map[string]string{}, ids: map[string]int{}}
	serializer := NewProtoSerializer(registry)

	_, _, err := serializer.Serialize("test-value", &wrapperspb.Int32Value{Value: 1})
	if !errors.Is(err, ErrSchemaNotCached) {
		t.Fatalf("expected ErrSchemaNotCached, got %v", err)
	}
}

func TestProtoSerializer_RejectsEmptySchema(t *testing.T) {
	registry := &mockRegistry{schemas: map[string]string{}, ids: map[string]int{}}
	serializer := NewProtoSerializer(registry)

	_, _, err := serializer.SerializeWithSchema("test-value", "", &wrapperspb.Int32Value{Value: 1})
	if !errors.Is(err, ErrSchemaRequired) {
		t.Fatalf("expected ErrSchemaRequired, got %v", err)
	}
}

func TestProtoSerializer_ReRegistersOnSchemaChange(t *testing.T) {
	registry := &mockRegistry{schemas: map[string]string{}, ids: map[string]int{}}
	serializer := NewProtoSerializer(registry)
	msg := &wrapperspb.Int32Value{Value: 1}

	_, id1, err := serializer.SerializeWithSchema(
		"test-value",
		`syntax = "proto3"; message Int32Value { int32 value = 1; }`,
		msg,
	)
	if err != nil {
		t.Fatalf("first serialize failed: %v", err)
	}

	_, id2, err := serializer.SerializeWithSchema(
		"test-value",
		`syntax = "proto3"; message Int32Value { int32 value = 1; string note = 2; }`,
		msg,
	)
	if err != nil {
		t.Fatalf("second serialize failed: %v", err)
	}

	if id1 == id2 {
		t.Fatalf("expected different schema IDs for changed schema")
	}
	if len(registry.registerWithRefsCalls) != 2 {
		t.Fatalf("expected 2 RegisterSchemaWithRefs calls, got %d", len(registry.registerWithRefsCalls))
	}
}

type mockRegistry struct {
	schemas               map[string]string
	ids                   map[string]int
	getCalls              []string
	registerCalls         []string
	registerWithRefsCalls []registerCall
}

func (m *mockRegistry) GetLatestSchema(subject string) (string, int, error) {
	m.getCalls = append(m.getCalls, subject)
	schema, ok := m.schemas[subject]
	if !ok {
		return "", 0, nil
	}
	id, ok := m.ids[subject]
	if !ok {
		return "", 0, nil
	}
	return schema, id, nil
}

func (m *mockRegistry) RegisterSchema(subject, schema string) (int, error) {
	m.registerCalls = append(m.registerCalls, subject)
	if id, ok := m.ids[subject]; ok && m.schemas[subject] == schema {
		return id, nil
	}
	id := len(m.ids) + 1
	m.schemas[subject] = schema
	m.ids[subject] = id
	return id, nil
}

func (m *mockRegistry) RegisterSchemaWithRefs(subject, schema string, refs []SchemaReference) (int, error) {
	m.registerWithRefsCalls = append(m.registerWithRefsCalls, registerCall{subject: subject, schema: schema, refs: refs})
	if id, ok := m.ids[subject]; ok && m.schemas[subject] == schema {
		return id, nil
	}
	id := len(m.ids) + 1
	m.schemas[subject] = schema
	m.ids[subject] = id
	return id, nil
}

type registerCall struct {
	subject string
	schema  string
	refs    []SchemaReference
}
