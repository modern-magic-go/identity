// Package identity 提供 Headless 身份核能力——把外部标识（手机号/用户名/微信 OpenID）到系统内实体的映射和凭证（密码/TOTP）校验封装为可复用的原子库。
//
// 核心概念
//
//   - SubjectID：全局唯一用户标识（type SubjectID string），兼容 Snowflake int64 和 UUID 两种 ID 体系。
//     构造通道：SubjectIDFromInt64(id int64) / SubjectIDFromString(id string)
//   - Realm：领域/命名空间（string），账号池的物理隔离单位，同一标识在不同 Realm 下对应不同 SubjectID
//   - IdentityType：凭证类型，内置 PASSWORD / TOTP / WECHAT_OPENID / WECHAT_UNIONID / EMAIL / SMS
//   - Credential：原子凭证，记录一个 Subject 在某 Realm 下的一种登录方式。
//     包含 IsActive 字段，查询时由 store 层附带，false 表示该 Subject 已被禁止认证（冻结/注销/风控）
//   - CredentialSummary：凭证摘要，脱敏后不含 CredentialData，含 IsActive，安全返回给调用方
//   - TransactionalStore：可选接口扩展，支持事务内执行多步 IdentityStore 操作。实现者可选满足
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
