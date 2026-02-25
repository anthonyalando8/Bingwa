// internal/repository/postgres/agent_config_repository.go
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bingwa-service/internal/domain/config"
	xerrors "bingwa-service/internal/pkg/errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentConfigRepository struct {
	db *pgxpool.Pool
}

func NewAgentConfigRepository(db *pgxpool.Pool) *AgentConfigRepository {
	return &AgentConfigRepository{db: db}
}

// Create creates a new agent config
func (r *AgentConfigRepository) Create(ctx context.Context, cfg *config.AgentConfig) error {
	query := `
		INSERT INTO agent_configs (
			agent_identity_id, config_key, config_value, description,
			device_id, is_global, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	var configValueJSON, metadataJSON []byte
	var err error

	configValueJSON, err = json.Marshal(cfg.ConfigValue)
	if err != nil {
		return fmt.Errorf("failed to marshal config_value: %w", err)
	}

	if cfg.Metadata != nil {
		metadataJSON, err = json.Marshal(cfg.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = r.db.QueryRow(
		ctx, query,
		cfg.AgentIdentityID, cfg.ConfigKey, configValueJSON, cfg.Description,
		cfg.DeviceID, cfg.IsGlobal, metadataJSON,
	).Scan(&cfg.ID, &cfg.CreatedAt, &cfg.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	return nil
}

// FindByID retrieves a config by ID
func (r *AgentConfigRepository) FindByID(ctx context.Context, id int64) (*config.AgentConfig, error) {
	query := `
		SELECT id, agent_identity_id, config_key, config_value, description,
		       device_id, is_global, metadata, created_at, updated_at
		FROM agent_configs
		WHERE id = $1
	`

	var cfg config.AgentConfig
	var configValueJSON, metadataJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&cfg.ID, &cfg.AgentIdentityID, &cfg.ConfigKey, &configValueJSON, &cfg.Description,
		&cfg.DeviceID, &cfg.IsGlobal, &metadataJSON, &cfg.CreatedAt, &cfg.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find config: %w", err)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(configValueJSON, &cfg.ConfigValue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config_value: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &cfg.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &cfg, nil
}

// FindByKey retrieves a config by agent ID and key
func (r *AgentConfigRepository) FindByKey(ctx context.Context, agentID int64, configKey string, deviceID *string) (*config.AgentConfig, error) {
	query := `
		SELECT id, agent_identity_id, config_key, config_value, description,
		       device_id, is_global, metadata, created_at, updated_at
		FROM agent_configs
		WHERE agent_identity_id = $1 AND config_key = $2 AND 
		      (device_id = $3 OR (device_id IS NULL AND $3 IS NULL))
	`

	var cfg config.AgentConfig
	var configValueJSON, metadataJSON []byte

	var deviceIDParam interface{}
	if deviceID != nil {
		deviceIDParam = *deviceID
	}

	err := r.db.QueryRow(ctx, query, agentID, configKey, deviceIDParam).Scan(
		&cfg.ID, &cfg.AgentIdentityID, &cfg.ConfigKey, &configValueJSON, &cfg.Description,
		&cfg.DeviceID, &cfg.IsGlobal, &metadataJSON, &cfg.CreatedAt, &cfg.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find config: %w", err)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(configValueJSON, &cfg.ConfigValue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config_value: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &cfg.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &cfg, nil
}

// Update updates a config
func (r *AgentConfigRepository) Update(ctx context.Context, id int64, cfg *config.AgentConfig) error {
	query := `
		UPDATE agent_configs
		SET config_value = $1, description = $2, metadata = $3, updated_at = $4
		WHERE id = $5
	`

	var configValueJSON, metadataJSON []byte
	var err error

	configValueJSON, err = json.Marshal(cfg.ConfigValue)
	if err != nil {
		return fmt.Errorf("failed to marshal config_value: %w", err)
	}

	if cfg.Metadata != nil {
		metadataJSON, err = json.Marshal(cfg.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	result, err := r.db.Exec(
		ctx, query,
		configValueJSON, cfg.Description, metadataJSON, time.Now(), id,
	)

	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// Delete deletes a config
func (r *AgentConfigRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM agent_configs WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// List retrieves configs with filters
func (r *AgentConfigRepository) List(ctx context.Context, agentID int64, filters *config.ConfigListFilters) ([]config.AgentConfig, int64, error) {
	// Build WHERE clause
	conditions := []string{"agent_identity_id = $1"}
	args := []interface{}{agentID}
	argPos := 2

	if filters.IsGlobal != nil {
		conditions = append(conditions, fmt.Sprintf("is_global = $%d", argPos))
		args = append(args, *filters.IsGlobal)
		argPos++
	}

	if filters.DeviceID != "" {
		conditions = append(conditions, fmt.Sprintf("device_id = $%d", argPos))
		args = append(args, filters.DeviceID)
		argPos++
	}

	if filters.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(config_key ILIKE $%d OR description ILIKE $%d)",
			argPos, argPos,
		))
		args = append(args, "%"+filters.Search+"%")
		argPos++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM agent_configs WHERE %s", whereClause)
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count configs: %w", err)
	}

	// Pagination
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PageSize < 1 {
		filters.PageSize = 20
	}

	offset := (filters.Page - 1) * filters.PageSize
	limit := filters.PageSize

	// Query configs
	query := fmt.Sprintf(`
		SELECT id, agent_identity_id, config_key, config_value, description,
		       device_id, is_global, metadata, created_at, updated_at
		FROM agent_configs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list configs: %w", err)
	}
	defer rows.Close()

	configs := []config.AgentConfig{}
	for rows.Next() {
		var cfg config.AgentConfig
		var configValueJSON, metadataJSON []byte

		err := rows.Scan(
			&cfg.ID, &cfg.AgentIdentityID, &cfg.ConfigKey, &configValueJSON, &cfg.Description,
			&cfg.DeviceID, &cfg.IsGlobal, &metadataJSON, &cfg.CreatedAt, &cfg.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan config: %w", err)
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(configValueJSON, &cfg.ConfigValue); err == nil {
			if len(metadataJSON) > 0 {
				json.Unmarshal(metadataJSON, &cfg.Metadata)
			}
			configs = append(configs, cfg)
		}
	}

	return configs, total, nil
}

// GetAllByAgent retrieves all configs for an agent
func (r *AgentConfigRepository) GetAllByAgent(ctx context.Context, agentID int64) ([]config.AgentConfig, error) {
	query := `
		SELECT id, agent_identity_id, config_key, config_value, description,
		       device_id, is_global, metadata, created_at, updated_at
		FROM agent_configs
		WHERE agent_identity_id = $1
		ORDER BY config_key ASC
	`

	rows, err := r.db.Query(ctx, query, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get configs: %w", err)
	}
	defer rows.Close()

	configs := []config.AgentConfig{}
	for rows.Next() {
		var cfg config.AgentConfig
		var configValueJSON, metadataJSON []byte

		err := rows.Scan(
			&cfg.ID, &cfg.AgentIdentityID, &cfg.ConfigKey, &configValueJSON, &cfg.Description,
			&cfg.DeviceID, &cfg.IsGlobal, &metadataJSON, &cfg.CreatedAt, &cfg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan config: %w", err)
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(configValueJSON, &cfg.ConfigValue); err == nil {
			if len(metadataJSON) > 0 {
				json.Unmarshal(metadataJSON, &cfg.Metadata)
			}
			configs = append(configs, cfg)
		}
	}

	return configs, nil
}

// GetGlobalConfigs retrieves all global configs for an agent
func (r *AgentConfigRepository) GetGlobalConfigs(ctx context.Context, agentID int64) ([]config.AgentConfig, error) {
	query := `
		SELECT id, agent_identity_id, config_key, config_value, description,
		       device_id, is_global, metadata, created_at, updated_at
		FROM agent_configs
		WHERE agent_identity_id = $1 AND is_global = TRUE
		ORDER BY config_key ASC
	`

	rows, err := r.db.Query(ctx, query, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get global configs: %w", err)
	}
	defer rows.Close()

	configs := []config.AgentConfig{}
	for rows.Next() {
		var cfg config.AgentConfig
		var configValueJSON, metadataJSON []byte

		err := rows.Scan(
			&cfg.ID, &cfg.AgentIdentityID, &cfg.ConfigKey, &configValueJSON, &cfg.Description,
			&cfg.DeviceID, &cfg.IsGlobal, &metadataJSON, &cfg.CreatedAt, &cfg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan config: %w", err)
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(configValueJSON, &cfg.ConfigValue); err == nil {
			if len(metadataJSON) > 0 {
				json.Unmarshal(metadataJSON, &cfg.Metadata)
			}
			configs = append(configs, cfg)
		}
	}

	return configs, nil
}

// GetDeviceConfigs retrieves all configs for a specific device
func (r *AgentConfigRepository) GetDeviceConfigs(ctx context.Context, agentID int64, deviceID string) ([]config.AgentConfig, error) {
	query := `
		SELECT id, agent_identity_id, config_key, config_value, description,
		       device_id, is_global, metadata, created_at, updated_at
		FROM agent_configs
		WHERE agent_identity_id = $1 AND device_id = $2
		ORDER BY config_key ASC
	`

	rows, err := r.db.Query(ctx, query, agentID, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device configs: %w", err)
	}
	defer rows.Close()

	configs := []config.AgentConfig{}
	for rows.Next() {
		var cfg config.AgentConfig
		var configValueJSON, metadataJSON []byte

		err := rows.Scan(
			&cfg.ID, &cfg.AgentIdentityID, &cfg.ConfigKey, &configValueJSON, &cfg.Description,
			&cfg.DeviceID, &cfg.IsGlobal, &metadataJSON, &cfg.CreatedAt, &cfg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan config: %w", err)
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(configValueJSON, &cfg.ConfigValue); err == nil {
			if len(metadataJSON) > 0 {
				json.Unmarshal(metadataJSON, &cfg.Metadata)
			}
			configs = append(configs, cfg)
		}
	}

	return configs, nil
}

// ExistsByKey checks if a config key exists for an agent
func (r *AgentConfigRepository) ExistsByKey(ctx context.Context, agentID int64, configKey string, deviceID *string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM agent_configs 
			WHERE agent_identity_id = $1 AND config_key = $2 AND 
			      (device_id = $3 OR (device_id IS NULL AND $3 IS NULL))
		)
	`
	
	var deviceIDParam interface{}
	if deviceID != nil {
		deviceIDParam = *deviceID
	}
	
	var exists bool
	err := r.db.QueryRow(ctx, query, agentID, configKey, deviceIDParam).Scan(&exists)
	return exists, err
}