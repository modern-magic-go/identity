# modern-magic-go/identity 架构总入口

> 状态：活跃（2026-04-30 totp-auth 完成）
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
| VerifyInput / VerifyOutput | struct | 凭证校验 API 入/出参 |
| GetOrInitSubjectInput / GetOrInitSubjectOutput | struct | 静默注册 API 入/出参 |
| MockStore | struct | 内存 IdentityStore 实现（internal/store） |
| TOTP | struct | 实现 CredentialVerifier 的 TOTP 动态码校验器（internal/crypto） |
| TOTP Secret | `string` | base32 编码的 TOTP 密钥，存储为 CredentialData |
| TOTP Code | `string` | 用户设备每 30 秒生成的 6 位数字，即 VerifyInput.InputData |

## 3. 子系统 / 模块索引

### 根包 `identity`（公开 API）

- `model.go` — SubjectID / Realm / IdentityType 类型定义 + Credential / CredentialSummary 结构体
- `errors.go` — 5 个哨兵错误（ErrInvalidCredential / ErrAccountLocked / ErrDuplicateCredential / ErrCredentialNotFound / ErrSubjectNotFound）
- `store.go` — IdentityStore 接口（4 方法：FindByRealmTypeIdentifier / CreateSubject / BindCredential / ListBySubjectRealm）
- `api.go` — VerifyInput / VerifyOutput / GetOrInitSubjectInput / GetOrInitSubjectOutput 四个 API 契约类型

### `usecase/`（业务编排）

- `verify_credential.go` — VerifyCredential 函数：查凭证 → 找 Verifier → 比对 → 返回结果
- `get_or_init_subject.go` — GetOrInitializeSubjectID 函数：查凭证 → 已有返回 / 新创建 subject → 绑凭证

### `internal/idgen`（内部）

- `idgen.go` — IDGenerator 接口
- `snowflake.go` — Snowflake 实现（封装 `bwmarrin/snowflake`），`New(nodeID int64)` 构造

### `internal/crypto`（内部）

- `verifier.go` — CredentialVerifier 接口（Type / Verify）
- `bcrypt.go` — bcrypt 实现（封装 `golang.org/x/crypto/bcrypt`），暴露 `Hash()` / `Verify()` 函数 + `Bcrypt` 结构体实现 CredentialVerifier
- `totp.go` — TOTP 实现（封装 `pquerna/otp`），`TOTP` 结构体实现 CredentialVerifier + `GenerateTOTPKey(issuer, accountName)` 生成密钥对

### `internal/store`（内部）

- `mock.go` — MockStore：内存 IdentityStore 实现，持有 IDGenerator，用于测试和演示

### roadmap 规划

- `codestable/roadmap/identity-core/` — 5 条子 feature 拆解，当前 f1-f3 已完成

## 4. 关键架构决定

1. **凭证平权化**：密码、TOTP、第三方授权统一建模为 Credential，区别仅在 IdentityType
2. **Realm 隔离**：不用 TenantID / AppID，用 Realm 作为账号池物理隔离单位
3. **仓储分离**：模块定义 IdentityStore 接口，真实持久化由调用方注入
4. **Headless**：不管理 Token/Session，不存储用户画像，不编排登录流程
5. **纯函数编排**：usecase 为独立纯函数（参数注入依赖），f5 由 IdentityCore 结构体包裹

## 5. 已知约束 / 硬边界

- 不直接连数据库——IdentityStore 由调用方实现
- 不管理会话——验证成功后仅返回 subject_id
- 不存储用户画像（昵称/头像等）
- usecase 不直接调用 IDGenerator——ID 生成由 store 实现内部持有
- CredentialVerifier 通过 map 注入——扩展新验证类型只需追加 map 条目
- TOTP secret 存储为明文 base32——加密存储由调用方仓储层负责
- Go 版本 ≥ 1.25.0（受 `golang.org/x/crypto` 约束）
- 外部依赖：`bwmarrin/snowflake`（ID 生成）、`golang.org/x/crypto`（bcrypt）、`github.com/pquerna/otp`（TOTP）

## 6. 当前实现状态

| Feature | 状态 | Roadmap |
|---------|------|---------|
| domain-and-crypto（领域模型 + 密码学基础） | ✅ done | identity-core f1 |
| password-verify（密码校验编排 + MockStore） | ✅ done | identity-core f2 |
| totp-auth（TOTP 2FA） | ✅ done | identity-core f3 |
| credential-crud（凭证管理） | ⬜ planned | identity-core f4 |
| core-api（公共 API 组装） | ⬜ planned | identity-core f5 |
