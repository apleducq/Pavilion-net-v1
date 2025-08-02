package services

import (
	"fmt"
	"testing"
)

func TestDataTransformer_BasicTransformation(t *testing.T) {
	config := DataTransformerConfig{
		DefaultFormat:    "json",
		EnableEnrichment: false,
		MissingDataPolicy: MissingDataPolicy{
			Strategy: MissingDataSkip,
		},
	}
	
	transformer := NewDataTransformer(config)
	
	// Test basic transformation
	sourceData := map[string]interface{}{
		"name": "John Doe",
		"age":  30,
		"email": "john@example.com",
	}
	
	transformations := []TransformationRule{
		{
			SourceField:    "name",
			TargetField:    "full_name",
			Transformation: "copy",
		},
		{
			SourceField:    "age",
			TargetField:    "user_age",
			Transformation: "copy",
		},
		{
			SourceField:    "email",
			TargetField:    "user_email",
			Transformation: "lowercase",
		},
	}
	
	req := TransformationRequest{
		Data:           sourceData,
		Transformations: transformations,
	}
	
	response := transformer.TransformData(req)
	
	if !response.Success {
		t.Errorf("Expected successful transformation, got errors: %v", response.Errors)
	}
	
	if response.Metrics.TotalFields != 3 {
		t.Errorf("Expected 3 total fields, got %d", response.Metrics.TotalFields)
	}
	
	if response.Metrics.TransformedFields != 3 {
		t.Errorf("Expected 3 transformed fields, got %d", response.Metrics.TransformedFields)
	}
}

func TestDataTransformer_TypeConversions(t *testing.T) {
	config := DataTransformerConfig{
		DefaultFormat:    "json",
		EnableEnrichment: false,
		MissingDataPolicy: MissingDataPolicy{
			Strategy: MissingDataSkip,
		},
	}
	
	transformer := NewDataTransformer(config)
	
	sourceData := map[string]interface{}{
		"string_value": "42",
		"number_value": 123.45,
		"boolean_value": "true",
		"mixed_value": "hello world",
	}
	
	transformations := []TransformationRule{
		{
			SourceField:    "string_value",
			TargetField:    "converted_number",
			Transformation: "number",
		},
		{
			SourceField:    "number_value",
			TargetField:    "converted_string",
			Transformation: "string",
		},
		{
			SourceField:    "boolean_value",
			TargetField:    "converted_boolean",
			Transformation: "boolean",
		},
		{
			SourceField:    "mixed_value",
			TargetField:    "uppercase_value",
			Transformation: "uppercase",
		},
	}
	
	req := TransformationRequest{
		Data:           sourceData,
		Transformations: transformations,
	}
	
	response := transformer.TransformData(req)
	
	if !response.Success {
		t.Errorf("Expected successful transformation, got errors: %v", response.Errors)
	}
	
	// Check transformed data
	result, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}
	
	if result["converted_number"] != 42.0 {
		t.Errorf("Expected 42.0, got %v", result["converted_number"])
	}
	
	if result["converted_string"] != "123.45" {
		t.Errorf("Expected '123.45', got %v", result["converted_string"])
	}
	
	if result["converted_boolean"] != true {
		t.Errorf("Expected true, got %v", result["converted_boolean"])
	}
	
	if result["uppercase_value"] != "HELLO WORLD" {
		t.Errorf("Expected 'HELLO WORLD', got %v", result["uppercase_value"])
	}
}

func TestDataTransformer_StringTransformations(t *testing.T) {
	config := DataTransformerConfig{
		DefaultFormat:    "json",
		EnableEnrichment: false,
		MissingDataPolicy: MissingDataPolicy{
			Strategy: MissingDataSkip,
		},
	}
	
	transformer := NewDataTransformer(config)
	
	sourceData := map[string]interface{}{
		"name": "  John Doe  ",
		"email": "JOHN@EXAMPLE.COM",
		"text": "hello,world,test",
		"message": "hello world",
	}
	
	transformations := []TransformationRule{
		{
			SourceField:    "name",
			TargetField:    "trimmed_name",
			Transformation: "trim",
		},
		{
			SourceField:    "email",
			TargetField:    "lowercase_email",
			Transformation: "lowercase",
		},
		{
			SourceField:    "text",
			TargetField:    "split_text",
			Transformation: "split",
			Parameters: map[string]interface{}{
				"separator": ",",
			},
		},
		{
			SourceField:    "message",
			TargetField:    "replaced_message",
			Transformation: "replace",
			Parameters: map[string]interface{}{
				"old": "hello",
				"new": "hi",
			},
		},
	}
	
	req := TransformationRequest{
		Data:           sourceData,
		Transformations: transformations,
	}
	
	response := transformer.TransformData(req)
	
	if !response.Success {
		t.Errorf("Expected successful transformation, got errors: %v", response.Errors)
	}
	
	result, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}
	
	if result["trimmed_name"] != "John Doe" {
		t.Errorf("Expected 'John Doe', got %v", result["trimmed_name"])
	}
	
	if result["lowercase_email"] != "john@example.com" {
		t.Errorf("Expected 'john@example.com', got %v", result["lowercase_email"])
	}
	
	splitResult, ok := result["split_text"].([]interface{})
	if !ok {
		t.Fatal("Expected array result for split")
	}
	
	if len(splitResult) != 3 {
		t.Errorf("Expected 3 items, got %d", len(splitResult))
	}
	
	if result["replaced_message"] != "hi world" {
		t.Errorf("Expected 'hi world', got %v", result["replaced_message"])
	}
}

func TestDataTransformer_DateTransformations(t *testing.T) {
	config := DataTransformerConfig{
		DefaultFormat:    "json",
		EnableEnrichment: false,
		MissingDataPolicy: MissingDataPolicy{
			Strategy: MissingDataSkip,
		},
	}
	
	transformer := NewDataTransformer(config)
	
	sourceData := map[string]interface{}{
		"date": "2023-01-15",
		"timestamp": int64(1642204800), // 2022-01-15 00:00:00 UTC
		"date_string": "15/01/2023",
	}
	
	transformations := []TransformationRule{
		{
			SourceField:    "date",
			TargetField:    "formatted_date",
			Transformation: "format_date",
			Parameters: map[string]interface{}{
				"format": "2006-01-02",
			},
		},
		{
			SourceField:    "timestamp",
			TargetField:    "formatted_timestamp",
			Transformation: "format_date",
			Parameters: map[string]interface{}{
				"format": "2006-01-02",
			},
		},
		{
			SourceField:    "date_string",
			TargetField:    "parsed_date",
			Transformation: "parse_date",
			Parameters: map[string]interface{}{
				"format": "02/01/2006",
			},
		},
	}
	
	req := TransformationRequest{
		Data:           sourceData,
		Transformations: transformations,
	}
	
	response := transformer.TransformData(req)
	
	if !response.Success {
		t.Errorf("Expected successful transformation, got errors: %v", response.Errors)
	}
	
	result, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}
	
	if result["formatted_date"] != "2023-01-15" {
		t.Errorf("Expected '2023-01-15', got %v", result["formatted_date"])
	}
	
	if result["formatted_timestamp"] != "2022-01-15" {
		t.Errorf("Expected '2022-01-15', got %v", result["formatted_timestamp"])
	}
	
	if result["parsed_date"] != "2023-01-15" {
		t.Errorf("Expected '2023-01-15', got %v", result["parsed_date"])
	}
}

func TestDataTransformer_MissingDataHandling(t *testing.T) {
	config := DataTransformerConfig{
		DefaultFormat:    "json",
		EnableEnrichment: false,
		MissingDataPolicy: MissingDataPolicy{
			Strategy: MissingDataDefault,
			DefaultValue: "default_value",
		},
	}
	
	transformer := NewDataTransformer(config)
	
	sourceData := map[string]interface{}{
		"existing_field": "value",
		"null_field": nil,
	}
	
	transformations := []TransformationRule{
		{
			SourceField:    "existing_field",
			TargetField:    "transformed_field",
			Transformation: "copy",
		},
		{
			SourceField:    "missing_field",
			TargetField:    "default_field",
			Transformation: "default",
			DefaultValue:   "custom_default",
		},
		{
			SourceField:    "null_field",
			TargetField:    "null_transformed",
			Transformation: "string",
		},
	}
	
	req := TransformationRequest{
		Data:           sourceData,
		Transformations: transformations,
	}
	
	response := transformer.TransformData(req)
	
	if !response.Success {
		t.Errorf("Expected successful transformation, got errors: %v", response.Errors)
	}
	
	result, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}
	
	if result["transformed_field"] != "value" {
		t.Errorf("Expected 'value', got %v", result["transformed_field"])
	}
	
	if result["default_field"] != "custom_default" {
		t.Errorf("Expected 'custom_default', got %v", result["default_field"])
	}
	
	if result["null_transformed"] != "" {
		t.Errorf("Expected empty string, got %v", result["null_transformed"])
	}
}

func TestDataTransformer_SchemaTransformation(t *testing.T) {
	config := DataTransformerConfig{
		DefaultFormat:    "json",
		EnableEnrichment: false,
		MissingDataPolicy: MissingDataPolicy{
			Strategy: MissingDataSkip,
		},
	}
	
	transformer := NewDataTransformer(config)
	
	sourceData := map[string]interface{}{
		"name": "John Doe",
		"age":  "30",
		"active": "true",
		"score": "95.5",
	}
	
	targetSchema := TransformationSchema{
		Type: "object",
		Properties: map[string]SchemaField{
			"name": {
				Type: "string",
			},
			"age": {
				Type: "integer",
			},
			"active": {
				Type: "boolean",
			},
			"score": {
				Type: "number",
			},
		},
	}
	
	req := TransformationRequest{
		Data:         sourceData,
		TargetSchema: targetSchema,
	}
	
	response := transformer.TransformData(req)
	
	if !response.Success {
		t.Errorf("Expected successful transformation, got errors: %v", response.Errors)
	}
	
	result, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}
	
	if result["name"] != "John Doe" {
		t.Errorf("Expected 'John Doe', got %v", result["name"])
	}
	
	if result["age"] != 30 {
		t.Errorf("Expected 30, got %v", result["age"])
	}
	
	if result["active"] != true {
		t.Errorf("Expected true, got %v", result["active"])
	}
	
	if result["score"] != 95.5 {
		t.Errorf("Expected 95.5, got %v", result["score"])
	}
}

func TestDataTransformer_CustomTransformations(t *testing.T) {
	config := DataTransformerConfig{
		DefaultFormat:    "json",
		EnableEnrichment: false,
		MissingDataPolicy: MissingDataPolicy{
			Strategy: MissingDataSkip,
		},
		CustomTransformers: map[string]TransformFunction{
			"double_value": {
				Name: "double_value",
				Function: func(value interface{}) (interface{}, error) {
					if num, ok := value.(float64); ok {
						return num * 2, nil
					}
					return nil, fmt.Errorf("value must be number")
				},
			},
			"reverse_string": {
				Name: "reverse_string",
				Function: func(value interface{}) (interface{}, error) {
					if str, ok := value.(string); ok {
						runes := []rune(str)
						for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
							runes[i], runes[j] = runes[j], runes[i]
						}
						return string(runes), nil
					}
					return nil, fmt.Errorf("value must be string")
				},
			},
		},
	}
	
	transformer := NewDataTransformer(config)
	
	sourceData := map[string]interface{}{
		"number": 42.0,
		"text":   "hello",
	}
	
	transformations := []TransformationRule{
		{
			SourceField:    "number",
			TargetField:    "doubled",
			Transformation: "custom",
			Parameters: map[string]interface{}{
				"name": "double_value",
			},
		},
		{
			SourceField:    "text",
			TargetField:    "reversed",
			Transformation: "custom",
			Parameters: map[string]interface{}{
				"name": "reverse_string",
			},
		},
	}
	
	req := TransformationRequest{
		Data:           sourceData,
		Transformations: transformations,
	}
	
	response := transformer.TransformData(req)
	
	if !response.Success {
		t.Errorf("Expected successful transformation, got errors: %v", response.Errors)
	}
	
	result, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}
	
	if result["doubled"] != 84.0 {
		t.Errorf("Expected 84.0, got %v", result["doubled"])
	}
	
	if result["reversed"] != "olleh" {
		t.Errorf("Expected 'olleh', got %v", result["reversed"])
	}
}

func TestDataTransformer_DataEnrichment(t *testing.T) {
	config := DataTransformerConfig{
		DefaultFormat:    "json",
		EnableEnrichment: true,
		MissingDataPolicy: MissingDataPolicy{
			Strategy: MissingDataSkip,
		},
	}
	
	transformer := NewDataTransformer(config)
	
	sourceData := map[string]interface{}{
		"name": "John Doe",
		"age":  30,
	}
	
	req := TransformationRequest{
		Data: sourceData,
		Options: TransformationOptions{
			EnableEnrichment: true,
		},
	}
	
	response := transformer.TransformData(req)
	
	if !response.Success {
		t.Errorf("Expected successful transformation, got errors: %v", response.Errors)
	}
	
	result, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}
	
	// Check that enrichment fields were added
	if _, exists := result["_transformed_at"]; !exists {
		t.Error("Expected _transformed_at field")
	}
	
	if _, exists := result["_metadata"]; !exists {
		t.Error("Expected _metadata field")
	}
	
	if response.Metrics.EnrichmentCount != 2 {
		t.Errorf("Expected 2 enrichment operations, got %d", response.Metrics.EnrichmentCount)
	}
}

func TestDataTransformer_ErrorHandling(t *testing.T) {
	config := DataTransformerConfig{
		DefaultFormat:    "json",
		EnableEnrichment: false,
		MissingDataPolicy: MissingDataPolicy{
			Strategy: MissingDataError,
		},
	}
	
	transformer := NewDataTransformer(config)
	
	sourceData := map[string]interface{}{
		"valid_field": "value",
	}
	
	transformations := []TransformationRule{
		{
			SourceField:    "missing_field",
			TargetField:    "error_field",
			Transformation: "copy",
		},
		{
			SourceField:    "valid_field",
			TargetField:    "invalid_conversion",
			Transformation: "number",
		},
	}
	
	req := TransformationRequest{
		Data:           sourceData,
		Transformations: transformations,
	}
	
	response := transformer.TransformData(req)
	
	// Should have errors due to missing field and invalid conversion
	if len(response.Errors) == 0 {
		t.Error("Expected transformation errors")
	}
	
	foundMissingFieldError := false
	foundConversionError := false
	
	for _, err := range response.Errors {
		if err.Code == "MISSING_SOURCE_FIELD" {
			foundMissingFieldError = true
		}
		if err.Code == "TRANSFORMATION_ERROR" {
			foundConversionError = true
		}
	}
	
	if !foundMissingFieldError {
		t.Error("Expected MISSING_SOURCE_FIELD error")
	}
	
	if !foundConversionError {
		t.Error("Expected TRANSFORMATION_ERROR")
	}
}

func TestDataTransformer_Metrics(t *testing.T) {
	config := DataTransformerConfig{
		DefaultFormat:    "json",
		EnableEnrichment: false,
		MissingDataPolicy: MissingDataPolicy{
			Strategy: MissingDataSkip,
		},
	}
	
	transformer := NewDataTransformer(config)
	
	sourceData := map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
		"field3": "value3",
	}
	
	transformations := []TransformationRule{
		{
			SourceField:    "field1",
			TargetField:    "transformed1",
			Transformation: "copy",
		},
		{
			SourceField:    "field2",
			TargetField:    "transformed2",
			Transformation: "uppercase",
		},
		{
			SourceField:    "field3",
			TargetField:    "transformed3",
			Transformation: "lowercase",
		},
	}
	
	req := TransformationRequest{
		Data:           sourceData,
		Transformations: transformations,
	}
	
	response := transformer.TransformData(req)
	
	if !response.Success {
		t.Errorf("Expected successful transformation, got errors: %v", response.Errors)
	}
	
	// Check metrics
	if response.Metrics.TotalFields != 3 {
		t.Errorf("Expected 3 total fields, got %d", response.Metrics.TotalFields)
	}
	
	if response.Metrics.TransformedFields != 3 {
		t.Errorf("Expected 3 transformed fields, got %d", response.Metrics.TransformedFields)
	}
	
	if response.Metrics.ProcessingTime <= 0 {
		t.Error("Expected positive processing time")
	}
}

func TestDataTransformer_Integration(t *testing.T) {
	config := DataTransformerConfig{
		DefaultFormat:    "json",
		EnableEnrichment: true,
		MissingDataPolicy: MissingDataPolicy{
			Strategy: MissingDataDefault,
			DefaultValue: "unknown",
		},
		CustomTransformers: map[string]TransformFunction{
			"add_prefix": {
				Name: "add_prefix",
				Function: func(value interface{}) (interface{}, error) {
					if str, ok := value.(string); ok {
						return "prefix_" + str, nil
					}
					return nil, fmt.Errorf("value must be string")
				},
			},
		},
	}
	
	transformer := NewDataTransformer(config)
	
	sourceData := map[string]interface{}{
		"user_id": "12345",
		"name":    "John Doe",
		"email":   "JOHN@EXAMPLE.COM",
		"age":     "30",
		"status":  "active",
	}
	
	transformations := []TransformationRule{
		{
			SourceField:    "user_id",
			TargetField:    "id",
			Transformation: "copy",
		},
		{
			SourceField:    "name",
			TargetField:    "full_name",
			Transformation: "trim",
		},
		{
			SourceField:    "email",
			TargetField:    "user_email",
			Transformation: "lowercase",
		},
		{
			SourceField:    "age",
			TargetField:    "user_age",
			Transformation: "integer",
		},
		{
			SourceField:    "status",
			TargetField:    "user_status",
			Transformation: "custom",
			Parameters: map[string]interface{}{
				"name": "add_prefix",
			},
		},
	}
	
	targetSchema := TransformationSchema{
		Type: "object",
		Properties: map[string]SchemaField{
			"id": {
				Type: "string",
			},
			"full_name": {
				Type: "string",
			},
			"user_email": {
				Type: "string",
			},
			"user_age": {
				Type: "integer",
			},
			"user_status": {
				Type: "string",
			},
		},
	}
	
	req := TransformationRequest{
		Data:           sourceData,
		TargetSchema:   targetSchema,
		Transformations: transformations,
		Options: TransformationOptions{
			EnableEnrichment: true,
		},
	}
	
	response := transformer.TransformData(req)
	
	if !response.Success {
		t.Errorf("Expected successful transformation, got errors: %v", response.Errors)
	}
	
	result, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}
	
	// Check transformed values
	if result["id"] != "12345" {
		t.Errorf("Expected '12345', got %v", result["id"])
	}
	
	if result["full_name"] != "John Doe" {
		t.Errorf("Expected 'John Doe', got %v", result["full_name"])
	}
	
	if result["user_email"] != "john@example.com" {
		t.Errorf("Expected 'john@example.com', got %v", result["user_email"])
	}
	
	if result["user_age"] != 30 {
		t.Errorf("Expected 30, got %v", result["user_age"])
	}
	
	if result["user_status"] != "prefix_active" {
		t.Errorf("Expected 'prefix_active', got %v", result["user_status"])
	}
	
	// Check enrichment
	if _, exists := result["_transformed_at"]; !exists {
		t.Error("Expected _transformed_at field")
	}
	
	if _, exists := result["_metadata"]; !exists {
		t.Error("Expected _metadata field")
	}
	
	// Check metrics
	if response.Metrics.TotalFields == 0 {
		t.Error("Expected non-zero total fields")
	}
	
	if response.Metrics.TransformedFields == 0 {
		t.Error("Expected non-zero transformed fields")
	}
	
	if response.Metrics.ProcessingTime <= 0 {
		t.Error("Expected positive processing time")
	}
} 