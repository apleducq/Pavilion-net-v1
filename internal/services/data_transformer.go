package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DataTransformer provides comprehensive data transformation capabilities
type DataTransformer struct {
	config DataTransformerConfig
}

// DataTransformerConfig holds configuration for the data transformer
type DataTransformerConfig struct {
	DefaultFormat     string
	EnableEnrichment  bool
	MissingDataPolicy MissingDataPolicy
	CustomTransformers map[string]TransformFunction
}

// MissingDataPolicy defines how to handle missing data
type MissingDataPolicy struct {
	Strategy MissingDataStrategy
	DefaultValue interface{}
	SkipFields   []string
}

// MissingDataStrategy defines the strategy for handling missing data
type MissingDataStrategy string

const (
	MissingDataSkip     MissingDataStrategy = "skip"
	MissingDataDefault  MissingDataStrategy = "default"
	MissingDataNull     MissingDataStrategy = "null"
	MissingDataError    MissingDataStrategy = "error"
)

// TransformFunction defines a custom transformation function
type TransformFunction struct {
	Name     string
	Function func(interface{}) (interface{}, error)
	Description string
}

// TransformationRequest represents a transformation request
type TransformationRequest struct {
	Data           interface{}           `json:"data"`
	SourceSchema   TransformationSchema  `json:"sourceSchema"`
	TargetSchema   TransformationSchema  `json:"targetSchema"`
	Transformations []TransformationRule `json:"transformations,omitempty"`
	Options        TransformationOptions `json:"options,omitempty"`
}

// TransformationSchema defines the structure for data transformation
type TransformationSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]SchemaField    `json:"properties,omitempty"`
	Required   []string                  `json:"required,omitempty"`
	Format     string                    `json:"format,omitempty"`
	Encoding   string                    `json:"encoding,omitempty"`
}

// TransformationRule defines a transformation rule
type TransformationRule struct {
	SourceField      string                 `json:"sourceField"`
	TargetField      string                 `json:"targetField"`
	Transformation   string                 `json:"transformation"`
	Parameters       map[string]interface{} `json:"parameters,omitempty"`
	Condition        string                 `json:"condition,omitempty"`
	DefaultValue     interface{}            `json:"defaultValue,omitempty"`
	Required         bool                   `json:"required,omitempty"`
}

// TransformationOptions provides additional transformation options
type TransformationOptions struct {
	EnableEnrichment  bool                        `json:"enableEnrichment,omitempty"`
	MissingDataPolicy MissingDataPolicy           `json:"missingDataPolicy,omitempty"`
	CustomTransformers map[string]TransformFunction `json:"customTransformers,omitempty"`
	SkipFields        []string                    `json:"skipFields,omitempty"`
	ValidateOutput    bool                        `json:"validateOutput,omitempty"`
}

// TransformationResponse represents the result of a transformation operation
type TransformationResponse struct {
	Success     bool                   `json:"success"`
	Data        interface{}            `json:"data"`
	Errors      []TransformationError  `json:"errors,omitempty"`
	Warnings    []TransformationWarning `json:"warnings,omitempty"`
	Metrics     TransformationMetrics  `json:"metrics,omitempty"`
}

// TransformationError represents a transformation error
type TransformationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
	Value   interface{} `json:"value,omitempty"`
}

// TransformationWarning represents a transformation warning
type TransformationWarning struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
	Value   interface{} `json:"value,omitempty"`
}

// TransformationMetrics provides metrics about the transformation process
type TransformationMetrics struct {
	TotalFields     int     `json:"totalFields"`
	TransformedFields int   `json:"transformedFields"`
	ErrorFields     int     `json:"errorFields"`
	WarningFields   int     `json:"warningFields"`
	ProcessingTime  float64 `json:"processingTimeMs"`
	EnrichmentCount int     `json:"enrichmentCount"`
}

// NewDataTransformer creates a new data transformer instance
func NewDataTransformer(config DataTransformerConfig) *DataTransformer {
	if config.DefaultFormat == "" {
		config.DefaultFormat = "json"
	}
	return &DataTransformer{
		config: config,
	}
}

// TransformData performs comprehensive data transformation
func (dt *DataTransformer) TransformData(req TransformationRequest) TransformationResponse {
	startTime := time.Now()
	
	response := TransformationResponse{
		Success:  true,
		Data:     nil,
		Errors:   make([]TransformationError, 0),
		Warnings: make([]TransformationWarning, 0),
		Metrics: TransformationMetrics{
			TotalFields:      0,
			TransformedFields: 0,
			ErrorFields:      0,
			WarningFields:    0,
			EnrichmentCount:  0,
		},
	}

	// Merge options with default config
	options := req.Options
	if options.MissingDataPolicy.Strategy == "" {
		options.MissingDataPolicy = dt.config.MissingDataPolicy
	}
	if !options.EnableEnrichment {
		options.EnableEnrichment = dt.config.EnableEnrichment
	}

	// Merge custom transformers
	if options.CustomTransformers == nil {
		options.CustomTransformers = make(map[string]TransformFunction)
	}
	for name, transformer := range dt.config.CustomTransformers {
		if _, exists := options.CustomTransformers[name]; !exists {
			options.CustomTransformers[name] = transformer
		}
	}

	// Transform data according to rules
	transformedData, err := dt.applyTransformations(req.Data, req.Transformations, req.TargetSchema, &response, options)
	if err != nil {
		response.Errors = append(response.Errors, TransformationError{
			Field:   "transformation",
			Message: err.Error(),
			Code:    "TRANSFORMATION_FAILED",
		})
		response.Success = false
	}

	// Apply schema transformation if target schema is provided
	if req.TargetSchema.Type != "" {
		transformedData, err = dt.applySchemaTransformation(transformedData, req.TargetSchema, &response, options)
		if err != nil {
			response.Errors = append(response.Errors, TransformationError{
				Field:   "schema",
				Message: err.Error(),
				Code:    "SCHEMA_TRANSFORMATION_FAILED",
			})
			response.Success = false
		}
	}

	// Handle missing data
	transformedData = dt.handleMissingData(transformedData, req.TargetSchema, &response, options)

	// Enrich data if enabled
	if options.EnableEnrichment {
		transformedData = dt.enrichData(transformedData, &response, options)
	}

	response.Data = transformedData

	// Calculate metrics
	processingTime := time.Since(startTime)
	response.Metrics.ProcessingTime = float64(processingTime.Microseconds()) / 1000.0
	if response.Metrics.ProcessingTime <= 0 {
		response.Metrics.ProcessingTime = 0.001
	}

	// Determine overall success
	if len(response.Errors) > 0 {
		response.Success = false
	}

	return response
}

// applyTransformations applies transformation rules to the data
func (dt *DataTransformer) applyTransformations(data interface{}, rules []TransformationRule, targetSchema TransformationSchema, response *TransformationResponse, options TransformationOptions) (interface{}, error) {
	if len(rules) == 0 {
		return data, nil
	}

	result := make(map[string]interface{})
	
	// Convert input data to map if it's not already
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("input data must be an object for transformation")
	}

	for _, rule := range rules {
		response.Metrics.TotalFields++
		
		// Get source value
		sourceValue, exists := dataMap[rule.SourceField]
		if !exists {
			// Handle missing source field
			switch options.MissingDataPolicy.Strategy {
			case MissingDataSkip:
				continue
			case MissingDataDefault:
				sourceValue = rule.DefaultValue
			case MissingDataNull:
				sourceValue = nil
			case MissingDataError:
				response.Errors = append(response.Errors, TransformationError{
					Field:   rule.SourceField,
					Message: "source field is missing",
					Code:    "MISSING_SOURCE_FIELD",
				})
				response.Metrics.ErrorFields++
				continue
			}
		}

		// Apply transformation
		transformedValue, err := dt.applyTransformation(sourceValue, rule, options)
		if err != nil {
			response.Errors = append(response.Errors, TransformationError{
				Field:   rule.SourceField,
				Message: err.Error(),
				Code:    "TRANSFORMATION_ERROR",
				Value:   sourceValue,
			})
			response.Metrics.ErrorFields++
			continue
		}

		// Set target field
		targetField := rule.TargetField
		if targetField == "" {
			targetField = rule.SourceField
		}

		result[targetField] = transformedValue
		response.Metrics.TransformedFields++
	}

	return result, nil
}

// applyTransformation applies a single transformation rule
func (dt *DataTransformer) applyTransformation(value interface{}, rule TransformationRule, options TransformationOptions) (interface{}, error) {
	switch rule.Transformation {
	case "copy":
		return value, nil
	case "string":
		return dt.toString(value)
	case "number":
		return dt.toNumber(value)
	case "integer":
		return dt.toInteger(value)
	case "boolean":
		return dt.toBoolean(value)
	case "uppercase":
		return dt.toUppercase(value)
	case "lowercase":
		return dt.toLowercase(value)
	case "trim":
		return dt.trim(value)
	case "format_date":
		return dt.formatDate(value, rule.Parameters)
	case "parse_date":
		return dt.parseDate(value, rule.Parameters)
	case "concat":
		return dt.concat(value, rule.Parameters)
	case "split":
		return dt.split(value, rule.Parameters)
	case "replace":
		return dt.replace(value, rule.Parameters)
	case "default":
		return dt.applyDefault(value, rule.DefaultValue)
	case "custom":
		return dt.applyCustomTransformation(value, rule, options)
	default:
		return nil, fmt.Errorf("unknown transformation: %s", rule.Transformation)
	}
}

// toString converts value to string
func (dt *DataTransformer) toString(value interface{}) (interface{}, error) {
	if value == nil {
		return "", nil
	}
	return fmt.Sprintf("%v", value), nil
}

// toNumber converts value to number
func (dt *DataTransformer) toNumber(value interface{}) (interface{}, error) {
	if value == nil {
		return 0.0, nil
	}
	
	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, nil
		}
		return 0.0, fmt.Errorf("cannot convert string to number: %s", v)
	default:
		return 0.0, fmt.Errorf("cannot convert type %T to number", value)
	}
}

// toInteger converts value to integer
func (dt *DataTransformer) toInteger(value interface{}) (interface{}, error) {
	if value == nil {
		return 0, nil
	}
	
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i, nil
		}
		return 0, fmt.Errorf("cannot convert string to integer: %s", v)
	default:
		return 0, fmt.Errorf("cannot convert type %T to integer", value)
	}
}

// toBoolean converts value to boolean
func (dt *DataTransformer) toBoolean(value interface{}) (interface{}, error) {
	if value == nil {
		return false, nil
	}
	
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		switch strings.ToLower(v) {
		case "true", "1", "yes", "on":
			return true, nil
		case "false", "0", "no", "off":
			return false, nil
		default:
			return false, fmt.Errorf("cannot convert string to boolean: %s", v)
		}
	case int, int64, float64:
		return v != 0, nil
	default:
		return false, fmt.Errorf("cannot convert type %T to boolean", value)
	}
}

// toUppercase converts string to uppercase
func (dt *DataTransformer) toUppercase(value interface{}) (interface{}, error) {
	if value == nil {
		return "", nil
	}
	
	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value must be string for uppercase transformation")
	}
	
	return strings.ToUpper(str), nil
}

// toLowercase converts string to lowercase
func (dt *DataTransformer) toLowercase(value interface{}) (interface{}, error) {
	if value == nil {
		return "", nil
	}
	
	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value must be string for lowercase transformation")
	}
	
	return strings.ToLower(str), nil
}

// trim trims whitespace from string
func (dt *DataTransformer) trim(value interface{}) (interface{}, error) {
	if value == nil {
		return "", nil
	}
	
	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value must be string for trim transformation")
	}
	
	return strings.TrimSpace(str), nil
}

// formatDate formats date according to parameters
func (dt *DataTransformer) formatDate(value interface{}, params map[string]interface{}) (interface{}, error) {
	if value == nil {
		return "", nil
	}
	
	format := "2006-01-02"
	if formatParam, exists := params["format"]; exists {
		if formatStr, ok := formatParam.(string); ok {
			format = formatStr
		}
	}
	
	// Try to parse the input as a date
	var t time.Time
	var err error
	
	switch v := value.(type) {
	case string:
		t, err = time.Parse("2006-01-02", v)
		if err != nil {
			t, err = time.Parse(time.RFC3339, v)
		}
	case int64:
		t = time.Unix(v, 0)
	default:
		return "", fmt.Errorf("cannot convert type %T to date", value)
	}
	
	if err != nil {
		return "", fmt.Errorf("invalid date format: %v", err)
	}
	
	return t.Format(format), nil
}

// parseDate parses date string according to parameters
func (dt *DataTransformer) parseDate(value interface{}, params map[string]interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("value must be string for date parsing")
	}
	
	format := "2006-01-02"
	if formatParam, exists := params["format"]; exists {
		if formatStr, ok := formatParam.(string); ok {
			format = formatStr
		}
	}
	
	t, err := time.Parse(format, str)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %v", err)
	}
	
	return t.Format("2006-01-02"), nil
}

// concat concatenates values according to parameters
func (dt *DataTransformer) concat(value interface{}, params map[string]interface{}) (interface{}, error) {
	if value == nil {
		return "", nil
	}
	
	separator := ""
	if sepParam, exists := params["separator"]; exists {
		if sepStr, ok := sepParam.(string); ok {
			separator = sepStr
		}
	}
	
	switch v := value.(type) {
	case string:
		return v, nil
	case []interface{}:
		parts := make([]string, 0)
		for _, part := range v {
			parts = append(parts, fmt.Sprintf("%v", part))
		}
		return strings.Join(parts, separator), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// split splits string according to parameters
func (dt *DataTransformer) split(value interface{}, params map[string]interface{}) (interface{}, error) {
	if value == nil {
		return []interface{}{}, nil
	}
	
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("value must be string for split transformation")
	}
	
	separator := " "
	if sepParam, exists := params["separator"]; exists {
		if sepStr, ok := sepParam.(string); ok {
			separator = sepStr
		}
	}
	
	parts := strings.Split(str, separator)
	result := make([]interface{}, len(parts))
	for i, part := range parts {
		result[i] = part
	}
	
	return result, nil
}

// replace replaces string according to parameters
func (dt *DataTransformer) replace(value interface{}, params map[string]interface{}) (interface{}, error) {
	if value == nil {
		return "", nil
	}
	
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("value must be string for replace transformation")
	}
	
	oldStr, exists := params["old"]
	if !exists {
		return str, nil
	}
	
	newStr, exists := params["new"]
	if !exists {
		return str, nil
	}
	
	old, ok := oldStr.(string)
	if !ok {
		return str, nil
	}
	
	new, ok := newStr.(string)
	if !ok {
		return str, nil
	}
	
	return strings.ReplaceAll(str, old, new), nil
}

// applyDefault applies default value if value is nil or empty
func (dt *DataTransformer) applyDefault(value interface{}, defaultValue interface{}) (interface{}, error) {
	if value == nil || value == "" {
		return defaultValue, nil
	}
	return value, nil
}

// applyCustomTransformation applies a custom transformation
func (dt *DataTransformer) applyCustomTransformation(value interface{}, rule TransformationRule, options TransformationOptions) (interface{}, error) {
	customName, exists := rule.Parameters["name"]
	if !exists {
		return nil, fmt.Errorf("custom transformation requires 'name' parameter")
	}
	
	name, ok := customName.(string)
	if !ok {
		return nil, fmt.Errorf("custom transformation name must be string")
	}
	
	transformer, exists := options.CustomTransformers[name]
	if !exists {
		return nil, fmt.Errorf("custom transformation '%s' not found", name)
	}
	
	return transformer.Function(value)
}

// applySchemaTransformation applies schema transformation
func (dt *DataTransformer) applySchemaTransformation(data interface{}, schema TransformationSchema, response *TransformationResponse, options TransformationOptions) (interface{}, error) {
	if schema.Type == "object" {
		return dt.transformToObject(data, schema, response, options)
	}
	
	return data, nil
}

// transformToObject transforms data to match object schema
func (dt *DataTransformer) transformToObject(data interface{}, schema TransformationSchema, response *TransformationResponse, options TransformationOptions) (interface{}, error) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data must be an object for schema transformation")
	}
	
	result := make(map[string]interface{})
	
	for fieldName, fieldSchema := range schema.Properties {
		response.Metrics.TotalFields++
		
		value, exists := dataMap[fieldName]
		if !exists {
			// Handle missing field
			switch options.MissingDataPolicy.Strategy {
			case MissingDataSkip:
				continue
			case MissingDataDefault:
				value = fieldSchema.Default
			case MissingDataNull:
				value = nil
			case MissingDataError:
				response.Errors = append(response.Errors, TransformationError{
					Field:   fieldName,
					Message: "required field is missing",
					Code:    "MISSING_REQUIRED_FIELD",
				})
				response.Metrics.ErrorFields++
				continue
			}
		}
		
		// Transform value to match field schema
		transformedValue, err := dt.transformValueToSchema(value, fieldSchema)
		if err != nil {
			response.Errors = append(response.Errors, TransformationError{
				Field:   fieldName,
				Message: err.Error(),
				Code:    "SCHEMA_TRANSFORMATION_ERROR",
				Value:   value,
			})
			response.Metrics.ErrorFields++
			continue
		}
		
		result[fieldName] = transformedValue
		response.Metrics.TransformedFields++
	}
	
	return result, nil
}

// transformValueToSchema transforms a value to match a field schema
func (dt *DataTransformer) transformValueToSchema(value interface{}, fieldSchema SchemaField) (interface{}, error) {
	if value == nil {
		return fieldSchema.Default, nil
	}
	
	switch fieldSchema.Type {
	case "string":
		return dt.toString(value)
	case "number":
		return dt.toNumber(value)
	case "integer":
		return dt.toInteger(value)
	case "boolean":
		return dt.toBoolean(value)
	default:
		return value, nil
	}
}

// handleMissingData handles missing data according to policy
func (dt *DataTransformer) handleMissingData(data interface{}, schema TransformationSchema, response *TransformationResponse, options TransformationOptions) interface{} {
	if data == nil {
		return data
	}
	
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}
	
	result := make(map[string]interface{})
	
	for fieldName, value := range dataMap {
		if value == nil {
			switch options.MissingDataPolicy.Strategy {
			case MissingDataDefault:
				result[fieldName] = options.MissingDataPolicy.DefaultValue
			case MissingDataNull:
				result[fieldName] = nil
			case MissingDataSkip:
				// Skip this field
				continue
			case MissingDataError:
				response.Warnings = append(response.Warnings, TransformationWarning{
					Field:   fieldName,
					Message: "missing data found",
					Code:    "MISSING_DATA",
				})
				response.Metrics.WarningFields++
				result[fieldName] = nil
			}
		} else {
			result[fieldName] = value
		}
	}
	
	return result
}

// enrichData enriches data with additional information
func (dt *DataTransformer) enrichData(data interface{}, response *TransformationResponse, options TransformationOptions) interface{} {
	if data == nil {
		return data
	}
	
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}
	
	// Add timestamp
	dataMap["_transformed_at"] = time.Now().Format(time.RFC3339)
	response.Metrics.EnrichmentCount++
	
	// Add metadata
	dataMap["_metadata"] = map[string]interface{}{
		"transformer_version": "1.0.0",
		"transformation_count": response.Metrics.TransformedFields,
	}
	response.Metrics.EnrichmentCount++
	
	return dataMap
}

// GetTransformationMetrics returns transformation metrics
func (dt *DataTransformer) GetTransformationMetrics() TransformationMetrics {
	return TransformationMetrics{}
} 