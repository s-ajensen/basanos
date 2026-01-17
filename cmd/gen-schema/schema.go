package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

func FindEventTypes(source string) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, 0)
	if err != nil {
		return nil
	}
	return findEventTypesFromAST(file)
}

func findEventTypesFromAST(file *ast.File) []string {
	var events []string
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec := spec.(*ast.TypeSpec)
			if _, isStruct := typeSpec.Type.(*ast.StructType); !isStruct {
				continue
			}
			if !isEventTypeName(typeSpec.Name.Name) {
				continue
			}
			events = append(events, typeSpec.Name.Name)
		}
	}
	return events
}

func isEventTypeName(name string) bool {
	return strings.HasSuffix(name, "Event")
}

type FieldInfo struct {
	Name     string
	Type     string
	Required bool
}

func ExtractFields(source string, typeName string) []FieldInfo {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, 0)
	if err != nil {
		return nil
	}
	return extractFieldsFromAST(file, typeName)
}

func extractFieldsFromAST(file *ast.File, typeName string) []FieldInfo {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec := spec.(*ast.TypeSpec)
			if typeSpec.Name.Name != typeName {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			return extractFieldsFromStruct(file, structType)
		}
	}
	return nil
}

func extractFieldsFromStruct(file *ast.File, structType *ast.StructType) []FieldInfo {
	var fields []FieldInfo
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			embeddedTypeName := field.Type.(*ast.Ident).Name
			embeddedFields := extractFieldsFromAST(file, embeddedTypeName)
			fields = append(fields, embeddedFields...)
			continue
		}
		tagName, required := parseJsonTag(field.Tag)
		fields = append(fields, FieldInfo{
			Name:     fieldName(tagName, field.Names[0].Name),
			Type:     mapGoTypeToJsonSchema(field.Type),
			Required: required,
		})
	}
	return fields
}

func fieldName(tagName, goName string) string {
	if tagName != "" {
		return tagName
	}
	return goName
}

func parseJsonTag(tag *ast.BasicLit) (string, bool) {
	if tag == nil {
		return "", true
	}
	tagValue := tag.Value
	jsonPrefix := "`json:\""
	startIdx := strings.Index(tagValue, jsonPrefix)
	if startIdx == -1 {
		return "", true
	}
	startIdx += len(jsonPrefix)
	endIdx := strings.Index(tagValue[startIdx:], "\"")
	if endIdx == -1 {
		return "", true
	}
	jsonContent := tagValue[startIdx : startIdx+endIdx]
	parts := strings.Split(jsonContent, ",")
	name := parts[0]
	required := !strings.Contains(jsonContent, ",omitempty")
	return name, required
}

func mapGoTypeToJsonSchema(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "string":
			return "string"
		case "int":
			return "integer"
		}
	case *ast.SelectorExpr:
		if isTimeType(t) {
			return "string"
		}
	}
	return "string"
}

func isTimeType(expr *ast.SelectorExpr) bool {
	ident, ok := expr.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "time" && expr.Sel.Name == "Time"
}

func GenerateSchema(source string) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, 0)
	if err != nil {
		return nil, err
	}

	eventTypes := findEventTypesFromAST(file)

	defs := make(map[string]interface{})
	oneOf := make([]interface{}, 0, len(eventTypes))

	for _, typeName := range eventTypes {
		fields := extractFieldsFromAST(file, typeName)
		defs[typeName] = buildTypeDef(fields)
		oneOf = append(oneOf, map[string]string{"$ref": "#/$defs/" + typeName})
	}

	schema := map[string]interface{}{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"oneOf":   oneOf,
		"$defs":   defs,
	}

	return json.MarshalIndent(schema, "", "  ")
}

func buildTypeDef(fields []FieldInfo) map[string]interface{} {
	return map[string]interface{}{
		"type":                 "object",
		"properties":           buildProperties(fields),
		"required":             collectRequiredFields(fields),
		"additionalProperties": false,
	}
}

func buildProperties(fields []FieldInfo) map[string]interface{} {
	properties := make(map[string]interface{})
	for _, field := range fields {
		properties[field.Name] = map[string]string{"type": field.Type}
	}
	return properties
}

func collectRequiredFields(fields []FieldInfo) []string {
	required := make([]string, 0)
	for _, field := range fields {
		if !field.Required {
			continue
		}
		required = append(required, field.Name)
	}
	return required
}
