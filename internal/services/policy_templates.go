package services

import (
	"context"
	"fmt"
	"time"

	"github.com/pavilion-trust/core-broker/internal/models"
)

// PolicyTemplateService handles policy template operations
type PolicyTemplateService struct {
	storage models.PolicyStorage
}

// NewPolicyTemplateService creates a new policy template service
func NewPolicyTemplateService(storage models.PolicyStorage) *PolicyTemplateService {
	return &PolicyTemplateService{
		storage: storage,
	}
}

// CreateDefaultTemplates creates the default policy templates
func (pts *PolicyTemplateService) CreateDefaultTemplates(ctx context.Context, createdBy string) error {
	templates := []*models.PolicyTemplate{
		pts.createAgeVerificationTemplate(createdBy),
		pts.createStudentStatusTemplate(createdBy),
		pts.createEmploymentVerificationTemplate(createdBy),
		pts.createAddressVerificationTemplate(createdBy),
	}

	for _, template := range templates {
		if err := pts.storage.CreateTemplate(ctx, template); err != nil {
			return fmt.Errorf("failed to create template %s: %w", template.Name, err)
		}
	}

	return nil
}

// createAgeVerificationTemplate creates an age verification template
func (pts *PolicyTemplateService) createAgeVerificationTemplate(createdBy string) *models.PolicyTemplate {
	policy := models.Policy{
		ID:          "age-verification-template",
		Version:     "1.0",
		Name:        "Age Verification",
		Description: "Verify that a person is of a certain age or older",
		Conditions: models.PolicyConditions{
			Operator: "AND",
			Rules: []models.PolicyRule{
				{
					Type:           "credential_required",
					CredentialType: "IdentityCredential",
				},
				{
					Type:           "claim_greater_than",
					Claim:          "age",
					Value:          18,
				},
				{
					Type: "not_expired",
				},
			},
		},
		Privacy: models.PrivacySettings{
			PPRLEnabled:        true,
			SelectiveDisclosure: true,
			AuditLevel:         "minimal",
			RetentionDays:      90,
		},
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
		CreatedBy: createdBy,
		Status:    "active",
	}

	return models.NewPolicyTemplate(
		"Age Verification",
		"Template for verifying minimum age requirements",
		"age_verification",
		createdBy,
		policy,
	)
}

// createStudentStatusTemplate creates a student status verification template
func (pts *PolicyTemplateService) createStudentStatusTemplate(createdBy string) *models.PolicyTemplate {
	policy := models.Policy{
		ID:          "student-status-template",
		Version:     "1.0",
		Name:        "Student Status Verification",
		Description: "Verify that a person is currently enrolled as a student",
		Conditions: models.PolicyConditions{
			Operator: "AND",
			Rules: []models.PolicyRule{
				{
					Type:           "credential_required",
					CredentialType: "StudentCredential",
				},
				{
					Type:           "claim_equals",
					Claim:          "status",
					Value:          "enrolled",
				},
				{
					Type: "not_expired",
				},
			},
		},
		Privacy: models.PrivacySettings{
			PPRLEnabled:        true,
			SelectiveDisclosure: true,
			AuditLevel:         "minimal",
			RetentionDays:      90,
		},
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
		CreatedBy: createdBy,
		Status:    "active",
	}

	return models.NewPolicyTemplate(
		"Student Status Verification",
		"Template for verifying student enrollment status",
		"student_verification",
		createdBy,
		policy,
	)
}

// createEmploymentVerificationTemplate creates an employment verification template
func (pts *PolicyTemplateService) createEmploymentVerificationTemplate(createdBy string) *models.PolicyTemplate {
	policy := models.Policy{
		ID:          "employment-verification-template",
		Version:     "1.0",
		Name:        "Employment Verification",
		Description: "Verify that a person is currently employed",
		Conditions: models.PolicyConditions{
			Operator: "AND",
			Rules: []models.PolicyRule{
				{
					Type:           "credential_required",
					CredentialType: "EmploymentCredential",
				},
				{
					Type:           "claim_equals",
					Claim:          "employment_status",
					Value:          "active",
				},
				{
					Type: "not_expired",
				},
			},
		},
		Privacy: models.PrivacySettings{
			PPRLEnabled:        true,
			SelectiveDisclosure: true,
			AuditLevel:         "minimal",
			RetentionDays:      90,
		},
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
		CreatedBy: createdBy,
		Status:    "active",
	}

	return models.NewPolicyTemplate(
		"Employment Verification",
		"Template for verifying employment status",
		"employment_verification",
		createdBy,
		policy,
	)
}

// createAddressVerificationTemplate creates an address verification template
func (pts *PolicyTemplateService) createAddressVerificationTemplate(createdBy string) *models.PolicyTemplate {
	policy := models.Policy{
		ID:          "address-verification-template",
		Version:     "1.0",
		Name:        "Address Verification",
		Description: "Verify that a person resides at a specific address",
		Conditions: models.PolicyConditions{
			Operator: "AND",
			Rules: []models.PolicyRule{
				{
					Type:           "credential_required",
					CredentialType: "AddressCredential",
				},
				{
					Type: "not_expired",
				},
			},
		},
		Privacy: models.PrivacySettings{
			PPRLEnabled:        true,
			SelectiveDisclosure: true,
			AuditLevel:         "minimal",
			RetentionDays:      90,
		},
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
		CreatedBy: createdBy,
		Status:    "active",
	}

	return models.NewPolicyTemplate(
		"Address Verification",
		"Template for verifying residential address",
		"address_verification",
		createdBy,
		policy,
	)
}

// GetTemplateByCategory retrieves templates by category
func (pts *PolicyTemplateService) GetTemplateByCategory(ctx context.Context, category string) ([]*models.PolicyTemplate, error) {
	filters := map[string]interface{}{
		"category": category,
		"status":   "active",
	}

	return pts.storage.ListTemplates(ctx, filters)
}

// CreatePolicyFromTemplate creates a new policy from a template
func (pts *PolicyTemplateService) CreatePolicyFromTemplate(ctx context.Context, templateID string, customizations map[string]interface{}, createdBy string) (*models.Policy, error) {
	// Get the template
	template, err := pts.storage.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Create a new policy based on the template
	policy := template.Template
	policy.ID = "" // Will be generated by the storage layer
	policy.CreatedBy = createdBy
	policy.Status = "draft"

	// Apply customizations
	if err := pts.applyCustomizations(&policy, customizations); err != nil {
		return nil, fmt.Errorf("failed to apply customizations: %w", err)
	}

	// Create the policy
	if err := pts.storage.CreatePolicy(ctx, &policy); err != nil {
		return nil, fmt.Errorf("failed to create policy from template: %w", err)
	}

	return &policy, nil
}

// applyCustomizations applies customizations to a policy
func (pts *PolicyTemplateService) applyCustomizations(policy *models.Policy, customizations map[string]interface{}) error {
	// Customize policy name
	if name, ok := customizations["name"].(string); ok {
		policy.Name = name
	}

	// Customize policy description
	if description, ok := customizations["description"].(string); ok {
		policy.Description = description
	}

	// Customize rules
	if rules, ok := customizations["rules"].([]interface{}); ok {
		if err := pts.customizeRules(&policy.Conditions, rules); err != nil {
			return fmt.Errorf("failed to customize rules: %w", err)
		}
	}

	// Customize privacy settings
	if privacy, ok := customizations["privacy"].(map[string]interface{}); ok {
		if err := pts.customizePrivacy(&policy.Privacy, privacy); err != nil {
			return fmt.Errorf("failed to customize privacy settings: %w", err)
		}
	}

	return nil
}

// customizeRules customizes policy rules
func (pts *PolicyTemplateService) customizeRules(conditions *models.PolicyConditions, rules []interface{}) error {
	for _, ruleData := range rules {
		ruleMap, ok := ruleData.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid rule format")
		}

		rule := models.PolicyRule{}

		// Set rule type
		if ruleType, ok := ruleMap["type"].(string); ok {
			rule.Type = ruleType
		} else {
			return fmt.Errorf("rule type is required")
		}

		// Set credential type if present
		if credentialType, ok := ruleMap["credential_type"].(string); ok {
			rule.CredentialType = credentialType
		}

		// Set issuer if present
		if issuer, ok := ruleMap["issuer"].(string); ok {
			rule.Issuer = issuer
		}

		// Set claim if present
		if claim, ok := ruleMap["claim"].(string); ok {
			rule.Claim = claim
		}

		// Set value if present
		if value, ok := ruleMap["value"]; ok {
			rule.Value = value
		}

		// Set min value if present
		if minValue, ok := ruleMap["min_value"]; ok {
			rule.MinValue = minValue
		}

		// Set max value if present
		if maxValue, ok := ruleMap["max_value"]; ok {
			rule.MaxValue = maxValue
		}

		conditions.Rules = append(conditions.Rules, rule)
	}

	return nil
}

// customizePrivacy customizes privacy settings
func (pts *PolicyTemplateService) customizePrivacy(privacy *models.PrivacySettings, customizations map[string]interface{}) error {
	if pprlEnabled, ok := customizations["pprl_enabled"].(bool); ok {
		privacy.PPRLEnabled = pprlEnabled
	}

	if selectiveDisclosure, ok := customizations["selective_disclosure"].(bool); ok {
		privacy.SelectiveDisclosure = selectiveDisclosure
	}

	if auditLevel, ok := customizations["audit_level"].(string); ok {
		privacy.AuditLevel = auditLevel
	}

	if retentionDays, ok := customizations["retention_days"].(float64); ok {
		privacy.RetentionDays = int(retentionDays)
	}

	return nil
}

// ListAvailableTemplates lists all available templates
func (pts *PolicyTemplateService) ListAvailableTemplates(ctx context.Context) ([]*models.PolicyTemplate, error) {
	filters := map[string]interface{}{
		"status": "active",
	}

	return pts.storage.ListTemplates(ctx, filters)
}

// GetTemplateCategories returns all available template categories
func (pts *PolicyTemplateService) GetTemplateCategories(ctx context.Context) ([]string, error) {
	templates, err := pts.ListAvailableTemplates(ctx)
	if err != nil {
		return nil, err
	}

	categories := make(map[string]bool)
	for _, template := range templates {
		categories[template.Category] = true
	}

	var result []string
	for category := range categories {
		result = append(result, category)
	}

	return result, nil
} 