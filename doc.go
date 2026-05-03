// Package identity 提供 Headless 身份核能力——把外部标识（手机号/用户名/微信 OpenID）到系统内实体的映射和凭证（密码/TOTP）校验封装为可复用的原子库。
//
// 核心概念
//
//   - SubjectID：全局唯一用户标识（type SubjectID string），兼容 Snowflake int64 和 UUID 两种 ID 体系。
//     构造通道：SubjectIDFromInt64(id int64) / SubjectIDFromString(id string)
//   - Realm：领域/命名空间（string），账号池的物理隔离单位，同一标识在不同 Realm 下对应不同 SubjectID
//   - IdentityType：凭证类型，内置 PASSWORD / TOTP / WECHAT_OPENID / WECHAT_UNIONID / EMAIL / SMS
//   - Subject：用户主体，含 IsActive 标记账号级是否可认证（false 时所有凭证认证均被拒绝）
//   - Credential：原子凭证，记录一个 Subject 在某 Realm 下的一种登录方式。
//     包含 SubjectActive（store 层填充的 Subject 级活跃状态，第一道闸）和 IsActive（单凭证禁用标记，第二道闸）。
//     还包含 Meta map[string]string 字段，记录认证附属信息（如微信 appid、TOTP issuer），key/value 由各 IdentityType 约定。
//   - CredentialSummary：凭证摘要，脱敏后不含 CredentialData，含 IsActive，安全返回给调用方
//   - Meta：凭证元信息（map[string]string），核心库不做 schema 校验，仅做容器。已知 key：appid（第三方登录应用标识）、
//     totp_issuer（TOTP 认证器发行者名称）
//   - TransactionalStore：可选接口扩展，支持事务内执行多步 IdentityStore 操作。实现者可选满足
//
// 验证流程（三层闸）
//
//	FindByRealmTypeIdentifier → Credential{SubjectActive, IsActive}
//	  if !SubjectActive → ACCOUNT_LOCKED（账号被封）
//	  if !IsActive → CREDENTIAL_DISABLED（单登录方式被禁）
//	  find verifier → verify credential_data
//
//	GetOrInitializeSubjectID 只查 SubjectActive，不查 CredentialActive（标识映射不受单凭证禁用影响）
//
// 如何使用
//
// 本包（identity）只定义类型、接口和错误哨兵，不包含可执行的业务逻辑。请通过 core 子包创建入口：
//
//	import "github.com/modern-magic-go/identity/core"
//	ic := core.NewIdentityCore(myStore)
//
// 然后调用 ic.VerifyCredential / ic.GetOrInitializeSubjectID / ic.BindCredential / ic.ListCredentials。
//
// 本包不管理 Token、不持有 Session、不写数据库。只做原子凭证校验——校验成功返回 SubjectID，Token 签发由上层负责。
package identity
