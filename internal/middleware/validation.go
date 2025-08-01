package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/pavilion-trust/core-broker/internal/models"
)

// ValidationMiddleware validates request payloads
func ValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only validate POST requests with JSON content
		if r.Method == "POST" && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			// Parse and validate the request
			var req models.VerificationRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeValidationError(w, "INVALID_JSON", "Failed to parse request body", getRequestID(r.Context()))
				return
			}

			// Validate the request
			if err := req.Validate(); err != nil {
				writeValidationError(w, "VALIDATION_FAILED", err.Error(), getRequestID(r.Context()))
				return
			}

			// Store the validated request in context for downstream handlers
			ctx := r.Context()
			ctx = context.WithValue(ctx, "validated_request", &req)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// writeValidationError writes a structured validation error response
func writeValidationError(w http.ResponseWriter, code, message, requestID string) {
	errorResponse := models.NewErrorResponse(code, message, requestID)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	
	if data, err := errorResponse.ToJSON(); err == nil {
		w.Write(data)
	} else {
		// Fallback error response
		w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"Failed to format error response"}}`))
	}
}

// getValidatedRequest retrieves the validated request from context
func getValidatedRequest(ctx context.Context) *models.VerificationRequest {
	if req, ok := ctx.Value("validated_request").(*models.VerificationRequest); ok {
		return req
	}
	return nil
}

// getRequestID extracts request ID from context
func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return "unknown"
}

// ValidationErrorHandler handles validation errors from the validator package
func ValidationErrorHandler(err error) *models.ValidationErrors {
	validationErrors := &models.ValidationErrors{}
	
	if validatorErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validatorErrors {
			field := fieldError.Field()
			tag := fieldError.Tag()
			value := fieldError.Value()
			
			message := getValidationMessage(field, tag, value)
			validationErrors.AddError(field, message, fmt.Sprintf("%v", value))
		}
	}
	
	return validationErrors
}

// getValidationMessage returns a user-friendly validation message
func getValidationMessage(field, tag string, value interface{}) string {
	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, value)
	case "max":
		return fmt.Sprintf("%s must be no more than %s characters", field, value)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, value)
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
} 