package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/modern-magic-go/identity"
)

const timeLayout = "2006-01-02 15:04:05.999999"

func timeToNullTime(value *time.Time) sql.NullTime {
	if value == nil || value.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value.UTC(), Valid: true}
}

func nullTimeToPtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time.UTC()
	return &t
}

func scanSubject(row interface{ Scan(dest ...any) error }) (*identity.Subject, error) {
	var deletedAt sql.NullTime
	var subject identity.Subject
	if err := row.Scan(
		&subject.ID,
		&subject.SubjectNo,
		&subject.SubjectType,
		&subject.Realm,
		&subject.Status,
		&subject.CreatedAt,
		&subject.UpdatedAt,
		&deletedAt,
	); err != nil {
		return nil, err
	}
	subject.DeletedAt = nullTimeToPtr(deletedAt)
	return &subject, nil
}

func scanIdentity(row interface{ Scan(dest ...any) error }) (*identity.Identity, error) {
	var lastUsedAt sql.NullTime
	var item identity.Identity
	if err := row.Scan(
		&item.ID,
		&item.SubjectID,
		&item.Realm,
		&item.Provider,
		&item.IdentityType,
		&item.ProviderAppID,
		&item.Identifier,
		&item.UnionID,
		&item.Status,
		&lastUsedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.LastUsedAt = nullTimeToPtr(lastUsedAt)
	return &item, nil
}

func scanPasswordCredential(row interface{ Scan(dest ...any) error }) (*identity.PasswordCredential, error) {
	var lockedUntil sql.NullTime
	var passwordUpdatedAt sql.NullTime
	var item identity.PasswordCredential
	if err := row.Scan(
		&item.ID,
		&item.SubjectID,
		&item.Realm,
		&item.PasswordHash,
		&item.PasswordAlgo,
		&item.PasswordVersion,
		&item.NeedReset,
		&item.FailedCount,
		&lockedUntil,
		&passwordUpdatedAt,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.LockedUntil = nullTimeToPtr(lockedUntil)
	item.PasswordUpdatedAt = nullTimeToPtr(passwordUpdatedAt)
	return &item, nil
}

func saveResult(err error, rows int64) error {
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("no rows affected")
	}
	return nil
}

// SubjectRepository stores subjects in MySQL.
type SubjectRepository struct{ db *sql.DB }

// NewSubjectRepository creates a MySQL-backed subject repository.
func NewSubjectRepository(db *sql.DB) *SubjectRepository { return &SubjectRepository{db: db} }

// GetByID loads a subject by ID.
func (r *SubjectRepository) GetByID(ctx context.Context, id int64) (*identity.Subject, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, subject_no, subject_type, realm, status, created_at, updated_at, deleted_at
		FROM subjects
		WHERE id = ?
		LIMIT 1`, id)
	item, err := scanSubject(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

// GetBySubjectNo loads a subject by its unique subject number.
func (r *SubjectRepository) GetBySubjectNo(ctx context.Context, subjectNo string) (*identity.Subject, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, subject_no, subject_type, realm, status, created_at, updated_at, deleted_at
		FROM subjects
		WHERE subject_no = ?
		LIMIT 1`, subjectNo)
	item, err := scanSubject(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

// Save inserts or updates a subject.
func (r *SubjectRepository) Save(ctx context.Context, subject *identity.Subject) error {
	if subject == nil {
		return fmt.Errorf("subject is nil")
	}
	query := `
		INSERT INTO subjects (id, subject_no, subject_type, realm, status, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			subject_type = VALUES(subject_type),
			realm = VALUES(realm),
			status = VALUES(status),
			updated_at = VALUES(updated_at),
			deleted_at = VALUES(deleted_at)`
	result, err := r.db.ExecContext(ctx, query,
		subject.ID,
		subject.SubjectNo,
		subject.SubjectType,
		subject.Realm,
		subject.Status,
		subject.CreatedAt.UTC(),
		subject.UpdatedAt.UTC(),
		timeToNullTime(subject.DeletedAt),
	)
	if err != nil {
		return err
	}
	if subject.ID == 0 {
		id, err := result.LastInsertId()
		if err == nil && id > 0 {
			subject.ID = id
		}
	}
	return nil
}

// IdentityRepository stores identities in MySQL.
type IdentityRepository struct{ db *sql.DB }

// NewIdentityRepository creates a MySQL-backed identity repository.
func NewIdentityRepository(db *sql.DB) *IdentityRepository { return &IdentityRepository{db: db} }

// FindByLoginIdentity finds an identity by its login identity key.
func (r *IdentityRepository) FindByLoginIdentity(ctx context.Context, realm, provider, identityType, identifier string) (*identity.Identity, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, subject_id, realm, provider, identity_type, provider_app_id, identifier, union_id, status, last_used_at, created_at, updated_at
		FROM identities
		WHERE realm = ? AND provider = ? AND identity_type = ? AND identifier = ?
		LIMIT 1`, realm, provider, identityType, identity.NormalizeIdentifier(identifier))
	item, err := scanIdentity(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

// FindBySubjectID returns all identities for a subject.
func (r *IdentityRepository) FindBySubjectID(ctx context.Context, subjectID int64) ([]*identity.Identity, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, subject_id, realm, provider, identity_type, provider_app_id, identifier, union_id, status, last_used_at, created_at, updated_at
		FROM identities
		WHERE subject_id = ?
		ORDER BY id ASC`, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*identity.Identity
	for rows.Next() {
		item, err := scanIdentity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// Save inserts or updates an identity.
func (r *IdentityRepository) Save(ctx context.Context, item *identity.Identity) error {
	if item == nil {
		return fmt.Errorf("identity is nil")
	}
	query := `
		INSERT INTO identities (id, subject_id, realm, provider, identity_type, provider_app_id, identifier, union_id, status, last_used_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			subject_id = VALUES(subject_id),
			provider_app_id = VALUES(provider_app_id),
			union_id = VALUES(union_id),
			status = VALUES(status),
			last_used_at = VALUES(last_used_at),
			updated_at = VALUES(updated_at)`
	result, err := r.db.ExecContext(ctx, query,
		item.ID,
		item.SubjectID,
		item.Realm,
		item.Provider,
		item.IdentityType,
		item.ProviderAppID,
		identity.NormalizeIdentifier(item.Identifier),
		item.UnionID,
		item.Status,
		timeToNullTime(item.LastUsedAt),
		item.CreatedAt.UTC(),
		item.UpdatedAt.UTC(),
	)
	if err != nil {
		return err
	}
	if item.ID == 0 {
		id, err := result.LastInsertId()
		if err == nil && id > 0 {
			item.ID = id
		}
	}
	return nil
}

// PasswordCredentialRepository stores password credentials in MySQL.
type PasswordCredentialRepository struct{ db *sql.DB }

// NewPasswordCredentialRepository creates a MySQL-backed credential repository.
func NewPasswordCredentialRepository(db *sql.DB) *PasswordCredentialRepository { return &PasswordCredentialRepository{db: db} }

// FindBySubjectAndRealm loads a credential by subject and realm.
func (r *PasswordCredentialRepository) FindBySubjectAndRealm(ctx context.Context, subjectID int64, realm string) (*identity.PasswordCredential, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, subject_id, realm, password_hash, password_algo, password_version, need_reset, failed_count, locked_until, password_updated_at, status, created_at, updated_at
		FROM password_credentials
		WHERE subject_id = ? AND realm = ?
		LIMIT 1`, subjectID, realm)
	item, err := scanPasswordCredential(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

// Save inserts or updates a password credential.
func (r *PasswordCredentialRepository) Save(ctx context.Context, item *identity.PasswordCredential) error {
	if item == nil {
		return fmt.Errorf("credential is nil")
	}
	query := `
		INSERT INTO password_credentials (id, subject_id, realm, password_hash, password_algo, password_version, need_reset, failed_count, locked_until, password_updated_at, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			password_hash = VALUES(password_hash),
			password_algo = VALUES(password_algo),
			password_version = VALUES(password_version),
			need_reset = VALUES(need_reset),
			failed_count = VALUES(failed_count),
			locked_until = VALUES(locked_until),
			password_updated_at = VALUES(password_updated_at),
			status = VALUES(status),
			updated_at = VALUES(updated_at)`
	result, err := r.db.ExecContext(ctx, query,
		item.ID,
		item.SubjectID,
		item.Realm,
		item.PasswordHash,
		item.PasswordAlgo,
		item.PasswordVersion,
		item.NeedReset,
		item.FailedCount,
		timeToNullTime(item.LockedUntil),
		timeToNullTime(item.PasswordUpdatedAt),
		item.Status,
		item.CreatedAt.UTC(),
		item.UpdatedAt.UTC(),
	)
	if err != nil {
		return err
	}
	if item.ID == 0 {
		id, err := result.LastInsertId()
		if err == nil && id > 0 {
			item.ID = id
		}
	}
	return nil
}

// IncrementFailedCount increments failed login count for a credential.
func (r *PasswordCredentialRepository) IncrementFailedCount(ctx context.Context, credentialID int64, now time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE password_credentials
		SET failed_count = failed_count + 1,
			updated_at = ?
		WHERE id = ?`, now.UTC(), credentialID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
