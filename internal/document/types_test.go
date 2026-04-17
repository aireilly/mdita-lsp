package document

import "testing"

func TestHeadingIsTitle(t *testing.T) {
	h := &Heading{Level: 1, Text: "My Title", ID: "my-title"}
	if !h.IsTitle() {
		t.Error("Level 1 heading should be a title")
	}
	h2 := &Heading{Level: 2, Text: "Section", ID: "section"}
	if h2.IsTitle() {
		t.Error("Level 2 heading should not be a title")
	}
}

func TestDocKind(t *testing.T) {
	if Topic.String() != "topic" {
		t.Errorf("Topic.String() = %q, want %q", Topic.String(), "topic")
	}
	if Map.String() != "map" {
		t.Errorf("Map.String() = %q, want %q", Map.String(), "map")
	}
}

func TestDitaSchemaFromString(t *testing.T) {
	tests := []struct {
		input string
		want  DitaSchema
	}{
		{"urn:oasis:names:tc:dita:xsd:task.xsd", SchemaTask},
		{"urn:oasis:names:tc:dita:rng:task.rng", SchemaTask},
		{"urn:oasis:names:tc:dita:xsd:concept.xsd", SchemaConcept},
		{"urn:oasis:names:tc:dita:rng:concept.rng", SchemaConcept},
		{"urn:oasis:names:tc:dita:xsd:reference.xsd", SchemaReference},
		{"urn:oasis:names:tc:dita:xsd:topic.xsd", SchemaTopic},
		{"urn:oasis:names:tc:dita:xsd:map.xsd", SchemaMap},
		{"urn:oasis:names:tc:mdita:xsd:topic.xsd", SchemaMditaTopic},
		{"urn:oasis:names:tc:mdita:core:xsd:topic.xsd", SchemaMditaCoreTopic},
		{"urn:oasis:names:tc:mdita:extended:xsd:topic.xsd", SchemaMditaExtendedTopic},
		{"something-unknown", SchemaUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := DitaSchemaFromString(tt.input)
			if got != tt.want {
				t.Errorf("DitaSchemaFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSymKind(t *testing.T) {
	if DefKind.String() != "def" {
		t.Errorf("DefKind.String() = %q", DefKind.String())
	}
	if RefKind.String() != "ref" {
		t.Errorf("RefKind.String() = %q", RefKind.String())
	}
}
