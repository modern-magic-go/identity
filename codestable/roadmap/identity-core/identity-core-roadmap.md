---
doc_type: roadmap
slug: identity-core
status: active
created: 2026-04-30
last_reviewed: 2026-04-30
summary: 通用身份核（Identity Core）Go 库——Headless 原子凭证校验底座，支持密码/TOTP/第三方登录的凭证绑定与校验
tags: [identity, auth, go-library, credential, realm]
---

# Identity Core Roadmap

> 来源：`codestable/requirements/prd.md` + `cs-brainstorm` case 3 移交讨论

## 1. 背景与动机

构建一个 Headless（无 UI、无 Session）的 Go 身份核库。上层服务（Auth Service / Gateway）持有 Token 管理和流程编排逻辑，本库只提供原子化的凭证校验能力。

核心价值：把"外部标识（手机号 / 微信 / 账号）→ 系统内实体（subject_id）"的映射和"凭证（密码 / TOTP）→ 合法/非法"的校验封装为可复用 Go 库。

## 2. 范围与明确不做

### 做

- 身份映射：外部标识 → subject_id（`GetOrInitializeSubjectID`）
- 凭证校验：密码 bcrypt 比对、TOTP code 验证（`VerifyCredential`）
- 凭证管理：绑定新凭证、列出已有凭证（`BindCredential` / `ListCredentials`）
- Realm 多领域隔离：同一标识在不同 Realm 下对应不同 subject_id
- Repository 抽象：`IdentityStore` 接口，调用方注入实现
- Snowflake 全局 ID 生成

### 明确不做

- Token / Session 管理——验证成功仅返回 subject_id + 状态，Token 签发由上层负责
- 用户画像（昵称、头像、性别等）——本库不存储任何业务属性
- 直接数据库写入——本库只定义数据模型和仓储接口，真实落库由调用方实现
- 登录流程编排——本库只提供原子校验，不决定"某个端用什么方式登录"
- OAuth / 第三方授权服务器交互——本库只校验已拿到的凭证（如微信 OpenID），不负责去微信服务器换 token

## 3. 模块拆分（概设）

| 模块 | 职责 | 一句话描述 |
|------|------|-----------|
| Domain Model | 核心类型 + 错误哨兵 | 全库共享的基础契约：SubjectID、Realm、IdentityType、Credential、CredentialSummary |
| IdentityStore 接口 | 持久化层合约 | 4 方法按业务操作命名（FindByRealmTypeIdentifier / CreateSubject / BindCredential / ListBySubjectRealm） |
| ID Generator | Snowflake ID 生成 | 封装 `bwmarrin/snowflake`，提供全局唯一 subject_id |
| Crypto | 密码学适配 | bcrypt 哈希/比对 + TOTP 密钥生成/验证，统一实现 `CredentialVerifier` 接口 |
| Use Cases | 业务编排 | 4 个核心 API 的编排逻辑，依赖上面 4 个模块 |
| Public API（根包） | 对外入口 | `IdentityCore` 结构体 + `NewIdentityCore` 构造函数，将所有能力暴露为方法 |

### 依赖方向

```
Domain Model （零依赖，纯类型）
    ├── IdentityStore 接口    ← Domain Model
    ├── ID Generator          ← Domain Model
    ├── Crypto                ← Domain Model
    └── Use Cases             ← Domain Model + IdentityStore + ID Generator + Crypto
            └── Public API    ← Use Cases（注入 IdentityStore）
```

### 包结构约定（骨架，子 feature 实现时落地）

```
identity/
├── identity.go              # IdentityCore 结构体 + NewIdentityCore 构造函数（根包公共 API）
├── model.go                 # 公开类型（SubjectID, Realm, IdentityType, Credential, CredentialSummary）
├── store.go                 # IdentityStore 接口定义
├── errors.go                # 哨兵错误
├── internal/
│   ├── crypto/
│   │   ├── bcrypt.go        # bcrypt 哈希/比对
│   │   └── totp.go          # TOTP 密钥生成/验证
│   ├── idgen/
│   │   └── snowflake.go     # Snowflake ID 生成器
│   └── store/
│       └── mock.go          # 内存 Mock Store（测试用）
└── usecase/
    ├── verify_credential.go
    ├── get_or_init_subject.go
    ├── bind_credential.go
    └── list_credentials.go
```

## 4. 接口契约（架构层详设）

以下接口是所有子 feature 的**硬约束**——feature-design 必须遵守，需要改动先回 roadmap update。

### 4.1 公共类型（根包导出）

```go
// SubjectID 全局唯一用户标识，由 Snowflake 算法生成
type SubjectID = int64

// Realm 领域/命名空间，账号池的物理隔离单位
type Realm = string

// IdentityType 凭证类型
type IdentityType string

const (
    TypePassword      IdentityType = "PASSWORD"
    TypeWechatOpenID  IdentityType = "WECHAT_OPENID"
    TypeWechatUnionID IdentityType = "WECHAT_UNIONID"
    TypeEmail         IdentityType = "EMAIL"
    TypeTOTP          IdentityType = "TOTP"
    TypeSMS           IdentityType = "SMS"
)
```

### 4.2 数据模型

```go
type Credential struct {
    SubjectID      int64        `json:"subject_id"`
    Realm          string       `json:"realm"`
    IdentityType   IdentityType `json:"identity_type"`
    Identifier     string       `json:"identifier"`
    CredentialData string       `json:"credential_data"` // 加密存储的凭证（bcrypt hash / TOTP secret），第三方登录可为空
}

type CredentialSummary struct {
    Type       IdentityType `json:"type"`
    Identifier string       `json:"identifier"`
}
```

### 4.3 IdentityStore 接口

```go
type IdentityStore interface {
    // FindByRealmTypeIdentifier 按 Realm + 类型 + 标识符查找凭证
    FindByRealmTypeIdentifier(ctx context.Context, realm string, identityType IdentityType, identifier string) (*Credential, error)

    // CreateSubject 创建新的用户主体，返回 Snowflake 生成的 subject_id
    CreateSubject(ctx context.Context) (int64, error)

    // BindCredential 将凭证绑定到指定 subject
    BindCredential(ctx context.Context, cred *Credential) error

    // ListBySubjectRealm 列出 subject 在指定 Realm 下的所有凭证（不含敏感数据）
    ListBySubjectRealm(ctx context.Context, subjectID int64, realm string) ([]CredentialSummary, error)
}
```

### 4.4 API 输入/输出类型

```go
// --- VerifyCredential ---

type VerifyInput struct {
    Realm        string       // 领域
    IdentityType IdentityType // 凭证类型
    Identifier   string       // 标识符（手机号/用户名/微信OpenID）
    InputData    string       // 用户输入的验证物（明文密码/TOTP Code），第三方授权可为空
}

type VerifyOutput struct {
    Success   bool   // 校验是否通过
    SubjectID int64  // 仅在 Success=true 时有效
    ErrorCode string // 错误码：ACCOUNT_LOCKED / INVALID_CREDENTIAL / CREDENTIAL_NOT_FOUND
    ErrorMsg  string // 人类可读错误描述
}

// --- GetOrInitializeSubjectID ---

type GetOrInitSubjectInput struct {
    Realm        string       // 领域
    IdentityType IdentityType // 凭证类型
    Identifier   string       // 标识符
}

type GetOrInitSubjectOutput struct {
    SubjectID int64 // 已有或新创建的 subject_id
    IsNewUser bool  // 是否为新注册用户
}

// --- BindCredential ---

type BindCredentialInput struct {
    SubjectID      int64        // 目标 subject
    Realm          string       // 领域
    IdentityType   IdentityType // 凭证类型
    Identifier     string       // 标识符
    CredentialData string       // 需加密存储的凭证数据
}

// --- ListCredentials ---

type ListCredentialsInput struct {
    SubjectID int64  // 目标 subject
    Realm     string // 领域
}
```

### 4.5 错误哨兵

```go
var (
    ErrInvalidCredential   = errors.New("identity: invalid credential")
    ErrAccountLocked       = errors.New("identity: account locked")
    ErrDuplicateCredential = errors.New("identity: duplicate credential already exists in realm")
    ErrCredentialNotFound  = errors.New("identity: credential not found")
    ErrSubjectNotFound     = errors.New("identity: subject not found")
)
```

调用方通过 `errors.Is(err, ErrInvalidCredential)` 判断错误类型。

### 4.6 内部接口（模块内部使用，不导出）

```go
// IDGenerator 全局 ID 生成器（内部接口）
type IDGenerator interface {
    Generate() int64
}

// CredentialVerifier 凭证校验策略（内部接口，按 IdentityType 注册）
type CredentialVerifier interface {
    Type() IdentityType
    Verify(storedData, inputData string) (bool, error)
}
```

- `IDGenerator` 由 Snowflake 实现
- `CredentialVerifier` 由 bcrypt（PASSWORD）和 TOTP（TOTP）分别实现
- 其他类型（WECHAT_OPENID 等）不注册 Verifier——此类凭证的"校验"由调用方在外部完成，模块只负责存储和映射

### 4.7 IdentityCore 公共 API（最终形态，f5 组装后暴露）

```go
type IdentityCore struct {
    // 内部组合：IdentityStore + IDGenerator + map[IdentityType]CredentialVerifier
}

func NewIdentityCore(store IdentityStore) *IdentityCore

func (c *IdentityCore) VerifyCredential(ctx context.Context, input VerifyInput) (VerifyOutput, error)
func (c *IdentityCore) GetOrInitializeSubjectID(ctx context.Context, input GetOrInitSubjectInput) (GetOrInitSubjectOutput, error)
func (c *IdentityCore) BindCredential(ctx context.Context, input BindCredentialInput) error
func (c *IdentityCore) ListCredentials(ctx context.Context, input ListCredentialsInput) ([]CredentialSummary, error)
```

## 5. 子 feature 拆解清单

| # | slug | 描述 | 依赖 | 最小闭环 |
|---|------|------|------|----------|
| 1 | `domain-and-crypto` | 领域模型（SubjectID/Realm/IdentityType/Credential/CredentialSummary）+ 错误哨兵 + IdentityStore 接口 + IDGenerator 接口 + CredentialVerifier 接口 + Snowflake 实现 + bcrypt 实现 | 无 | ✓ done |
| 2 | `password-verify` | VerifyCredential（密码校验 + 凭证查找编排）+ GetOrInitializeSubjectID（静默注册编排）+ 内存 Mock IdentityStore + 端到端测试：创建主体→绑定密码→校验凭证 | f1 | ✓ done |
| 3 | `totp-auth` | TOTP 密钥生成 + code 验证 + 注册 TOTP 到 CredentialVerifier 策略 + 扩展 VerifyCredential 支持 TOTP 类型 + 2FA 场景测试 | f1, f2 | |
| 4 | `credential-crud` | BindCredential 编排（含唯一性校验）+ ListCredentials 编排（含敏感数据脱敏）+ 完整凭证生命周期测试 | f1 | |
| 5 | `core-api` | IdentityCore 结构体 + NewIdentityCore 构造函数 + 根包公开 API（暴露 4 个方法）+ 集成测试（PRD 5.1 场景：密码+TOTP 双因素登录完整流程） | f2, f3, f4 | |

### 依赖图

```
f1 ──→ f2 ──→ f3 ──→ f5
│                     ↗
└──→ f4 ─────────────┘
```

- f1 无依赖，可立即启动
- f2 依赖 f1；f4 仅依赖 f1（与 f2 可并行）
- f3 依赖 f1 + f2（需要 VerifyCredential 的 Verifier 注册机制）
- f5 是收束点，依赖所有 usecase

## 6. 排期建议

技术依赖之外的排序由用户决定。技术依赖上：

1. **f1 `domain-and-crypto`** 无依赖，最先做——它是所有其他子 feature 的基石
2. f1 完成后，**f2 `password-verify`** 和 **f4 `credential-crud`** 可并行推进。f2 是最小闭环，建议优先
3. f2 完成后启动 **f3 `totp-auth`**
4. f2 + f3 + f4 全部完成后，**f5 `core-api`** 收束组装

## 7. 观察项

- `go.mod` 当前依赖为 testify / mysql / redis，PRD 推荐的 snowflake、bcrypt、TOTP 三个库尚未添加。f1 实现时需要同步更新 go.mod
- 本库定义数据模型 `user_subject` / `identity_credential` 但不直接落库——调用方仓储实现需要遵守 PRD 第 3 节的表结构和唯一约束
- `architecture/ARCHITECTURE.md` 当前为骨架状态，f2 完成（最小闭环可演示）后建议由 acceptance 回写架构 doc
