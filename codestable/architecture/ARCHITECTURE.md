# modern-magic-go/identity 架构总入口

> 状态：活跃（2026-04-30 domain-and-crypto 首次落地）
> 最后更新：2026-04-30

## 1. 项目简介

通用身份核（Identity Core）— Headless Go 身份核库。提供原子化凭证校验能力，不包含 Token/Session 管理、用户画像存储、登录流程编排。

## 2. 核心概念 / 术语表

| 术语 | 类型 | 定义 |
|------|------|------|
| SubjectID | `= int64` | 全局唯一用户标识，由 Snowflake 算法生成 |
| Realm | `= string` | 账号池物理隔离单位（如 `c_users`、`b_admins`），不同 Realm 之间数据完全隔离 |
| IdentityType | `string` | 凭证类别枚举：PASSWORD / WECHAT_OPENID / WECHAT_UNIONID / EMAIL / TOTP / SMS |
| Credential | struct | 原子凭证：记录 subject 在某 Realm 下的某种登录方式（含哈希/密钥等敏感数据） |
| CredentialSummary | struct | 凭证摘要（脱敏，不含 CredentialData） |
| IdentityStore | interface | 持久化层合约接口，调用方注入实现 |

## 3. 子系统 / 模块索引

### 根包 `identity`（公开 API）

- `model.go` — SubjectID / Realm / IdentityType 类型定义 + Credential / CredentialSummary 结构体
- `errors.go` — 5 个哨兵错误（ErrInvalidCredential / ErrAccountLocked / ErrDuplicateCredential / ErrCredentialNotFound / ErrSubjectNotFound）
- `store.go` — IdentityStore 接口（4 方法：FindByRealmTypeIdentifier / CreateSubject / BindCredential / ListBySubjectRealm）

### `internal/idgen`（内部）

- `idgen.go` — IDGenerator 接口
- `snowflake.go` — Snowflake 实现（封装 `bwmarrin/snowflake`），`New(nodeID int64)` 构造

### `internal/crypto`（内部）

- `verifier.go` — CredentialVerifier 接口（Type / Verify）
- `bcrypt.go` — bcrypt 实现（封装 `golang.org/x/crypto/bcrypt`），暴露 `Hash()` / `Verify()` 函数 + `Bcrypt` 结构体实现 CredentialVerifier

### `usecase/`

- 当前为空，后续 feature 实现业务编排

### roadmap 规划

- `codestable/roadmap/identity-core/` — 5 条子 feature 拆解，当前 f1 `domain-and-crypto` 已完成

## 4. 关键架构决定

1. **凭证平权化**：密码、TOTP、第三方授权统一建模为 Credential，区别仅在 IdentityType
2. **Realm 隔离**：不用 TenantID / AppID，用 Realm 作为账号池物理隔离单位
3. **仓储分离**：模块定义 IdentityStore 接口，真实持久化由调用方注入
4. **Headless**：不管理 Token/Session，不存储用户画像，不编排登录流程

## 5. 已知约束 / 硬边界

- 不直接连数据库——IdentityStore 由调用方实现
- 不管理会话——验证成功后仅返回 subject_id
- 不存储用户画像（昵称/头像等）
- Go 版本 ≥ 1.25.0（受 `golang.org/x/crypto` 约束）
- 外部依赖：`bwmarrin/snowflake`（ID 生成）、`golang.org/x/crypto`（bcrypt）

## 6. 当前实现状态

| Feature | 状态 | Roadmap |
|---------|------|---------|
| domain-and-crypto（领域模型 + 密码学基础） | ✅ done | identity-core f1 |
| password-verify（密码校验编排） | ⬜ planned | identity-core f2 |
| totp-auth（TOTP 2FA） | ⬜ planned | identity-core f3 |
| credential-crud（凭证管理） | ⬜ planned | identity-core f4 |
| core-api（公共 API 组装） | ⬜ planned | identity-core f5 |
