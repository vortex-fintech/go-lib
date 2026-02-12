package schemaregistry

import (
	"encoding/json"
	"testing"
)

func TestSchemaReference_MarshalJSON(t *testing.T) {
	refs := []SchemaReference{
		{Name: "reference.proto", Subject: "reference.v1-value", Version: 1},
	}

	data, err := json.Marshal(refs)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result []SchemaReference
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 reference, got %d", len(result))
	}

	if result[0].Name != "reference.proto" {
		t.Fatalf("expected name 'reference.proto', got '%s'", result[0].Name)
	}
}

func TestSchema_MarshalJSON(t *testing.T) {
	schema := Schema{
		Schema:     "syntax = \"proto3\"; message Test { string name = 1; }",
		SchemaType: "PROTOBUF",
		References: []SchemaReference{
			{Name: "common.proto", Subject: "common-value", Version: 1},
		},
	}

	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result Schema
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if result.SchemaType != "PROTOBUF" {
		t.Fatalf("expected type 'PROTOBUF', got '%s'", result.SchemaType)
	}

	if len(result.References) != 1 {
		t.Fatalf("expected 1 reference, got %d", len(result.References))
	}
}
