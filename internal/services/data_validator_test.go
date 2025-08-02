package services

import (
	"testing"
	"time"
)

func TestDataValidator_BasicValidation(t *testing.T) {
	config := DataValidatorConfig{
		StrictMode:    false,
		MaxErrors:     100,
		EnableMetrics: true,
	}
	
	validator := NewDataValidator(config)
	
	// Test basic string validation
	schema := ValidationSchema{
		Type: "object",
		Properties: map[string]SchemaField{
			"name": {
				Type:     "string",
				Required: true,
			},
			"age": {
				Type:     "integer",
				Required: true,
			},
		},
		Required: []string{"name", "age"},
	}
	
	data := map[string]interface{}{
		"name": "John Doe",
		"age":  30,
	}
	
	req := ValidationRequest{
		Data:   data,
		Schema: schema,
	}
	
	response := validator.ValidateData(req)
	
	if !response.Valid {
		t.Errorf("Expected valid data, got errors: %v", response.Errors)
	}
	
	if response.Metrics.TotalFields != 2 {
		t.Errorf("Expected 2 total fields, got %d", response.Metrics.TotalFields)
	}
	
	if response.Metrics.ValidFields != 2 {
		t.Errorf("Expected 2 valid fields, got %d", response.Metrics.ValidFields)
	}
}

func TestDataValidator_TypeValidation(t *testing.T) {
	config := DataValidatorConfig{
		StrictMode:    false,
		MaxErrors:     100,
		EnableMetrics: true,
	}
	
	validator := NewDataValidator(config)
	
	schema := ValidationSchema{
		Type: "object",
		Properties: map[string]SchemaField{
			"string_field": {
				Type: "string",
			},
			"number_field": {
				Type: "number",
			},
			"integer_field": {
				Type: "integer",
			},
			"boolean_field": {
				Type: "boolean",
			},
		},
	}
	
	data := map[string]interface{}{
		"string_field":  "test",
		"number_field":  123.45,
		"integer_field": 42,
		"boolean_field": true,
	}
	
	req := ValidationRequest{
		Data:   data,
		Schema: schema,
	}
	
	response := validator.ValidateData(req)
	
	if !response.Valid {
		t.Errorf("Expected valid data, got errors: %v", response.Errors)
	}
}

func TestDataValidator_TypeMismatch(t *testing.T) {
	config := DataValidatorConfig{
		StrictMode:    false,
		MaxErrors:     100,
		EnableMetrics: true,
	}
	
	validator := NewDataValidator(config)
	
	schema := ValidationSchema{
		Type: "object",
		Properties: map[string]SchemaField{
			"string_field": {
				Type: "string",
			},
		},
	}
	
	data := map[string]interface{}{
		"string_field": 123, // Wrong type
	}
	
	req := ValidationRequest{
		Data:   data,
		Schema: schema,
	}
	
	response := validator.ValidateData(req)
	
	if response.Valid {
		t.Error("Expected invalid data due to type mismatch")
	}
	
	if len(response.Errors) == 0 {
		t.Error("Expected validation errors")
	}
	
	// Check for specific error
	foundTypeError := false
	for _, err := range response.Errors {
		if err.Code == "TYPE_MISMATCH" {
			foundTypeError = true
			break
		}
	}
	
	if !foundTypeError {
		t.Error("Expected TYPE_MISMATCH error")
	}
}

func TestDataValidator_RequiredFields(t *testing.T) {
	config := DataValidatorConfig{
		StrictMode:    false,
		MaxErrors:     100,
		EnableMetrics: true,
	}
	
	validator := NewDataValidator(config)
	
	schema := ValidationSchema{
		Type: "object",
		Properties: map[string]SchemaField{
			"required_field": {
				Type:     "string",
				Required: true,
			},
			"optional_field": {
				Type: "string",
			},
		},
		Required: []string{"required_field"},
	}
	
	// Test missing required field
	data := map[string]interface{}{
		"optional_field": "test",
	}
	
	req := ValidationRequest{
		Data:   data,
		Schema: schema,
	}
	
	response := validator.ValidateData(req)
	
	if response.Valid {
		t.Error("Expected invalid data due to missing required field")
	}
	
	foundRequiredError := false
	for _, err := range response.Errors {
		if err.Code == "REQUIRED_FIELD_MISSING" {
			foundRequiredError = true
			break
		}
	}
	
	if !foundRequiredError {
		t.Error("Expected REQUIRED_FIELD_MISSING error")
	}
}

func TestDataValidator_StringConstraints(t *testing.T) {
	config := DataValidatorConfig{
		StrictMode:    false,
		MaxErrors:     100,
		EnableMetrics: true,
	}
	
	validator := NewDataValidator(config)
	
	minLength := 5
	maxLength := 10
	pattern := "^[A-Za-z]+$"
	enum := []interface{}{"apple", "banana", "cherry"}
	
	schema := ValidationSchema{
		Type: "object",
		Properties: map[string]SchemaField{
			"min_length_field": {
				Type:      "string",
				MinLength: &minLength,
			},
			"max_length_field": {
				Type:      "string",
				MaxLength: &maxLength,
			},
			"pattern_field": {
				Type:    "string",
				Pattern: &pattern,
			},
			"enum_field": {
				Type: "string",
				Enum:  enum,
			},
			"format_field": {
				Type:    "string",
				Format:  &[]string{"email"}[0],
			},
		},
	}
	
	// Test valid data
	validData := map[string]interface{}{
		"min_length_field": "hello",
		"max_length_field": "short",
		"pattern_field":    "Hello",
		"enum_field":       "apple",
		"format_field":     "test@example.com",
	}
	
	req := ValidationRequest{
		Data:   validData,
		Schema: schema,
	}
	
	response := validator.ValidateData(req)
	
	if !response.Valid {
		t.Errorf("Expected valid data, got errors: %v", response.Errors)
	}
}

func TestDataValidator_NumberConstraints(t *testing.T) {
	config := DataValidatorConfig{
		StrictMode:    false,
		MaxErrors:     100,
		EnableMetrics: true,
	}
	
	validator := NewDataValidator(config)
	
	minValue := 10.0
	maxValue := 100.0
	
	schema := ValidationSchema{
		Type: "object",
		Properties: map[string]SchemaField{
			"number_field": {
				Type:     "number",
				MinValue: &minValue,
				MaxValue: &maxValue,
			},
			"integer_field": {
				Type:     "integer",
				MinValue: &minValue,
				MaxValue: &maxValue,
			},
		},
	}
	
	// Test valid data
	validData := map[string]interface{}{
		"number_field":  50.0,
		"integer_field": 50,
	}
	
	req := ValidationRequest{
		Data:   validData,
		Schema: schema,
	}
	
	response := validator.ValidateData(req)
	
	if !response.Valid {
		t.Errorf("Expected valid data, got errors: %v", response.Errors)
	}
}

func TestDataValidator_ArrayValidation(t *testing.T) {
	config := DataValidatorConfig{
		StrictMode:    false,
		MaxErrors:     100,
		EnableMetrics: true,
	}
	
	validator := NewDataValidator(config)
	
	schema := ValidationSchema{
		Type: "object",
		Properties: map[string]SchemaField{
			"array_field": {
				Type: "array",
			},
		},
	}
	
	// Test valid array
	validData := map[string]interface{}{
		"array_field": []interface{}{"item1", "item2", "item3"},
	}
	
	req := ValidationRequest{
		Data:   validData,
		Schema: schema,
	}
	
	response := validator.ValidateData(req)
	
	if !response.Valid {
		t.Errorf("Expected valid data, got errors: %v", response.Errors)
	}
}

func TestDataValidator_FormatValidation(t *testing.T) {
	config := DataValidatorConfig{
		StrictMode:    false,
		MaxErrors:     100,
		EnableMetrics: true,
	}
	
	validator := NewDataValidator(config)
	
	emailFormat := "email"
	uriFormat := "uri"
	dateFormat := "date"
	dateTimeFormat := "date-time"
	uuidFormat := "uuid"
	
	schema := ValidationSchema{
		Type: "object",
		Properties: map[string]SchemaField{
			"email_field": {
				Type:    "string",
				Format:  &emailFormat,
			},
			"uri_field": {
				Type:    "string",
				Format:  &uriFormat,
			},
			"date_field": {
				Type:    "string",
				Format:  &dateFormat,
			},
			"datetime_field": {
				Type:    "string",
				Format:  &dateTimeFormat,
			},
			"uuid_field": {
				Type:    "string",
				Format:  &uuidFormat,
			},
		},
	}
	
	// Test valid formats
	validData := map[string]interface{}{
		"email_field":    "test@example.com",
		"uri_field":      "https://example.com",
		"date_field":     "2023-01-01",
		"datetime_field": time.Now().Format(time.RFC3339),
		"uuid_field":     "123e4567-e89b-12d3-a456-426614174000",
	}
	
	req := ValidationRequest{
		Data:   validData,
		Schema: schema,
	}
	
	response := validator.ValidateData(req)
	
	if !response.Valid {
		t.Errorf("Expected valid data, got errors: %v", response.Errors)
	}
}

func TestDataValidator_CustomRules(t *testing.T) {
	config := DataValidatorConfig{
		StrictMode:    false,
		MaxErrors:     100,
		EnableMetrics: true,
		CustomValidators: map[string]DataValidationRule{
			"positive_number": {
				Name: "positive_number",
				Function: func(value interface{}) (bool, string) {
					if num, ok := value.(float64); ok {
						if num > 0 {
							return true, ""
						}
						return false, "number must be positive"
					}
					return false, "invalid number type"
				},
			},
		},
	}
	
	validator := NewDataValidator(config)
	
	customRule := "positive_number"
	
	schema := ValidationSchema{
		Type: "object",
		Properties: map[string]SchemaField{
			"custom_field": {
				Type:       "number",
				CustomRule: &customRule,
			},
		},
	}
	
	// Test valid custom rule
	validData := map[string]interface{}{
		"custom_field": 42.0,
	}
	
	req := ValidationRequest{
		Data:   validData,
		Schema: schema,
	}
	
	response := validator.ValidateData(req)
	
	if !response.Valid {
		t.Errorf("Expected valid data, got errors: %v", response.Errors)
	}
	
	// Test invalid custom rule
	invalidData := map[string]interface{}{
		"custom_field": -5.0,
	}
	
	req = ValidationRequest{
		Data:   invalidData,
		Schema: schema,
	}
	
	response = validator.ValidateData(req)
	
	if response.Valid {
		t.Error("Expected invalid data due to custom rule violation")
	}
}

func TestDataValidator_StrictMode(t *testing.T) {
	config := DataValidatorConfig{
		StrictMode:    true,
		MaxErrors:     100,
		EnableMetrics: true,
	}
	
	validator := NewDataValidator(config)
	
	schema := ValidationSchema{
		Type: "object",
		Properties: map[string]SchemaField{
			"defined_field": {
				Type: "string",
			},
		},
	}
	
	// Test unknown field in strict mode
	data := map[string]interface{}{
		"defined_field": "test",
		"unknown_field": "test",
	}
	
	req := ValidationRequest{
		Data:   data,
		Schema: schema,
	}
	
	response := validator.ValidateData(req)
	
	// Should have warnings for unknown fields
	if len(response.Warnings) == 0 {
		t.Error("Expected warnings for unknown fields in strict mode")
	}
	
	foundUnknownFieldWarning := false
	for _, warning := range response.Warnings {
		if warning.Code == "UNKNOWN_FIELD" {
			foundUnknownFieldWarning = true
			break
		}
	}
	
	if !foundUnknownFieldWarning {
		t.Error("Expected UNKNOWN_FIELD warning")
	}
}

func TestDataValidator_Metrics(t *testing.T) {
	config := DataValidatorConfig{
		StrictMode:    false,
		MaxErrors:     100,
		EnableMetrics: true,
	}
	
	validator := NewDataValidator(config)
	
	schema := ValidationSchema{
		Type: "object",
		Properties: map[string]SchemaField{
			"field1": {
				Type: "string",
			},
			"field2": {
				Type: "integer",
			},
		},
	}
	
	data := map[string]interface{}{
		"field1": "test",
		"field2": 42,
	}
	
	req := ValidationRequest{
		Data:   data,
		Schema: schema,
		Options: ValidationOptions{
			EnableMetrics: true,
		},
	}
	
	response := validator.ValidateData(req)
	
	// Check metrics
	if response.Metrics.TotalFields != 2 {
		t.Errorf("Expected 2 total fields, got %d", response.Metrics.TotalFields)
	}
	
	if response.Metrics.ValidFields != 2 {
		t.Errorf("Expected 2 valid fields, got %d", response.Metrics.ValidFields)
	}
	
	if response.Metrics.ProcessingTime <= 0 {
		t.Error("Expected positive processing time")
	}
	
	if response.Metrics.QualityScore != 100.0 {
		t.Errorf("Expected 100%% quality score, got %f", response.Metrics.QualityScore)
	}
}

func TestDataValidator_ErrorHandling(t *testing.T) {
	config := DataValidatorConfig{
		StrictMode:    false,
		MaxErrors:     5, // Limit errors
		EnableMetrics: true,
	}
	
	validator := NewDataValidator(config)
	
	schema := ValidationSchema{
		Type: "object",
		Properties: map[string]SchemaField{
			"field1": {
				Type: "string",
			},
			"field2": {
				Type: "integer",
			},
			"field3": {
				Type: "boolean",
			},
			"field4": {
				Type: "number",
			},
			"field5": {
				Type: "string",
			},
			"field6": {
				Type: "integer",
			},
		},
	}
	
	// Data with multiple type mismatches
	data := map[string]interface{}{
		"field1": 123,        // Should be string
		"field2": "not_int",  // Should be integer
		"field3": "not_bool", // Should be boolean
		"field4": "not_num",  // Should be number
		"field5": 456,        // Should be string
		"field6": "not_int2", // Should be integer
	}
	
	req := ValidationRequest{
		Data:   data,
		Schema: schema,
	}
	
	response := validator.ValidateData(req)
	
	if response.Valid {
		t.Error("Expected invalid data due to type mismatches")
	}
	
	// Should respect max errors limit
	if len(response.Errors) > config.MaxErrors {
		t.Errorf("Expected max %d errors, got %d", config.MaxErrors, len(response.Errors))
	}
}

func TestDataValidator_Integration(t *testing.T) {
	config := DataValidatorConfig{
		StrictMode:    false,
		MaxErrors:     100,
		EnableMetrics: true,
	}
	
	validator := NewDataValidator(config)
	
	// Complex schema with nested objects and arrays
	schema := ValidationSchema{
		Type: "object",
		Properties: map[string]SchemaField{
			"user": {
				Type: "object",
				Required: true,
			},
			"settings": {
				Type: "object",
			},
			"tags": {
				Type: "array",
			},
		},
		Required: []string{"user"},
	}
	
	// Complex data structure
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "John Doe",
			"age":  30,
			"email": "john@example.com",
		},
		"settings": map[string]interface{}{
			"theme": "dark",
			"notifications": true,
		},
		"tags": []interface{}{
			"important",
			"urgent",
			"review",
		},
	}
	
	req := ValidationRequest{
		Data:   data,
		Schema: schema,
		Options: ValidationOptions{
			EnableMetrics: true,
		},
	}
	
	response := validator.ValidateData(req)
	
	if !response.Valid {
		t.Errorf("Expected valid data, got errors: %v", response.Errors)
	}
	
	// Check that metrics are calculated
	if response.Metrics.TotalFields == 0 {
		t.Error("Expected non-zero total fields")
	}
	
	if response.Metrics.ProcessingTime <= 0 {
		t.Error("Expected positive processing time")
	}
} 