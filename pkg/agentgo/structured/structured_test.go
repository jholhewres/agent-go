package structured

import (
	"testing"

	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

type SimpleStruct struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Count  int     `json:"count"`
	Active bool    `json:"active"`
}

type NestedStruct struct {
	Title  string       `json:"title"`
	Author SimpleStruct `json:"author"`
}

type WithOptionalFields struct {
	Required string `json:"required"`
	Optional string `json:"optional,omitempty"`
	AlsoReq  int    `json:"also_req"`
}

type WithSlice struct {
	Tags  []string       `json:"tags"`
	Items []SimpleStruct `json:"items"`
}

type WithDescription struct {
	Name string `json:"name" description:"The user's full name"`
	Age  int    `json:"age" description:"Age in years"`
}

type WithIgnored struct {
	Visible string `json:"visible"`
	Hidden  string `json:"-"`
	private string //nolint:unused
}

func TestSchemaFromType_SimpleStruct(t *testing.T) {
	schema, err := SchemaFromType(SimpleStruct{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if schema.Name != "SimpleStruct" {
		t.Errorf("expected name 'SimpleStruct', got %q", schema.Name)
	}

	props, ok := schema.Schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties map")
	}

	nameSchema := props["name"].(map[string]interface{})
	if nameSchema["type"] != "string" {
		t.Errorf("expected name type 'string', got %v", nameSchema["type"])
	}

	scoreSchema := props["score"].(map[string]interface{})
	if scoreSchema["type"] != "number" {
		t.Errorf("expected score type 'number', got %v", scoreSchema["type"])
	}

	countSchema := props["count"].(map[string]interface{})
	if countSchema["type"] != "integer" {
		t.Errorf("expected count type 'integer', got %v", countSchema["type"])
	}

	activeSchema := props["active"].(map[string]interface{})
	if activeSchema["type"] != "boolean" {
		t.Errorf("expected active type 'boolean', got %v", activeSchema["type"])
	}
}

func TestSchemaFromType_Pointer(t *testing.T) {
	schema, err := SchemaFromType(&SimpleStruct{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.Name != "SimpleStruct" {
		t.Errorf("expected name 'SimpleStruct', got %q", schema.Name)
	}
}

func TestSchemaFromType_NonStruct(t *testing.T) {
	_, err := SchemaFromType("not a struct")
	if err == nil {
		t.Fatal("expected error for non-struct type")
	}
}

func TestSchemaFromType_NestedStruct(t *testing.T) {
	schema, err := SchemaFromType(NestedStruct{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	props := schema.Schema["properties"].(map[string]interface{})
	authorSchema := props["author"].(map[string]interface{})
	if authorSchema["type"] != "object" {
		t.Errorf("expected nested type 'object', got %v", authorSchema["type"])
	}
	authorProps := authorSchema["properties"].(map[string]interface{})
	if _, ok := authorProps["name"]; !ok {
		t.Error("expected nested struct to have 'name' property")
	}
}

func TestSchemaFromType_RequiredFields(t *testing.T) {
	schema, err := SchemaFromType(WithOptionalFields{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	required, ok := schema.Schema["required"].([]string)
	if !ok {
		t.Fatal("expected required array")
	}

	reqMap := make(map[string]bool)
	for _, r := range required {
		reqMap[r] = true
	}

	if !reqMap["required"] {
		t.Error("'required' field should be in required list")
	}
	if !reqMap["also_req"] {
		t.Error("'also_req' field should be in required list")
	}
	if reqMap["optional"] {
		t.Error("'optional' field should NOT be in required list")
	}
}

func TestSchemaFromType_WithSlice(t *testing.T) {
	schema, err := SchemaFromType(WithSlice{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	props := schema.Schema["properties"].(map[string]interface{})

	tagsSchema := props["tags"].(map[string]interface{})
	if tagsSchema["type"] != "array" {
		t.Errorf("expected tags type 'array', got %v", tagsSchema["type"])
	}
	items := tagsSchema["items"].(map[string]interface{})
	if items["type"] != "string" {
		t.Errorf("expected tags items type 'string', got %v", items["type"])
	}

	itemsSchema := props["items"].(map[string]interface{})
	if itemsSchema["type"] != "array" {
		t.Errorf("expected items type 'array', got %v", itemsSchema["type"])
	}
	itemItems := itemsSchema["items"].(map[string]interface{})
	if itemItems["type"] != "object" {
		t.Errorf("expected items items type 'object', got %v", itemItems["type"])
	}
}

func TestSchemaFromType_WithDescription(t *testing.T) {
	schema, err := SchemaFromType(WithDescription{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	props := schema.Schema["properties"].(map[string]interface{})
	nameSchema := props["name"].(map[string]interface{})
	if nameSchema["description"] != "The user's full name" {
		t.Errorf("expected description tag, got %v", nameSchema["description"])
	}
}

func TestSchemaFromType_IgnoredFields(t *testing.T) {
	schema, err := SchemaFromType(WithIgnored{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	props := schema.Schema["properties"].(map[string]interface{})
	if _, ok := props["visible"]; !ok {
		t.Error("expected 'visible' property")
	}
	if _, ok := props["-"]; ok {
		t.Error("should not have '-' property")
	}
	if _, ok := props["Hidden"]; ok {
		t.Error("should not have 'Hidden' property (json:\"-\")")
	}
	if _, ok := props["private"]; ok {
		t.Error("should not have 'private' property (unexported)")
	}
}

func TestToResponseFormat(t *testing.T) {
	schema, _ := SchemaFromType(SimpleStruct{})
	rf := schema.ToResponseFormat()

	if rf.Type != "json_schema" {
		t.Errorf("expected type 'json_schema', got %q", rf.Type)
	}
	if rf.JSONSchema == nil {
		t.Error("expected non-nil JSONSchema")
	}
	if rf.JSONSchema["name"] != "SimpleStruct" {
		t.Errorf("expected JSONSchema name 'SimpleStruct', got %v", rf.JSONSchema["name"])
	}
	if _, ok := rf.JSONSchema["properties"]; !ok {
		t.Error("expected JSONSchema to contain 'properties'")
	}
}

func TestToResponseFormat_WithDescription(t *testing.T) {
	schema, _ := SchemaFromType(SimpleStruct{})
	schema.Description = "A simple test struct"
	rf := schema.ToResponseFormat()

	if rf.JSONSchema["description"] != "A simple test struct" {
		t.Errorf("expected description in JSONSchema, got %v", rf.JSONSchema["description"])
	}
}

func TestParseResponse_Valid(t *testing.T) {
	resp := &types.ModelResponse{
		Content: `{"name": "Alice", "score": 9.5, "count": 42, "active": true}`,
	}

	var result SimpleStruct
	err := ParseResponse(resp, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "Alice" {
		t.Errorf("expected name 'Alice', got %q", result.Name)
	}
	if result.Score != 9.5 {
		t.Errorf("expected score 9.5, got %f", result.Score)
	}
	if result.Count != 42 {
		t.Errorf("expected count 42, got %d", result.Count)
	}
	if !result.Active {
		t.Error("expected active true")
	}
}

func TestParseResponse_InvalidJSON(t *testing.T) {
	resp := &types.ModelResponse{Content: "not json"}
	var result SimpleStruct
	err := ParseResponse(resp, &result)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseResponse_EmptyContent(t *testing.T) {
	resp := &types.ModelResponse{Content: ""}
	var result SimpleStruct
	err := ParseResponse(resp, &result)
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestParseResponse_NilResponse(t *testing.T) {
	var result SimpleStruct
	err := ParseResponse(nil, &result)
	if err == nil {
		t.Fatal("expected error for nil response")
	}
}

func TestParseResponse_Nested(t *testing.T) {
	resp := &types.ModelResponse{
		Content: `{"title": "Book", "author": {"name": "Bob", "score": 8.0, "count": 1, "active": false}}`,
	}

	var result NestedStruct
	err := ParseResponse(resp, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Author.Name != "Bob" {
		t.Errorf("expected author name 'Bob', got %q", result.Author.Name)
	}
}
