package store

import (
	"context"
	"fmt"

	"github.com/modern-magic-go/identity"
	"github.com/modern-magic-go/identity/internal/idgen"
)

// MockStore 内存 IdentityStore 实现，用于测试和演示
type MockStore struct {
	idGen    idgen.IDGenerator
	subjects map[int64]bool
	creds    map[string]*identity.Credential
}

// NewMockStore 创建 MockStore 实例
func NewMockStore(idGen idgen.IDGenerator) *MockStore {
	return &MockStore{
		idGen:    idGen,
		subjects: make(map[int64]bool),
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
	return cred, nil
}

// CreateSubject 创建新的用户主体，返回 IDGenerator 生成的 subject_id
func (m *MockStore) CreateSubject(ctx context.Context) (int64, error) {
	id := m.idGen.Generate()
	m.subjects[id] = true
	return id, nil
}

// BindCredential 将凭证绑定到指定 subject
func (m *MockStore) BindCredential(ctx context.Context, cred *identity.Credential) error {
	if !m.subjects[cred.SubjectID] {
		return identity.ErrSubjectNotFound
	}

	key := credKey(cred.Realm, cred.IdentityType, cred.Identifier)
	if _, exists := m.creds[key]; exists {
		return identity.ErrDuplicateCredential
	}

	m.creds[key] = cred
	return nil
}

// ListBySubjectRealm 列出 subject 在指定 Realm 下的所有凭证（不含敏感数据）
func (m *MockStore) ListBySubjectRealm(ctx context.Context, subjectID int64, realm string) ([]identity.CredentialSummary, error) {
	var result []identity.CredentialSummary
	for _, cred := range m.creds {
		if cred.SubjectID == subjectID && cred.Realm == realm {
			result = append(result, identity.CredentialSummary{
				Type:       cred.IdentityType,
				Identifier: cred.Identifier,
			})
		}
	}
	return result, nil
}
