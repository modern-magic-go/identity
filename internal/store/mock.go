package store

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/modern-magic-go/identity"
)

// MockStore 内存 IdentityStore 实现，用于测试和演示
type MockStore struct {
	nextID   atomic.Int64
	subjects map[identity.SubjectID]bool
	creds    map[string]*identity.Credential
}

// NewMockStore 创建 MockStore 实例
func NewMockStore() *MockStore {
	return &MockStore{
		subjects: make(map[identity.SubjectID]bool),
		creds:    make(map[string]*identity.Credential),
	}
}

func credKey(realm string, identityType identity.IdentityType, identifier string) string {
	return fmt.Sprintf("%s|%s|%s", realm, identityType, identifier)
}

// FindByRealmTypeIdentifier 按 Realm + 类型 + 标识符查找凭证
func (m *MockStore) FindByRealmTypeIdentifier(ctx context.Context, realm string, identityType identity.IdentityType, identifier string) (*identity.Credential, error) {
	cred, ok := m.creds[credKey(realm, identityType, identifier)]
	if !ok {
		return nil, identity.ErrCredentialNotFound
	}
	if isActive, exists := m.subjects[cred.SubjectID]; exists {
		cred.IsActive = isActive
	} else {
		cred.IsActive = true
	}
	return cred, nil
}

// CreateSubject 创建新的用户主体，返回自增 ID
func (m *MockStore) CreateSubject(ctx context.Context) (identity.SubjectID, error) {
	id := strconv.FormatInt(m.nextID.Add(1), 10)
	subjectID := identity.SubjectID(id)
	m.subjects[subjectID] = true
	return subjectID, nil
}

// BindCredential 将凭证绑定到指定 subject
func (m *MockStore) BindCredential(ctx context.Context, cred *identity.Credential) error {
	if _, exists := m.subjects[cred.SubjectID]; !exists {
		return identity.ErrSubjectNotFound
	}

	key := credKey(cred.Realm, cred.IdentityType, cred.Identifier)
	if _, exists := m.creds[key]; exists {
		return identity.ErrDuplicateCredential
	}

	if isActive, exists := m.subjects[cred.SubjectID]; exists {
		cred.IsActive = isActive
	} else {
		cred.IsActive = true
	}
	m.creds[key] = cred
	return nil
}

// ListBySubjectRealm 列出 subject 在指定 Realm 下的所有凭证（不含敏感数据）
func (m *MockStore) ListBySubjectRealm(ctx context.Context, subjectID identity.SubjectID, realm string) ([]identity.CredentialSummary, error) {
	isActive, exists := m.subjects[subjectID]
	if !exists {
		isActive = true
	}
	var result []identity.CredentialSummary
	for _, cred := range m.creds {
		if cred.SubjectID == subjectID && cred.Realm == realm {
			result = append(result, identity.CredentialSummary{
				Type:       cred.IdentityType,
				Identifier: cred.Identifier,
				IsActive:   isActive,
			})
		}
	}
	return result, nil
}

// SetInactive 将 Subject 设为不可用（IsActive=false），用于测试
func (m *MockStore) SetInactive(subjectID identity.SubjectID) {
	m.subjects[subjectID] = false
}

// WithTransaction 在 MockStore 中直接执行 fn（内存实现无需真实事务）
func (m *MockStore) WithTransaction(ctx context.Context, fn identity.TxFunc) error {
	return fn(ctx)
}

var _ identity.TransactionalStore = (*MockStore)(nil)
