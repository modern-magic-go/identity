# identity

`github.com/modern-magic-go/identity` 提供可移植的身份底座能力，当前聚焦主体、身份、凭证与密码登录编排。

## 安装

```bash
go get github.com/modern-magic-go/identity
```

## 边界

- 不承载注册申请、用户画像、组织权限、风控策略。
- 不直接依赖项目 `internal/`、自动生成 DAO 或业务模型。
- 通过 ports 由项目层注入仓储、密码校验、时间源与标识生成实现。

## 模块架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        消费项目 (Consumer)                       │
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────────────┐ │
│  │ SubjectRepo  │  │ IdentityRepo │  │ PasswordCredentialRepo│ │
│  │ (DAO/SQL)    │  │ (DAO/SQL)    │  │ (DAO/SQL)             │ │
│  └──────┬───────┘  └──────┬───────┘  └──────────┬────────────┘ │
│         │                 │                      │              │
│  ┌──────▼─────────────────▼──────────────────────▼────────────┐ │
│  │              Ports 接口实现 (Consumer 提供)                 │ │
│  │  SubjectRepository │ IdentityRepository │ CredentialRepo   │ │
│  │  PasswordVerifier  │ Clock              │ IDGenerator      │ │
│  └──────────────────────────┬─────────────────────────────────┘ │
│                             │ 依赖注入                          │
├─────────────────────────────┼───────────────────────────────────┤
│                             ▼                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │              github.com/modern-magic-go/identity            │ │
│  │                                                            │ │
│  │  ┌──────────────────────────────────────────────────────┐  │ │
│  │  │  usecase/ (公共用例层)                                │  │ │
│  │  │                                                      │  │ │
│  │  │  ┌─────────────────────────────────────────────────┐ │  │ │
│  │  │  │  Service.PasswordLogin()                        │ │  │ │
│  │  │  │                                                 │ │  │ │
│  │  │  │  1. 规范化标识符                                 │ │  │ │
│  │  │  │  2. 查找身份映射 IdentityRepository             │ │  │ │
│  │  │  │  3. 校验主体状态 Subject.IsLoginable()           │ │  │ │
│  │  │  │  4. 查找密码凭证 PasswordCredentialRepository    │ │  │ │
│  │  │  │  5. 校验凭证可用性 PasswordCredential.IsUsable() │ │  │ │
│  │  │  │  6. 密码比对 PasswordVerifier.VerifyPassword()   │ │  │ │
│  │  │  │  7. 失败计数递增 (密码错误时)                     │ │  │ │
│  │  │  │  8. 返回 SubjectView                            │ │  │ │
│  │  │  └─────────────────────────────────────────────────┘ │  │ │
│  │  │                                                      │  │ │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌──────────────┐  │  │
│  │  │  │ types.go    │  │ helpers.go  │  │ service.go   │  │  │
│  │  │  │ Input/Output│  │ 辅助方法    │  │ 组装与校验   │  │  │
│  │  │  └─────────────┘  └─────────────┘  └──────────────┘  │  │
│  │  └──────────────────────────────────────────────────────┘  │ │
│  │                                                            │ │
│  │  ┌──────────────────────────────────────────────────────┐  │ │
│  │  │  根包 identity (门面)                                 │  │ │
│  │  │                                                      │  │ │
│  │  │  ┌───────────────┐  ┌───────────────┐  ┌───────────┐ │  │ │
│  │  │  │ identity.go   │  │ errors.go     │  │ ports.go  │ │  │ │
│  │  │  │ 领域实体      │  │ 领域错误      │  │ 接口契约  │ │  │ │
│  │  │  │ 状态枚举      │  │ 14 个错误定义  │  │ 7 个接口  │ │  │ │
│  │  │  └───────────────┘  └───────────────┘  └───────────┘ │  │ │
│  │  └──────────────────────────────────────────────────────┘  │ │
│  │                                                            │ │
│  │  ┌──────────────────────────────────────────────────────┐  │ │
│  │  │  internal/core (内部规则)                              │  │ │
│  │  │  rules.go — 状态校验 / 标识脱敏 / 可用性检查          │  │ │
│  │  └──────────────────────────────────────────────────────┘  │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## 密码登录流程

```
PasswordLogin(input)
    │
    ▼
┌─────────────────────┐
│ 规范化标识符         │  identity.NormalizeIdentifier()
│ "Admin_User" →       │
│ "admin_user"         │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ 查找身份映射         │  IdentityRepository.FindByLoginIdentity()
│ realm + provider +   │
│ identityType + ident │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐     ┌─────────────────────┐
│ 校验身份可用性       │ NO  │ ErrIdentity         │
│ Identity.IsAvailable()├──►│ Unavailable         │
└──────────┬──────────┘     └─────────────────────┘
           │ YES
           ▼
┌─────────────────────┐
│ 查找主体             │  SubjectRepository.GetByID()
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐     ┌─────────────────────┐
│ 校验主体可登录       │ NO  │ ErrSubject          │
│ Subject.IsLoginable()├──►│ NotLoginable        │
│ (仅 active 允许)     │     │                     │
└──────────┬──────────┘     └─────────────────────┘
           │ YES
           ▼
┌─────────────────────┐
│ 查找密码凭证         │  PasswordCredentialRepository
│                      │  .FindBySubjectAndRealm()
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐     ┌─────────────────────┐
│ 校验凭证可用性       │ NO  │ ErrCredential       │
│ Credential.IsUsable()├──►│ Unavailable         │
│ (状态/锁定/哈希)     │     │                     │
└──────────┬──────────┘     └─────────────────────┘
           │ YES
           ▼
┌─────────────────────┐     ┌─────────────────────┐
│ 密码比对             │ FAIL│ IncrementFailedCount│
│ PasswordVerifier     │     │ (失败计数+1)        │
│ .VerifyPassword()    │     └─────────────────────┘
└──────────┬──────────┘
           │ PASS
           ▼
┌─────────────────────┐
│ 返回 LoginResult     │  SubjectView {
│                      │    SubjectID, SubjectNo,
│                      │    SubjectType, Realm
│                      │  }
└─────────────────────┘
```

## 能力清单

| 能力 | 说明 | 位置 |
|---|---|---|
| **密码登录编排** | 身份查找 → 主体校验 → 凭证校验 → 密码比对 | `usecase/password_login.go` |
| **主体状态机** | `pending_activation → active → frozen → deactivating → deactivated` | `identity.go` |
| **身份可用性判断** | `active` 状态 + 标识符非空 | `identity.go:Identity.IsAvailable()` |
| **凭证可用性判断** | `active` 状态 + 未锁定 + 哈希非空 | `identity.go:PasswordCredential.IsUsable()` |
| **密码失败计数** | 密码错误时自动递增失败次数 | `ports.go:PasswordCredentialRepository` |
| **标识符规范化** | `TrimSpace + ToLower` 统一处理 | `identity.go:NormalizeIdentifier()` |
| **标识符脱敏** | 手机号/邮箱等脱敏规则 | `internal/core/rules.go:MaskIdentifier()` |
| **时间源注入** | 支持测试时钟注入 | `ports.go:Clock` |

## 领域模型

```
Subject (主体)
  ├── 1:N Identity (身份映射)
  │     ├── realm: admin / workbench / ...
  │     ├── provider: password / sms / ...
  │     └── identityType: username / mobile / ...
  │
  └── 1:1 PasswordCredential (密码凭证)
        ├── PasswordHash / PasswordAlgo
        ├── FailedCount / LockedUntil
        └── Status: active / blocked / revoked

  └── 1:1 TotpCredential (TOTP 凭证，可选)
        ├── CredentialValue
        └── CredentialMeta (JSON: issuer, account)

  └── N:1 VerifyChallenge (验证码挑战，领域类型)
        ├── ChallengeID / VerifyCode
        ├── MaxAttempt / UsedCount
        └── Status: pending / verified / expired / locked
```

## 目录结构

```
identity/
├── go.mod                    # go mod 架构，支持 go get 调用
├── identity.go               # 领域实体与状态定义
├── errors.go                 # 领域错误定义
├── ports.go                  # 仓储与基础设施契约（消费者需实现这些接口）
├── usecase/                  # 密码登录用例编排入口
│   ├── service.go            # 服务组装与依赖校验
│   ├── types.go              # Input/Output 类型定义
│   ├── helpers.go            # 辅助方法
│   └── password_login.go     # 密码登录核心流程
└── internal/
    └── core/                 # 内部规则实现（外部不可见）
        └── rules.go          # 状态校验 / 标识脱敏
```

## 使用方式

```go
import (
    "github.com/modern-magic-go/identity"
    "github.com/modern-magic-go/identity/usecase"
)

// 实现 ports 定义的接口
type mySubjectRepo struct{}
func (r *mySubjectRepo) GetByID(ctx context.Context, id int64) (*identity.Subject, error) { ... }
// ... 实现其他接口

// 创建服务
svc, err := usecase.NewService(usecase.Config{
    Subjects:         &mySubjectRepo{},
    Identities:       &myIdentityRepo{},
    Credentials:      &myCredentialRepo{},
    PasswordVerifier: &myPasswordVerifier{},
    Clock:            &myClock{},
    IDGenerator:      &myIDGenerator{},
})

// 密码登录
result, err := svc.PasswordLogin(ctx, identity.PasswordLoginInput{
    Realm:      "admin",
    Identifier: "admin_user",
    Password:   "123456",
    Platform:   "web",
    DeviceID:   "dev-1",
})
```
