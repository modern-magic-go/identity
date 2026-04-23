package mysqlstore

import (
	"context"
	"database/sql"
)

// EnsureSchema creates the tables required by the identity package.
func EnsureSchema(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS subjects (
			id BIGINT NOT NULL AUTO_INCREMENT,
			subject_no VARCHAR(64) NOT NULL,
			subject_type VARCHAR(64) NOT NULL DEFAULT '',
			realm VARCHAR(64) NOT NULL DEFAULT '',
			status VARCHAR(32) NOT NULL DEFAULT 'pending_activation',
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			deleted_at DATETIME(6) NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uk_subject_no (subject_no),
			KEY idx_subject_realm (realm),
			KEY idx_subject_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS identities (
			id BIGINT NOT NULL AUTO_INCREMENT,
			subject_id BIGINT NOT NULL,
			realm VARCHAR(64) NOT NULL DEFAULT '',
			provider VARCHAR(64) NOT NULL DEFAULT '',
			identity_type VARCHAR(64) NOT NULL DEFAULT '',
			provider_app_id VARCHAR(128) NOT NULL DEFAULT '',
			identifier VARCHAR(255) NOT NULL DEFAULT '',
			union_id VARCHAR(128) NOT NULL DEFAULT '',
			status VARCHAR(32) NOT NULL DEFAULT 'active',
			last_used_at DATETIME(6) NULL,
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uk_login_identity (realm, provider, identity_type, identifier),
			KEY idx_identity_subject_id (subject_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS password_credentials (
			id BIGINT NOT NULL AUTO_INCREMENT,
			subject_id BIGINT NOT NULL,
			realm VARCHAR(64) NOT NULL DEFAULT '',
			password_hash VARCHAR(255) NOT NULL DEFAULT '',
			password_algo VARCHAR(32) NOT NULL DEFAULT '',
			password_version INT NOT NULL DEFAULT 0,
			need_reset BOOLEAN NOT NULL DEFAULT FALSE,
			failed_count INT NOT NULL DEFAULT 0,
			locked_until DATETIME(6) NULL,
			password_updated_at DATETIME(6) NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'active',
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uk_password_credential (subject_id, realm),
			KEY idx_credential_status (status),
			KEY idx_credential_subject_realm (subject_id, realm)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	}

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}
