// Package core 是 identity 库的公共 API 入口。
//
// IdentityCore 聚合持久化层（IdentityStore）和密码学适配器（CredentialVerifier），
// 对外暴露 4 个方法：
//
//   - VerifyCredential：验证调用方提供的标识和凭证是否匹配（密码/TOTP）
//   - GetOrInitializeSubjectID：静默解析——标识符已存在则返回已有 SubjectID，否则创建新 Subject
//   - BindCredential：为已存在的 Subject 绑定新凭证
//   - ListCredentials：列出 Subject 在指定 Realm 下所有凭证摘要
//
// 快速开始：
//
//	import (
//	    "github.com/modern-magic-go/identity/core"
//	    "github.com/modern-magic-go/identity/internal/store"
//	)
//
//	ic := core.NewIdentityCore(store.NewMockStore())
//	out, _ := ic.GetOrInitializeSubjectID(ctx, identity.GetOrInitSubjectInput{...})
package core
