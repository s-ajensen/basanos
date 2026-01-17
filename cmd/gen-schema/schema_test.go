package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindEventTypes_SingleEvent(t *testing.T) {
	source := `package event

type FooEvent struct {
	Name string
}`

	result := FindEventTypes(source)

	assert.Equal(t, []string{"FooEvent"}, result)
}

func TestFindEventTypes_MultipleEvents(t *testing.T) {
	source := `package event

type FooEvent struct {
	Name string
}

type BarEvent struct {
	ID int
}`

	result := FindEventTypes(source)

	assert.Equal(t, []string{"FooEvent", "BarEvent"}, result)
}

func TestFindEventTypes_ExcludesNonEventStructs(t *testing.T) {
	source := `package event

type FooEvent struct {
	Name string
}

type Config struct {
	X int
}

type BarEvent struct {
	ID int
}`

	result := FindEventTypes(source)

	assert.Equal(t, []string{"FooEvent", "BarEvent"}, result)
}

func TestFindEventTypes_ExcludesBaseEvent(t *testing.T) {
	source := `package event

type BaseEvent struct {
	Event string
	RunID string
}

type FooEvent struct {
	BaseEvent
	Name string
}`

	result := FindEventTypes(source)

	assert.Equal(t, []string{"FooEvent"}, result)
}

func TestExtractFields_StringAndIntTypes(t *testing.T) {
	source := `package event

type FooEvent struct {
	Name  string ` + "`json:\"name\"`" + `
	Count int    ` + "`json:\"count\"`" + `
}`

	result := ExtractFields(source, "FooEvent")

	expected := []FieldInfo{
		{Name: "name", Type: "string", Required: true},
		{Name: "count", Type: "integer", Required: true},
	}
	assert.Equal(t, expected, result)
}

func TestExtractFields_OmitemptyMarksFieldNotRequired(t *testing.T) {
	source := `package event

type FooEvent struct {
	ID   string ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name,omitempty\"`" + `
}`

	result := ExtractFields(source, "FooEvent")

	expected := []FieldInfo{
		{Name: "id", Type: "string", Required: true},
		{Name: "name", Type: "string", Required: false},
	}
	assert.Equal(t, expected, result)
}

func TestExtractFields_NoJsonTagUsesGoFieldName(t *testing.T) {
	source := `package event

type FooEvent struct {
	Value string
}`

	result := ExtractFields(source, "FooEvent")

	expected := []FieldInfo{
		{Name: "Value", Type: "string", Required: true},
	}
	assert.Equal(t, expected, result)
}

func TestExtractFields_TimeTypeMapsToString(t *testing.T) {
	source := `package event

import "time"

type FooEvent struct {
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
}`

	result := ExtractFields(source, "FooEvent")

	expected := []FieldInfo{
		{Name: "created_at", Type: "string", Required: true},
	}
	assert.Equal(t, expected, result)
}

func TestGenerateSchema_SingleEventType(t *testing.T) {
	source := `package event

type FooEvent struct {
	Name string ` + "`json:\"name\"`" + `
}`

	result, err := GenerateSchema(source)

	assert.NoError(t, err)
	var schema map[string]interface{}
	assert.NoError(t, json.Unmarshal(result, &schema))
	assert.Equal(t, "https://json-schema.org/draft/2020-12/schema", schema["$schema"])
	defs := schema["$defs"].(map[string]interface{})
	assert.Contains(t, defs, "FooEvent")
	fooEvent := defs["FooEvent"].(map[string]interface{})
	assert.Equal(t, "object", fooEvent["type"])
	assert.Equal(t, false, fooEvent["additionalProperties"])
}

func TestGenerateSchema_MultipleEventsInOneOf(t *testing.T) {
	source := `package event

type FooEvent struct {
	Name string ` + "`json:\"name\"`" + `
}

type BarEvent struct {
	ID int ` + "`json:\"id\"`" + `
}`

	result, err := GenerateSchema(source)

	assert.NoError(t, err)
	var schema map[string]interface{}
	assert.NoError(t, json.Unmarshal(result, &schema))
	oneOf := schema["oneOf"].([]interface{})
	assert.Len(t, oneOf, 2)
	refs := []string{}
	for _, item := range oneOf {
		ref := item.(map[string]interface{})["$ref"].(string)
		refs = append(refs, ref)
	}
	assert.Contains(t, refs, "#/$defs/FooEvent")
	assert.Contains(t, refs, "#/$defs/BarEvent")
}

func TestGenerateSchema_RequiredFieldsInArray(t *testing.T) {
	source := `package event

type FooEvent struct {
	ID   string ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}`

	result, err := GenerateSchema(source)

	assert.NoError(t, err)
	var schema map[string]interface{}
	assert.NoError(t, json.Unmarshal(result, &schema))
	defs := schema["$defs"].(map[string]interface{})
	fooEvent := defs["FooEvent"].(map[string]interface{})
	required := fooEvent["required"].([]interface{})
	assert.Contains(t, required, "id")
	assert.Contains(t, required, "name")
}

func TestGenerateSchema_OmitemptyFieldsNotRequired(t *testing.T) {
	source := `package event

type FooEvent struct {
	ID   string ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name,omitempty\"`" + `
}`

	result, err := GenerateSchema(source)

	assert.NoError(t, err)
	var schema map[string]interface{}
	assert.NoError(t, json.Unmarshal(result, &schema))
	defs := schema["$defs"].(map[string]interface{})
	fooEvent := defs["FooEvent"].(map[string]interface{})
	required := fooEvent["required"].([]interface{})
	assert.Contains(t, required, "id")
	assert.NotContains(t, required, "name")
}

func TestExtractFields_EmbeddedStructFlattensFields(t *testing.T) {
	source := `package event

type BaseEvent struct {
	Event string ` + "`json:\"event\"`" + `
	RunID string ` + "`json:\"run_id\"`" + `
}

type FooEvent struct {
	BaseEvent
	Name string ` + "`json:\"name\"`" + `
}`

	result := ExtractFields(source, "FooEvent")

	expected := []FieldInfo{
		{Name: "event", Type: "string", Required: true},
		{Name: "run_id", Type: "string", Required: true},
		{Name: "name", Type: "string", Required: true},
	}
	assert.Equal(t, expected, result)
}
