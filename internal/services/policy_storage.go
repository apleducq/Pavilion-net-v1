package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
	_ "github.com/lib/pq"
)

// PolicyStorageImpl implements the PolicyStorage interface
type PolicyStorageImpl struct {
	db *sql.DB
}

// NewPolicyStorage creates a new policy storage implementation
func NewPolicyStorage(cfg *config.Config) (*PolicyStorageImpl, error) {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	storage := &PolicyStorageImpl{db: db}

	// Initialize the database schema
	if err := storage.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return storage, nil
}

// initSchema creates the necessary database tables
func (ps *PolicyStorageImpl) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS policies (
			id VARCHAR(255) PRIMARY KEY,
			version VARCHAR(50) NOT NULL,
			name VARCHAR(100) NOT NULL,
			description TEXT,
			conditions JSONB NOT NULL,
			privacy JSONB NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			created_by VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL,
			metadata JSONB,
			UNIQUE(id)
		)`,
		`CREATE TABLE IF NOT EXISTS policy_templates (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			description TEXT,
			category VARCHAR(100) NOT NULL,
			template JSONB NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			created_by VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL,
			metadata JSONB,
			UNIQUE(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_policies_status ON policies(status)`,
		`CREATE INDEX IF NOT EXISTS idx_policies_created_by ON policies(created_by)`,
		`CREATE INDEX IF NOT EXISTS idx_policy_templates_category ON policy_templates(category)`,
		`CREATE INDEX IF NOT EXISTS idx_policy_templates_status ON policy_templates(status)`,
	}

	for _, query := range queries {
		if _, err := ps.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}

	return nil
}

// CreatePolicy creates a new policy
func (ps *PolicyStorageImpl) CreatePolicy(ctx context.Context, policy *models.Policy) error {
	// Validate the policy
	if err := policy.Validate(); err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	// Marshal conditions and privacy settings to JSON
	conditionsJSON, err := json.Marshal(policy.Conditions)
	if err != nil {
		return fmt.Errorf("failed to marshal conditions: %w", err)
	}

	privacyJSON, err := json.Marshal(policy.Privacy)
	if err != nil {
		return fmt.Errorf("failed to marshal privacy settings: %w", err)
	}

	metadataJSON, err := json.Marshal(policy.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO policies (id, version, name, description, conditions, privacy, created_at, updated_at, created_by, status, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = ps.db.ExecContext(ctx, query,
		policy.ID,
		policy.Version,
		policy.Name,
		policy.Description,
		conditionsJSON,
		privacyJSON,
		policy.CreatedAt,
		policy.UpdatedAt,
		policy.CreatedBy,
		policy.Status,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}

	return nil
}

// GetPolicy retrieves a policy by ID
func (ps *PolicyStorageImpl) GetPolicy(ctx context.Context, id string) (*models.Policy, error) {
	query := `
		SELECT id, version, name, description, conditions, privacy, created_at, updated_at, created_by, status, metadata
		FROM policies
		WHERE id = $1
	`

	var policy models.Policy
	var conditionsJSON, privacyJSON, metadataJSON []byte

	err := ps.db.QueryRowContext(ctx, query, id).Scan(
		&policy.ID,
		&policy.Version,
		&policy.Name,
		&policy.Description,
		&conditionsJSON,
		&privacyJSON,
		&policy.CreatedAt,
		&policy.UpdatedAt,
		&policy.CreatedBy,
		&policy.Status,
		&metadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("policy not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(conditionsJSON, &policy.Conditions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conditions: %w", err)
	}

	if err := json.Unmarshal(privacyJSON, &policy.Privacy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal privacy settings: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &policy.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &policy, nil
}

// UpdatePolicy updates an existing policy
func (ps *PolicyStorageImpl) UpdatePolicy(ctx context.Context, policy *models.Policy) error {
	// Validate the policy
	if err := policy.Validate(); err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	// Update the timestamp
	policy.UpdatedAt = time.Now().Format(time.RFC3339)

	// Marshal conditions and privacy settings to JSON
	conditionsJSON, err := json.Marshal(policy.Conditions)
	if err != nil {
		return fmt.Errorf("failed to marshal conditions: %w", err)
	}

	privacyJSON, err := json.Marshal(policy.Privacy)
	if err != nil {
		return fmt.Errorf("failed to marshal privacy settings: %w", err)
	}

	metadataJSON, err := json.Marshal(policy.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE policies
		SET version = $2, name = $3, description = $4, conditions = $5, privacy = $6, updated_at = $7, created_by = $8, status = $9, metadata = $10
		WHERE id = $1
	`

	result, err := ps.db.ExecContext(ctx, query,
		policy.ID,
		policy.Version,
		policy.Name,
		policy.Description,
		conditionsJSON,
		privacyJSON,
		policy.UpdatedAt,
		policy.CreatedBy,
		policy.Status,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("policy not found: %s", policy.ID)
	}

	return nil
}

// DeletePolicy deletes a policy by ID
func (ps *PolicyStorageImpl) DeletePolicy(ctx context.Context, id string) error {
	query := `DELETE FROM policies WHERE id = $1`

	result, err := ps.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("policy not found: %s", id)
	}

	return nil
}

// ListPolicies lists policies with optional filters
func (ps *PolicyStorageImpl) ListPolicies(ctx context.Context, filters map[string]interface{}) ([]*models.Policy, error) {
	query := `
		SELECT id, version, name, description, conditions, privacy, created_at, updated_at, created_by, status, metadata
		FROM policies
	`
	
	var args []interface{}
	var conditions []string
	argIndex := 1

	// Add filters
	if status, ok := filters["status"].(string); ok {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, status)
		argIndex++
	}

	if createdBy, ok := filters["created_by"].(string); ok {
		conditions = append(conditions, fmt.Sprintf("created_by = $%d", argIndex))
		args = append(args, createdBy)
		argIndex++
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	query += " ORDER BY created_at DESC"

	rows, err := ps.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query policies: %w", err)
	}
	defer rows.Close()

	var policies []*models.Policy
	for rows.Next() {
		var policy models.Policy
		var conditionsJSON, privacyJSON, metadataJSON []byte

		err := rows.Scan(
			&policy.ID,
			&policy.Version,
			&policy.Name,
			&policy.Description,
			&conditionsJSON,
			&privacyJSON,
			&policy.CreatedAt,
			&policy.UpdatedAt,
			&policy.CreatedBy,
			&policy.Status,
			&metadataJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan policy: %w", err)
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(conditionsJSON, &policy.Conditions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal conditions: %w", err)
		}

		if err := json.Unmarshal(privacyJSON, &policy.Privacy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal privacy settings: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &policy.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		policies = append(policies, &policy)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over policies: %w", err)
	}

	return policies, nil
}

// CreateTemplate creates a new policy template
func (ps *PolicyStorageImpl) CreateTemplate(ctx context.Context, template *models.PolicyTemplate) error {
	// Validate the template
	if err := template.Validate(); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// Marshal template to JSON
	templateJSON, err := json.Marshal(template.Template)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	metadataJSON, err := json.Marshal(template.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO policy_templates (id, name, description, category, template, created_at, updated_at, created_by, status, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = ps.db.ExecContext(ctx, query,
		template.ID,
		template.Name,
		template.Description,
		template.Category,
		templateJSON,
		template.CreatedAt,
		template.UpdatedAt,
		template.CreatedBy,
		template.Status,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	return nil
}

// GetTemplate retrieves a policy template by ID
func (ps *PolicyStorageImpl) GetTemplate(ctx context.Context, id string) (*models.PolicyTemplate, error) {
	query := `
		SELECT id, name, description, category, template, created_at, updated_at, created_by, status, metadata
		FROM policy_templates
		WHERE id = $1
	`

	var template models.PolicyTemplate
	var templateJSON, metadataJSON []byte

	err := ps.db.QueryRowContext(ctx, query, id).Scan(
		&template.ID,
		&template.Name,
		&template.Description,
		&template.Category,
		&templateJSON,
		&template.CreatedAt,
		&template.UpdatedAt,
		&template.CreatedBy,
		&template.Status,
		&metadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("template not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(templateJSON, &template.Template); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &template.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &template, nil
}

// ListTemplates lists policy templates with optional filters
func (ps *PolicyStorageImpl) ListTemplates(ctx context.Context, filters map[string]interface{}) ([]*models.PolicyTemplate, error) {
	query := `
		SELECT id, name, description, category, template, created_at, updated_at, created_by, status, metadata
		FROM policy_templates
	`
	
	var args []interface{}
	var conditions []string
	argIndex := 1

	// Add filters
	if status, ok := filters["status"].(string); ok {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, status)
		argIndex++
	}

	if category, ok := filters["category"].(string); ok {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, category)
		argIndex++
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	query += " ORDER BY created_at DESC"

	rows, err := ps.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query templates: %w", err)
	}
	defer rows.Close()

	var templates []*models.PolicyTemplate
	for rows.Next() {
		var template models.PolicyTemplate
		var templateJSON, metadataJSON []byte

		err := rows.Scan(
			&template.ID,
			&template.Name,
			&template.Description,
			&template.Category,
			&templateJSON,
			&template.CreatedAt,
			&template.UpdatedAt,
			&template.CreatedBy,
			&template.Status,
			&metadataJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(templateJSON, &template.Template); err != nil {
			return nil, fmt.Errorf("failed to unmarshal template: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &template.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		templates = append(templates, &template)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over templates: %w", err)
	}

	return templates, nil
}

// Close closes the database connection
func (ps *PolicyStorageImpl) Close() error {
	return ps.db.Close()
} 