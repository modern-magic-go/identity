package recache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/modern-magic-go/identity"
	"github.com/redis/go-redis/v9"
)

func enabled(client *redis.Client, ttl time.Duration) bool {
	return client != nil && ttl > 0
}

func cacheGet[T any](ctx context.Context, client *redis.Client, key string) (*T, error) {
	value, err := client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var out T
	if err := json.Unmarshal(value, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func cacheSet(ctx context.Context, client *redis.Client, key string, ttl time.Duration, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return client.Set(ctx, key, data, ttl).Err()
}

func cacheDel(ctx context.Context, client *redis.Client, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return client.Del(ctx, keys...).Err()
}

func subjectByIDKey(id int64) string { return fmt.Sprintf("identity:subject:id:%d", id) }
func subjectByNoKey(subjectNo string) string { return "identity:subject:no:" + subjectNo }
func identityByLoginKey(realm, provider, identityType, identifier string) string {
	return fmt.Sprintf("identity:login:%s:%s:%s:%s", realm, provider, identityType, identity.NormalizeIdentifier(identifier))
}
func identitiesBySubjectKey(subjectID int64) string { return fmt.Sprintf("identity:identities:subject:%d", subjectID) }
func credentialBySubjectRealmKey(subjectID int64, realm string) string {
	return fmt.Sprintf("identity:credential:%d:%s", subjectID, realm)
}

// SubjectRepository adds Redis cache on top of a subject repository.
type SubjectRepository struct {
	base  identity.SubjectRepository
	cache *redis.Client
	ttl   time.Duration
}

// NewSubjectRepository creates a cached subject repository.
func NewSubjectRepository(base identity.SubjectRepository, client *redis.Client, ttl time.Duration) *SubjectRepository {
	return &SubjectRepository{base: base, cache: client, ttl: ttl}
}

func (r *SubjectRepository) GetByID(ctx context.Context, id int64) (*identity.Subject, error) {
	if enabled(r.cache, r.ttl) {
		if item, err := cacheGet[identity.Subject](ctx, r.cache, subjectByIDKey(id)); err == nil {
			return item, nil
		} else if err != nil && !errors.Is(err, redis.Nil) {
			return nil, err
		}
	}
	item, err := r.base.GetByID(ctx, id)
	if err != nil || item == nil || !enabled(r.cache, r.ttl) {
		return item, err
	}
	_ = cacheSet(ctx, r.cache, subjectByIDKey(item.ID), r.ttl, item)
	_ = cacheSet(ctx, r.cache, subjectByNoKey(item.SubjectNo), r.ttl, item)
	return item, nil
}

func (r *SubjectRepository) GetBySubjectNo(ctx context.Context, subjectNo string) (*identity.Subject, error) {
	if enabled(r.cache, r.ttl) {
		if item, err := cacheGet[identity.Subject](ctx, r.cache, subjectByNoKey(subjectNo)); err == nil {
			return item, nil
		} else if err != nil && !errors.Is(err, redis.Nil) {
			return nil, err
		}
	}
	item, err := r.base.GetBySubjectNo(ctx, subjectNo)
	if err != nil || item == nil || !enabled(r.cache, r.ttl) {
		return item, err
	}
	_ = cacheSet(ctx, r.cache, subjectByIDKey(item.ID), r.ttl, item)
	_ = cacheSet(ctx, r.cache, subjectByNoKey(item.SubjectNo), r.ttl, item)
	return item, nil
}

func (r *SubjectRepository) Save(ctx context.Context, subject *identity.Subject) error {
	if err := r.base.Save(ctx, subject); err != nil {
		return err
	}
	if enabled(r.cache, r.ttl) && subject != nil {
		_ = cacheSet(ctx, r.cache, subjectByIDKey(subject.ID), r.ttl, subject)
		_ = cacheSet(ctx, r.cache, subjectByNoKey(subject.SubjectNo), r.ttl, subject)
	}
	return nil
}

// IdentityRepository adds Redis cache on top of an identity repository.
type IdentityRepository struct {
	base  identity.IdentityRepository
	cache *redis.Client
	ttl   time.Duration
}

// NewIdentityRepository creates a cached identity repository.
func NewIdentityRepository(base identity.IdentityRepository, client *redis.Client, ttl time.Duration) *IdentityRepository {
	return &IdentityRepository{base: base, cache: client, ttl: ttl}
}

func (r *IdentityRepository) FindByLoginIdentity(ctx context.Context, realm, provider, identityType, identifier string) (*identity.Identity, error) {
	key := identityByLoginKey(realm, provider, identityType, identifier)
	if enabled(r.cache, r.ttl) {
		if item, err := cacheGet[identity.Identity](ctx, r.cache, key); err == nil {
			return item, nil
		} else if err != nil && !errors.Is(err, redis.Nil) {
			return nil, err
		}
	}
	item, err := r.base.FindByLoginIdentity(ctx, realm, provider, identityType, identifier)
	if err != nil || item == nil || !enabled(r.cache, r.ttl) {
		return item, err
	}
	_ = cacheSet(ctx, r.cache, key, r.ttl, item)
	_ = cacheSet(ctx, r.cache, identitiesBySubjectKey(item.SubjectID), r.ttl, []*identity.Identity{item})
	return item, nil
}

func (r *IdentityRepository) FindBySubjectID(ctx context.Context, subjectID int64) ([]*identity.Identity, error) {
	key := identitiesBySubjectKey(subjectID)
	if enabled(r.cache, r.ttl) {
		if items, err := cacheGet[[]*identity.Identity](ctx, r.cache, key); err == nil {
			return *items, nil
		} else if err != nil && !errors.Is(err, redis.Nil) {
			return nil, err
		}
	}
	items, err := r.base.FindBySubjectID(ctx, subjectID)
	if err != nil || !enabled(r.cache, r.ttl) {
		return items, err
	}
	_ = cacheSet(ctx, r.cache, key, r.ttl, items)
	for _, item := range items {
		_ = cacheSet(ctx, r.cache, identityByLoginKey(item.Realm, item.Provider, item.IdentityType, item.Identifier), r.ttl, item)
	}
	return items, nil
}

func (r *IdentityRepository) Save(ctx context.Context, item *identity.Identity) error {
	if err := r.base.Save(ctx, item); err != nil {
		return err
	}
	if enabled(r.cache, r.ttl) && item != nil {
		_ = cacheSet(ctx, r.cache, identityByLoginKey(item.Realm, item.Provider, item.IdentityType, item.Identifier), r.ttl, item)
		_ = cacheSet(ctx, r.cache, identitiesBySubjectKey(item.SubjectID), r.ttl, []*identity.Identity{item})
	}
	return nil
}

// PasswordCredentialRepository adds Redis cache on top of a credential repository.
type PasswordCredentialRepository struct {
	base  identity.PasswordCredentialRepository
	cache *redis.Client
	ttl   time.Duration
}

// NewPasswordCredentialRepository creates a cached credential repository.
func NewPasswordCredentialRepository(base identity.PasswordCredentialRepository, client *redis.Client, ttl time.Duration) *PasswordCredentialRepository {
	return &PasswordCredentialRepository{base: base, cache: client, ttl: ttl}
}

func (r *PasswordCredentialRepository) FindBySubjectAndRealm(ctx context.Context, subjectID int64, realm string) (*identity.PasswordCredential, error) {
	key := credentialBySubjectRealmKey(subjectID, realm)
	if enabled(r.cache, r.ttl) {
		if item, err := cacheGet[identity.PasswordCredential](ctx, r.cache, key); err == nil {
			return item, nil
		} else if err != nil && !errors.Is(err, redis.Nil) {
			return nil, err
		}
	}
	item, err := r.base.FindBySubjectAndRealm(ctx, subjectID, realm)
	if err != nil || item == nil || !enabled(r.cache, r.ttl) {
		return item, err
	}
	_ = cacheSet(ctx, r.cache, key, r.ttl, item)
	return item, nil
}

func (r *PasswordCredentialRepository) Save(ctx context.Context, item *identity.PasswordCredential) error {
	if err := r.base.Save(ctx, item); err != nil {
		return err
	}
	if enabled(r.cache, r.ttl) && item != nil {
		_ = cacheSet(ctx, r.cache, credentialBySubjectRealmKey(item.SubjectID, item.Realm), r.ttl, item)
	}
	return nil
}

func (r *PasswordCredentialRepository) IncrementFailedCount(ctx context.Context, credentialID int64, now time.Time) error {
	if err := r.base.IncrementFailedCount(ctx, credentialID, now); err != nil {
		return err
	}
	if !enabled(r.cache, r.ttl) {
		return nil
	}
	return cacheDel(ctx, r.cache, fmt.Sprintf("identity:credential:id:%d", credentialID))
}
