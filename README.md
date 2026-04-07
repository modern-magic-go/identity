# identity

`github.com/modern-magic-go/identity` 提供可移植的身份底座能力，职责仅限于主体、身份、凭证、验证码挑战、登录审计与会话索引。

## 安装

```bash
go get github.com/modern-magic-go/identity
```

## 边界

- 不承载注册申请、用户画像、组织权限、风控策略。
- 不直接依赖项目 `internal/`、自动生成 DAO 或业务模型。
- 通过 ports 由项目层注入仓储、密码校验、验证码发送和会话签发实现。

## 目录结构

```
identity/
├── go.mod                    # go mod 架构，支持 go get 调用
├── identity.go               # 领域实体与状态定义
├── errors.go                 # 领域错误定义
├── ports.go                  # 仓储与基础设施契约（消费者需实现这些接口）
├── usecase/                  # 登录、验证码、注销等用例编排入口
│   ├── service.go
│   ├── types.go
│   ├── helpers.go
│   ├── session.go
│   ├── password_login.go
│   └── verify_challenge.go
└── internal/
    └── core/                 # 内部规则实现
        └── rules.go
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
    Subjects:           &mySubjectRepo{},
    Identities:         &myIdentityRepo{},
    Credentials:        &myCredentialRepo{},
    Challenges:         &myChallengeRepo{},
    Audits:             &myAuditRepo{},
    SessionRefs:        &mySessionRefRepo{},
    PasswordVerifier:   &myPasswordVerifier{},
    VerifyCodeGenerator: &myCodeGenerator{},
    VerifyCodeSender:   &myCodeSender{},
    VerifyCodeVerifier: &myCodeVerifier{},
    SessionManager:     &mySessionManager{},
    Clock:              &myClock{},
    IDGenerator:        &myIDGenerator{},
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
