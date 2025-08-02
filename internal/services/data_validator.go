package services

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// DataValidator provides comprehensive data validation capabilities
type DataValidator struct {
	config DataValidatorConfig
}

// DataValidatorConfig holds configuration for the data validator
type DataValidatorConfig struct {
	StrictMode     bool
	MaxErrors      int
	EnableMetrics  bool
	CustomValidators map[string]DataValidationRule
}

// DataValidationRule defines a custom validation rule
type DataValidationRule struct {
	Name     string
	Function func(interface{}) (bool, string)
	Message  string
}

// ValidationSchema defines the structure for data validation
type ValidationSchema struct {
	Type       string                 `json:"type"`
	Required   []string              `json:"required,omitempty"`
	Properties map[string]SchemaField `json:"properties,omitempty"`
	MinItems   *int                  `json:"minItems,omitempty"`
	MaxItems   *int                  `json:"maxItems,omitempty"`
	Pattern    *string               `json:"pattern,omitempty"`
	Format     *string               `json:"format,omitempty"`
}

// SchemaField defines validation rules for a specific field
type SchemaField struct {
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required,omitempty"`
	MinLength   *int        `json:"minLength,omitempty"`
	MaxLength   *int        `json:"maxLength,omitempty"`
	Pattern     *string     `json:"pattern,omitempty"`
	MinValue    *float64    `json:"minValue,omitempty"`
	MaxValue    *float64    `json:"maxValue,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`
	Format      *string     `json:"format,omitempty"`
	CustomRule  *string     `json:"customRule,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// ValidationRequest represents a validation request
type ValidationRequest struct {
	Data   interface{}       `json:"data"`
	Schema ValidationSchema  `json:"schema"`
	Options ValidationOptions `json:"options,omitempty"`
}

// ValidationOptions provides additional validation options
type ValidationOptions struct {
	StrictMode     bool                        `json:"strictMode,omitempty"`
	MaxErrors      int                         `json:"maxErrors,omitempty"`
	EnableMetrics  bool                        `json:"enableMetrics,omitempty"`
	CustomRules    map[string]DataValidationRule `json:"customRules,omitempty"`
	SkipFields     []string                    `json:"skipFields,omitempty"`
}

// ValidationResponse represents the result of a validation operation
type ValidationResponse struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
	Metrics  ValidationMetrics `json:"metrics,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
	Value   interface{} `json:"value,omitempty"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
	Value   interface{} `json:"value,omitempty"`
}

// ValidationMetrics provides metrics about the validation process
type ValidationMetrics struct {
	TotalFields    int     `json:"totalFields"`
	ValidFields    int     `json:"validFields"`
	ErrorFields    int     `json:"errorFields"`
	WarningFields  int     `json:"warningFields"`
	ProcessingTime float64 `json:"processingTimeMs"`
	QualityScore   float64 `json:"qualityScore"`
}

// NewDataValidator creates a new data validator instance
func NewDataValidator(config DataValidatorConfig) *DataValidator {
	if config.MaxErrors == 0 {
		config.MaxErrors = 100
	}
	return &DataValidator{
		config: config,
	}
}

// ValidateData performs comprehensive data validation
func (dv *DataValidator) ValidateData(req ValidationRequest) ValidationResponse {
	startTime := time.Now()
	
	response := ValidationResponse{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
		Metrics: ValidationMetrics{
			TotalFields: 0,
			ValidFields: 0,
			ErrorFields: 0,
			WarningFields: 0,
		},
	}

	// Merge options with default config
	options := req.Options
	if options.MaxErrors == 0 {
		options.MaxErrors = dv.config.MaxErrors
	}
	if !options.StrictMode {
		options.StrictMode = dv.config.StrictMode
	}
	
	// Merge custom rules
	if options.CustomRules == nil {
		options.CustomRules = make(map[string]DataValidationRule)
	}
	for name, rule := range dv.config.CustomValidators {
		if _, exists := options.CustomRules[name]; !exists {
			options.CustomRules[name] = rule
		}
	}

	// Validate schema structure
	if err := dv.validateSchema(req.Schema); err != nil {
		response.Errors = append(response.Errors, ValidationError{
			Field:   "schema",
			Message: err.Error(),
			Code:    "INVALID_SCHEMA",
		})
		response.Valid = false
	}

	// Validate data against schema
	dv.validateDataAgainstSchema(req.Data, req.Schema, "", &response, options)

	// Calculate metrics
	processingTime := time.Since(startTime)
	response.Metrics.ProcessingTime = float64(processingTime.Microseconds()) / 1000.0
	if response.Metrics.ProcessingTime <= 0 {
		response.Metrics.ProcessingTime = 0.001 // Minimum value for display
	}
	if response.Metrics.TotalFields > 0 {
		response.Metrics.QualityScore = float64(response.Metrics.ValidFields) / float64(response.Metrics.TotalFields) * 100.0
	}

	// Determine overall validity
	if len(response.Errors) > 0 {
		response.Valid = false
	}
	
	// Limit errors to max errors
	if len(response.Errors) > options.MaxErrors {
		response.Errors = response.Errors[:options.MaxErrors]
	}

	return response
}

// validateSchema validates the schema structure itself
func (dv *DataValidator) validateSchema(schema ValidationSchema) error {
	if schema.Type == "" {
		return fmt.Errorf("schema type is required")
	}

	// Validate required fields are present in properties
	if len(schema.Required) > 0 && len(schema.Properties) > 0 {
		for _, required := range schema.Required {
			if _, exists := schema.Properties[required]; !exists {
				return fmt.Errorf("required field '%s' not found in properties", required)
			}
		}
	}

	return nil
}

// validateDataAgainstSchema recursively validates data against the schema
func (dv *DataValidator) validateDataAgainstSchema(data interface{}, schema ValidationSchema, path string, response *ValidationResponse, options ValidationOptions) {
	dv.validateType(data, schema.Type, path, response, options)
	
	if schema.Type == "object" {
		dv.validateObject(data, schema, path, response, options)
	} else if schema.Type == "array" {
		dv.validateArray(data, schema, path, response, options)
	}
}

// validateType checks if the data matches the expected type
func (dv *DataValidator) validateType(data interface{}, expectedType string, path string, response *ValidationResponse, options ValidationOptions) {
	// Only count fields at the leaf level (not objects/arrays)
	if path != "" && !strings.HasSuffix(path, "]") {
		response.Metrics.TotalFields++
	}
	
	actualType := reflect.TypeOf(data).Kind()
	
	switch expectedType {
	case "string":
		if actualType != reflect.String {
			response.Errors = append(response.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("expected string, got %v", actualType),
				Code:    "TYPE_MISMATCH",
				Value:   data,
			})
			response.Metrics.ErrorFields++
			return
		}
		dv.validateString(data.(string), path, response, options)
		
	case "number":
		if actualType != reflect.Float64 && actualType != reflect.Int && actualType != reflect.Int64 {
			response.Errors = append(response.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("expected number, got %v", actualType),
				Code:    "TYPE_MISMATCH",
				Value:   data,
			})
			response.Metrics.ErrorFields++
			return
		}
		dv.validateNumber(data, path, response, options)
		
	case "integer":
		if actualType != reflect.Int && actualType != reflect.Int64 {
			response.Errors = append(response.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("expected integer, got %v", actualType),
				Code:    "TYPE_MISMATCH",
				Value:   data,
			})
			response.Metrics.ErrorFields++
			return
		}
		dv.validateInteger(data, path, response, options)
		
	case "boolean":
		if actualType != reflect.Bool {
			response.Errors = append(response.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("expected boolean, got %v", actualType),
				Code:    "TYPE_MISMATCH",
				Value:   data,
			})
			response.Metrics.ErrorFields++
			return
		}
		
	case "object":
		if actualType != reflect.Map {
			response.Errors = append(response.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("expected object, got %v", actualType),
				Code:    "TYPE_MISMATCH",
				Value:   data,
			})
			response.Metrics.ErrorFields++
			return
		}
		
	case "array":
		if actualType != reflect.Slice {
			response.Errors = append(response.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("expected array, got %v", actualType),
				Code:    "TYPE_MISMATCH",
				Value:   data,
			})
			response.Metrics.ErrorFields++
			return
		}
		
	case "null":
		if data != nil {
			response.Errors = append(response.Errors, ValidationError{
				Field:   path,
				Message: "expected null value",
				Code:    "TYPE_MISMATCH",
				Value:   data,
			})
			response.Metrics.ErrorFields++
			return
		}
	}
	
	// Only count valid fields at the leaf level
	if path != "" && !strings.HasSuffix(path, "]") {
		response.Metrics.ValidFields++
	}
}

// validateString validates string-specific constraints
func (dv *DataValidator) validateString(value string, path string, response *ValidationResponse, options ValidationOptions) {
	// TODO: Re-enable for production - schema field validation
	// This is commented out for MVP to avoid linter errors
	// Will be re-enabled for production with proper schema field access
	/*
	if schema.MinLength != nil && len(value) < *schema.MinLength {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("string length %d is less than minimum %d", len(value), *schema.MinLength),
			Code:    "MIN_LENGTH_VIOLATION",
			Value:   value,
		})
		response.Metrics.ErrorFields++
	}
	
	if schema.MaxLength != nil && len(value) > *schema.MaxLength {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("string length %d exceeds maximum %d", len(value), *schema.MaxLength),
			Code:    "MAX_LENGTH_VIOLATION",
			Value:   value,
		})
		response.Metrics.ErrorFields++
	}
	
	if schema.Pattern != nil {
		matched, err := regexp.MatchString(*schema.Pattern, value)
		if err != nil {
			response.Errors = append(response.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("invalid regex pattern: %v", err),
				Code:    "INVALID_PATTERN",
				Value:   value,
			})
			response.Metrics.ErrorFields++
		} else if !matched {
			response.Errors = append(response.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("value does not match pattern: %s", *schema.Pattern),
				Code:    "PATTERN_MISMATCH",
				Value:   value,
			})
			response.Metrics.ErrorFields++
		}
	}
	
	if schema.Enum != nil {
		found := false
		for _, enumValue := range schema.Enum {
			if enumValue == value {
				found = true
				break
			}
		}
		if !found {
			response.Errors = append(response.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("value not in allowed enum values"),
				Code:    "ENUM_VIOLATION",
				Value:   value,
			})
			response.Metrics.ErrorFields++
		}
	}
	
	if schema.Format != nil {
		if err := dv.validateFormat(value, *schema.Format); err != nil {
			response.Errors = append(response.Errors, ValidationError{
				Field:   path,
				Message: err.Error(),
				Code:    "FORMAT_VIOLATION",
				Value:   value,
			})
			response.Metrics.ErrorFields++
		}
	}
	
	if schema.CustomRule != nil {
		if rule, exists := options.CustomRules[*schema.CustomRule]; exists {
			if valid, message := rule.Function(value); !valid {
				response.Errors = append(response.Errors, ValidationError{
					Field:   path,
					Message: message,
					Code:    "CUSTOM_RULE_VIOLATION",
					Value:   value,
				})
				response.Metrics.ErrorFields++
			}
		}
	}
	*/
}

// validateNumber validates number-specific constraints
func (dv *DataValidator) validateNumber(value interface{}, path string, response *ValidationResponse, options ValidationOptions) {
	// TODO: Re-enable for production - schema field validation
	// This is commented out for MVP to avoid linter errors
	// Will be re-enabled for production with proper schema field access
	/*
	var numValue float64
	switch v := value.(type) {
	case float64:
		numValue = v
	case int:
		numValue = float64(v)
	case int64:
		numValue = float64(v)
	default:
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: "invalid number type",
			Code:    "INVALID_NUMBER",
			Value:   value,
		})
		response.Metrics.ErrorFields++
		return
	}
	
	if schema.MinValue != nil && numValue < *schema.MinValue {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("value %f is less than minimum %f", numValue, *schema.MinValue),
			Code:    "MIN_VALUE_VIOLATION",
			Value:   value,
		})
		response.Metrics.ErrorFields++
	}
	
	if schema.MaxValue != nil && numValue > *schema.MaxValue {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("value %f exceeds maximum %f", numValue, *schema.MaxValue),
			Code:    "MAX_VALUE_VIOLATION",
			Value:   value,
		})
		response.Metrics.ErrorFields++
	}
	*/
}

// validateInteger validates integer-specific constraints
func (dv *DataValidator) validateInteger(value interface{}, path string, response *ValidationResponse, options ValidationOptions) {
	// TODO: Re-enable for production - schema field validation
	// This is commented out for MVP to avoid linter errors
	// Will be re-enabled for production with proper schema field access
	/*
	var intValue int64
	switch v := value.(type) {
	case int:
		intValue = int64(v)
	case int64:
		intValue = v
	default:
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: "invalid integer type",
			Code:    "INVALID_INTEGER",
			Value:   value,
		})
		response.Metrics.ErrorFields++
		return
	}
	
	if schema.MinValue != nil && float64(intValue) < *schema.MinValue {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("value %d is less than minimum %f", intValue, *schema.MinValue),
			Code:    "MIN_VALUE_VIOLATION",
			Value:   value,
		})
		response.Metrics.ErrorFields++
	}
	
	if schema.MaxValue != nil && float64(intValue) > *schema.MaxValue {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("value %d exceeds maximum %f", intValue, *schema.MaxValue),
			Code:    "MAX_VALUE_VIOLATION",
			Value:   value,
		})
		response.Metrics.ErrorFields++
	}
	*/
}

// validateObject validates object-specific constraints
func (dv *DataValidator) validateObject(data interface{}, schema ValidationSchema, path string, response *ValidationResponse, options ValidationOptions) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: "invalid object type",
			Code:    "INVALID_OBJECT",
			Value:   data,
		})
		response.Metrics.ErrorFields++
		return
	}

	// Check required fields
	for _, required := range schema.Required {
		if _, exists := dataMap[required]; !exists {
			response.Errors = append(response.Errors, ValidationError{
				Field:   fmt.Sprintf("%s.%s", path, required),
				Message: "required field is missing",
				Code:    "REQUIRED_FIELD_MISSING",
			})
			response.Metrics.ErrorFields++
		}
	}

	// Validate each field
	for fieldName, fieldValue := range dataMap {
		fieldPath := fieldName
		if path != "" {
			fieldPath = fmt.Sprintf("%s.%s", path, fieldName)
		}

		// Skip fields if specified
		if dv.containsString(options.SkipFields, fieldName) {
			continue
		}

		if fieldSchema, exists := schema.Properties[fieldName]; exists {
			dv.validateField(fieldValue, fieldSchema, fieldPath, response, options)
		} else if options.StrictMode {
			response.Warnings = append(response.Warnings, ValidationWarning{
				Field:   fieldPath,
				Message: "unknown field",
				Code:    "UNKNOWN_FIELD",
				Value:   fieldValue,
			})
			response.Metrics.WarningFields++
		}
	}

	// Check array constraints
	if schema.MinItems != nil && len(dataMap) < *schema.MinItems {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("object has %d items, minimum is %d", len(dataMap), *schema.MinItems),
			Code:    "MIN_ITEMS_VIOLATION",
			Value:   data,
		})
		response.Metrics.ErrorFields++
	}

	if schema.MaxItems != nil && len(dataMap) > *schema.MaxItems {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("object has %d items, maximum is %d", len(dataMap), *schema.MaxItems),
			Code:    "MAX_ITEMS_VIOLATION",
			Value:   data,
		})
		response.Metrics.ErrorFields++
	}
}

// validateArray validates array-specific constraints
func (dv *DataValidator) validateArray(data interface{}, schema ValidationSchema, path string, response *ValidationResponse, options ValidationOptions) {
	dataSlice, ok := data.([]interface{})
	if !ok {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: "invalid array type",
			Code:    "INVALID_ARRAY",
			Value:   data,
		})
		response.Metrics.ErrorFields++
		return
	}

	// Check array constraints
	if schema.MinItems != nil && len(dataSlice) < *schema.MinItems {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("array has %d items, minimum is %d", len(dataSlice), *schema.MinItems),
			Code:    "MIN_ITEMS_VIOLATION",
			Value:   data,
		})
		response.Metrics.ErrorFields++
	}

	if schema.MaxItems != nil && len(dataSlice) > *schema.MaxItems {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("array has %d items, maximum is %d", len(dataSlice), *schema.MaxItems),
			Code:    "MAX_ITEMS_VIOLATION",
			Value:   data,
		})
		response.Metrics.ErrorFields++
	}

	// Validate each array item with a simple schema
	itemSchema := ValidationSchema{
		Type: "string", // Default to string for array items
	}
	for i, item := range dataSlice {
		itemPath := fmt.Sprintf("%s[%d]", path, i)
		dv.validateDataAgainstSchema(item, itemSchema, itemPath, response, options)
	}
}

// validateField validates a specific field against its schema
func (dv *DataValidator) validateField(value interface{}, fieldSchema SchemaField, path string, response *ValidationResponse, options ValidationOptions) {
	// Check if field is required
	if fieldSchema.Required && value == nil {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: "required field is missing",
			Code:    "REQUIRED_FIELD_MISSING",
		})
		response.Metrics.ErrorFields++
		return
	}

	// Apply default value if field is nil
	if value == nil && fieldSchema.Default != nil {
		value = fieldSchema.Default
	}

	// Validate type
	dv.validateType(value, fieldSchema.Type, path, response, options)

	// Validate string-specific constraints
	if fieldSchema.Type == "string" && value != nil {
		if strValue, ok := value.(string); ok {
			dv.validateStringConstraints(strValue, fieldSchema, path, response, options)
		}
	}

	// Validate number-specific constraints
	if fieldSchema.Type == "number" && value != nil {
		dv.validateNumberConstraints(value, fieldSchema, path, response, options)
	}

	// Validate integer-specific constraints
	if fieldSchema.Type == "integer" && value != nil {
		dv.validateIntegerConstraints(value, fieldSchema, path, response, options)
	}
}

// validateStringConstraints validates string field constraints
func (dv *DataValidator) validateStringConstraints(value string, fieldSchema SchemaField, path string, response *ValidationResponse, options ValidationOptions) {
	if fieldSchema.MinLength != nil && len(value) < *fieldSchema.MinLength {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("string length %d is less than minimum %d", len(value), *fieldSchema.MinLength),
			Code:    "MIN_LENGTH_VIOLATION",
			Value:   value,
		})
		response.Metrics.ErrorFields++
	}

	if fieldSchema.MaxLength != nil && len(value) > *fieldSchema.MaxLength {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("string length %d exceeds maximum %d", len(value), *fieldSchema.MaxLength),
			Code:    "MAX_LENGTH_VIOLATION",
			Value:   value,
		})
		response.Metrics.ErrorFields++
	}

	if fieldSchema.Pattern != nil {
		matched, err := regexp.MatchString(*fieldSchema.Pattern, value)
		if err != nil {
			response.Errors = append(response.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("invalid regex pattern: %v", err),
				Code:    "INVALID_PATTERN",
				Value:   value,
			})
			response.Metrics.ErrorFields++
		} else if !matched {
			response.Errors = append(response.Errors, ValidationError{
				Field:   path,
				Message: fmt.Sprintf("value does not match pattern: %s", *fieldSchema.Pattern),
				Code:    "PATTERN_MISMATCH",
				Value:   value,
			})
			response.Metrics.ErrorFields++
		}
	}

	if fieldSchema.Enum != nil {
		found := false
		for _, enumValue := range fieldSchema.Enum {
			if enumValue == value {
				found = true
				break
			}
		}
		if !found {
			response.Errors = append(response.Errors, ValidationError{
				Field:   path,
				Message: "value not in allowed enum values",
				Code:    "ENUM_VIOLATION",
				Value:   value,
			})
			response.Metrics.ErrorFields++
		}
	}

	if fieldSchema.Format != nil {
		if err := dv.validateFormat(value, *fieldSchema.Format); err != nil {
			response.Errors = append(response.Errors, ValidationError{
				Field:   path,
				Message: err.Error(),
				Code:    "FORMAT_VIOLATION",
				Value:   value,
			})
			response.Metrics.ErrorFields++
		}
	}

	if fieldSchema.CustomRule != nil {
		if rule, exists := options.CustomRules[*fieldSchema.CustomRule]; exists {
			if valid, message := rule.Function(value); !valid {
				response.Errors = append(response.Errors, ValidationError{
					Field:   path,
					Message: message,
					Code:    "CUSTOM_RULE_VIOLATION",
					Value:   value,
				})
				response.Metrics.ErrorFields++
			}
		}
	}
}

// validateNumberConstraints validates number field constraints
func (dv *DataValidator) validateNumberConstraints(value interface{}, fieldSchema SchemaField, path string, response *ValidationResponse, options ValidationOptions) {
	var numValue float64
	switch v := value.(type) {
	case float64:
		numValue = v
	case int:
		numValue = float64(v)
	case int64:
		numValue = float64(v)
	default:
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: "invalid number type",
			Code:    "INVALID_NUMBER",
			Value:   value,
		})
		response.Metrics.ErrorFields++
		return
	}

	if fieldSchema.MinValue != nil && numValue < *fieldSchema.MinValue {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("value %f is less than minimum %f", numValue, *fieldSchema.MinValue),
			Code:    "MIN_VALUE_VIOLATION",
			Value:   value,
		})
		response.Metrics.ErrorFields++
	}

	if fieldSchema.MaxValue != nil && numValue > *fieldSchema.MaxValue {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("value %f exceeds maximum %f", numValue, *fieldSchema.MaxValue),
			Code:    "MAX_VALUE_VIOLATION",
			Value:   value,
		})
		response.Metrics.ErrorFields++
	}

	// Validate custom rule
	if fieldSchema.CustomRule != nil {
		if rule, exists := options.CustomRules[*fieldSchema.CustomRule]; exists {
			if valid, message := rule.Function(value); !valid {
				response.Errors = append(response.Errors, ValidationError{
					Field:   path,
					Message: message,
					Code:    "CUSTOM_RULE_VIOLATION",
					Value:   value,
				})
				response.Metrics.ErrorFields++
			}
		}
	}
}

// validateIntegerConstraints validates integer field constraints
func (dv *DataValidator) validateIntegerConstraints(value interface{}, fieldSchema SchemaField, path string, response *ValidationResponse, options ValidationOptions) {
	var intValue int64
	switch v := value.(type) {
	case int:
		intValue = int64(v)
	case int64:
		intValue = v
	default:
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: "invalid integer type",
			Code:    "INVALID_INTEGER",
			Value:   value,
		})
		response.Metrics.ErrorFields++
		return
	}

	if fieldSchema.MinValue != nil && float64(intValue) < *fieldSchema.MinValue {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("value %d is less than minimum %f", intValue, *fieldSchema.MinValue),
			Code:    "MIN_VALUE_VIOLATION",
			Value:   value,
		})
		response.Metrics.ErrorFields++
	}

	if fieldSchema.MaxValue != nil && float64(intValue) > *fieldSchema.MaxValue {
		response.Errors = append(response.Errors, ValidationError{
			Field:   path,
			Message: fmt.Sprintf("value %d exceeds maximum %f", intValue, *fieldSchema.MaxValue),
			Code:    "MAX_VALUE_VIOLATION",
			Value:   value,
		})
		response.Metrics.ErrorFields++
	}
}

// validateFormat validates string format
func (dv *DataValidator) validateFormat(value string, format string) error {
	switch format {
	case "email":
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(value) {
			return fmt.Errorf("invalid email format")
		}
	case "uri":
		if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
			return fmt.Errorf("invalid URI format")
		}
	case "date":
		_, err := time.Parse("2006-01-02", value)
		if err != nil {
			return fmt.Errorf("invalid date format (expected YYYY-MM-DD)")
		}
	case "date-time":
		_, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return fmt.Errorf("invalid date-time format (expected RFC3339)")
		}
	case "uuid":
		uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
		if !uuidRegex.MatchString(strings.ToLower(value)) {
			return fmt.Errorf("invalid UUID format")
		}
	}
	return nil
}

// validateCustomRule validates using a custom rule
func (dv *DataValidator) validateCustomRule(value interface{}, rule DataValidationRule) (bool, string) {
	return rule.Function(value)
}

// containsString checks if a slice contains a string
func (dv *DataValidator) containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetValidationMetrics returns validation metrics
func (dv *DataValidator) GetValidationMetrics() ValidationMetrics {
	return ValidationMetrics{}
} 