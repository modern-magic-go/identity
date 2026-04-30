---
doc_type: feature-design
feature: 2026-04-30-totp-auth
status: approved
roadmap: identity-core
roadmap_item: totp-auth
requirement: prd
summary: TOTP 2FA 支持——新增 TOTP 密钥生成 + 6 位动态码验证 + TOTP 实现 CredentialVerifier 接口。不修改任何现有代码，Verifier map 注入机制使 TOTP 零成本接入 VerifyCredential
tags: [identity, totp, 2fa, crypto, verifier]
---

# totp-auth 方案

> 依赖：f1 `domain-and-crypto`（done）+ f2 `password-verify`（done）
> 来源：`codestable/roadmap/identity-core/` items.yaml 第 3 条
> roadmap 含硬约束接口契约——设计必须遵循第 4 节 `CredentialVerifier` 接口，改动需先回 roadmap update

## 0. 术语 / 口径

- **TOTP**: Time-based One-Time Password（基于时间的动态口令）。密钥 + 当前时间步 → HMAC → 截断 → 6 位数字
- **TOTP Secret**: base32 编码的密钥，对战前服务器生成并分发给用户设备（扫码或手动输入）。存储为 `CredentialData`
- **TOTP Code**: 用户设备（Google Authenticator / Authy 等）每 30 秒生成的最新 6 位数字，即 `VerifyInput.InputData`
- **issuer / accountName**: 用于生成 `otpauth://` URL 的元信息——issuer 是应用名（如 "MyApp"），accountName 是账户名（如 "alice@example.com"）

## 1. 决策与约束

### 1.1 做什么

- 新增 `internal/crypto/totp.go` — TOTP 密钥生成 + 动态码验证
- 新增 TOTP 外部库依赖 `github.com/pquerna/otp`
- 扩展 usecase 测试——加入 TypeTOTP 的 verifier 映射条目 + 2FA 场景测试
- **不修改** `VerifyCredential` / `GetOrInitializeSubjectID` 编排代码——verifier map 注入已支持白盒扩展

### 1.2 不做什么

- 不实现 QR 码图片生成——调用方拿到 `otpauth://` URL 后自行生成 QR 码
- 不实现 TOTP 密钥的加密存储——`CredentialData` 为明文 base32 密钥，加密是调用方仓储层的责任
- 不导出 TOTP 相关类型到公开 API——TOTP 实现放在 `internal/crypto`，仅通过 `CredentialVerifier` 接口暴露
- 不修改 `model.go` / `store.go` / `api.go` / `errors.go`——`TypeTOTP` 常量已在 f1 声明，所有现有接口对 TOTP 通用
- 不实现 TOTP 的 skew（时间窗口偏斜）自定义——使用 `pquerna/otp` 默认值（±1 step，即 ±30s）
- 不实现 HOTP（基于计数器）——PRD 只提 TOTP

### 1.3 复杂度档位

无偏离——走库项目默认组合：L3（严防）+ modules + reasonable + public + stable + traced + tested + validated。

唯一观察项：可读性 `public` 需要在 `GenerateTOTPKey` 函数签名上补充 godoc 示例。

### 1.4 关键决定

- D1：**TOTP secret 存储为明文 base32**。标准 TOTP 实现中 secret 本身不是密码——代码基于 secret+time 计算得出。Server-side 无需对 secret 做哈希（不同于 bcrypt 密码存储）。加密存储由调用方仓储层负责，本库只定义格式
- D2：**密钥生成与验证分离**。`GenerateTOTPKey(issuer, accountName)` 生成 secret + URL；`TOTP.Verify(storedData, inputData)` 仅做验证。调用方负责：调用 GenerateTOTPKey → 存储 secret 到 CredentialData → 向用户展示 URL（作为 QR 码）→ 用户输入 code → 调 VerifyCredential
- D3：**不修改 usecase 代码**（这是本 feature 最关键的约束）。f2 设计的 `map[IdentityType]CredentialVerifier` 注入模式已经保证了白盒扩展性——添加 TOTP verifier 是调用方的配置行为，不是 usecase 的修改行为

## 2. 现状 → 变化

### 2.1 名词层

#### 现状

| 条目 | 位置 | 现状 |
|------|------|------|
| `TypeTOTP` | `model.go:17` | `IdentityType = "TOTP"` 常量（f1 声明，尚未有对应的 Verifier） |
| `CredentialVerifier` | `internal/crypto/verifier.go:6-8` | 接口：`Type() IdentityType` + `Verify(storedData, inputData string) (bool, error)` |
| `Bcrypt` | `internal/crypto/bcrypt.go:12` | 唯一已实现的 Verifier（`TypePassword`） |
| `Credential.CredentialData` | `model.go:27` | `string` 字段——用于 bcrypt 哈希，同样可用于 TOTP secret |
| `VerifyInput.InputData` | `api.go:4-8` | `string` 字段——用于明文密码，同样用于 TOTP 6 位码 |

#### 变化

| 条目 | 变化 | 位置（新） |
|------|------|-----------|
| `TOTP` struct | 新增，实现 `CredentialVerifier`，`Type()` 返回 `TypeTOTP` | `internal/crypto/totp.go` |
| `GenerateTOTPKey(issuer, accountName string) (secret string, url string, err error)` | 新增公开函数，生成 TOTP 密钥对 | 同上 |

**接口契约**（来自 roadmap §4.6，硬约束）：

```go
// TOTP 实现 CredentialVerifier，用于 TOTP 动态码校验
type TOTP struct{}

func (t *TOTP) Type() identity.IdentityType { return identity.TypeTOTP }

// Verify 验证 TOTP 码
// storedData = base32 编码的 TOTP secret（即 CredentialData）
// inputData  = 用户输入的 6 位数字码
func (t *TOTP) Verify(storedData, inputData string) (bool, error)
```

**密钥生成函数**（新增，非接口约束）：

```go
// GenerateTOTPKey 生成 TOTP 密钥对
// issuer: 应用名，如 "MyApp"（出现在 Google Authenticator 标题）
// accountName: 账户标识，如 "alice@example.com"
// 返回: secret (base32 密钥，存储到 CredentialData), url (otpauth://totp/...，生成 QR 码用), error
func GenerateTOTPKey(issuer, accountName string) (secret string, url string, err error)
```

不新增根包导出类型——TOTP 是内部实现，通过 `CredentialVerifier` 接口暴露。

### 2.2 编排层

#### 主流程图（VerifyCredential 在 TOTP 类型上的行为——0 行新增代码）

```
调用方
  │
  ├─ 1. 注册 Verifier: verifiers[TypeTOTP] = &crypto.TOTP{}
  │
  └─ 2. 调 VerifyCredential(ctx, store, verifiers, VerifyInput{
         Realm:        "admins",
         IdentityType: TypeTOTP,
         Identifier:   "totp_device",
         InputData:    "837261",  ← 用户输入的 6 位 TOTP
       })
            │
            ▼
     VerifyCredential（无修改，f2 现成）
       ├─ FindByRealmTypeIdentifier → Credential{CredentialData: "JBSWY3DPEHPK3PXP"}  ← base32 secret
       ├─ verifiers[TypeTOTP] → &crypto.TOTP{}  ← map 查表
       ├─ TOTP.Verify("JBSWY3DPEHPK3PXP", "837261") → (true, nil)
       └─ 返回 VerifyOutput{Success: true, SubjectID: 888}
```

**编排层无变化**。`VerifyCredential` 的 find→lookup→verify→return 四步拓扑对 TOTP 与 PASSWORD 完全相同，区别仅在 Verifier 实例。f2 的 map 注入设计使 TOTP 零成本接入。

#### 端到端 2FA 场景（user code 视角，由测试演示）

```
1. GenerateTOTPKey("MyApp", "alice") → ("JBSWY3DPEHPK3PXP", "otpauth://totp/...", nil)
2. CreateSubject → subjectID=888
3. BindCredential(SubjectID=888, Type=TOTP, CredentialData="JBSWY3DPEHPK3PXP")
4. （用户扫描 otpauth:// QR 码，Google Authenticator 开始生成 6 位码）
5. VerifyCredential(Type=TOTP, InputData="当前码") → Success=true
```

### 2.3 挂载点

按"删了它 feature 是否消失"判据：

| 编号 | 挂载点 | 类型 | 说明 |
|------|--------|------|------|
| M1 | `internal/crypto/totp.go` | 新增 | TOTP struct + GenerateTOTPKey 函数。删除后 TOTP 校验能力消失 |
| M2 | `go.mod` | 变动 | `require github.com/pquerna/otp vX.Y.Z`。删除后编译失败 |

**非挂载点**（内部改动，删了 feature 仍完整）：usecase 测试扩展（属于验证层）、`go.sum`（自动生成）。

**拔除沙盘推演**：删除 `internal/crypto/totp.go` + 回退 go.mod → grep 不含 `TypeTOTP` Verifier 引用 → feature 消失。

### 2.4 推进策略

按 paradigm 维度切片：

| 步 | 维度 | 内容 | 退出信号 |
|----|------|------|----------|
| Step 1 | 计算节点（TOTP Verifier） | 新增 `internal/crypto/totp.go`：TOTP struct + GenerateTOTPKey；`go get github.com/pquerna/otp`；实现 `Type()` 和 `Verify()` | `TOTP` 通过 `implements CredentialVerifier` 编译断言 |
| Step 2 | 测试（单元测试） | 新增 `internal/crypto/totp_test.go`：密钥生成正确性、Verify 正确码通过、Verify 错误码拒绝 | `go test ./internal/crypto/...` 通过 |
| Step 3 | 测试（编排层集成） | 扩展 `usecase/usecase_test.go`：verifier map 加 TypeTOTP → 创建 subject → 生成 TOTP 密钥 → BindCredential → VerifyCredential 校验 TOTP → 成功；错误码拒绝 | `go test ./usecase/...` 通过 |
| Step 4 | 2FA 场景（end-to-end test） | 新增测试：同一 subject 绑 PASSWORD + TOTP → 分别验证 → 双因子场景完整性 | 测试通过 |
| Step 5 | 回归 | `go test ./...` + `validate-yaml` | 全部通过 |

## 3. 验收契约

每条写成"输入 / 触发 → 期望可观察结果"。

| 编号 | 场景 | 输入 / 触发 | 期望结果 |
|------|------|------------|----------|
| S1 | 生成 TOTP 密钥 | `GenerateTOTPKey("MyApp", "alice")` | 返回 non-empty secret（base32, 长度 ≥ 16）、non-empty URL（以 `otpauth://totp/` 开头） |
| S2 | 验证正确的 TOTP 码 | `TOTP.Verify(secret, totp.GenerateCode(secret, time.Now()))` | 返回 `(true, nil)` |
| S3 | 验证错误的 TOTP 码 | `TOTP.Verify(secret, "000000")`（对非恰好的 secret 大概率不对） | 返回 `(false, nil)` |
| S4 | VerifyCredential 集成——TOTP 正确码 | verifier map 含 TOTP → 创建 subject → Bind TOTP → VerifyCredential(InputData=正确码) | `Success=true` + SubjectID 匹配 |
| S5 | VerifyCredential 集成——TOTP 错误码 | 同上，InputData="000000" | `Success=false` + `ErrorCode=INVALID_CREDENTIAL` |
| S6 | TOTP 类型未注册 Verifier | verifier map 不含 TypeTOTP → VerifyCredential(Type=TOTP, ...) | `Success=false` + `ErrorCode=UNSUPPORTED_TYPE` |
| S7 | 2FA 场景：密码 + TOTP 双因素 | 同一 subject 绑 PASSWORD + TOTP → 分别验证 | 两次 VerifyCredential 均 Success=true |
| S8 | 不同 Realm 的 TOTP 隔离 | Realm="A" 下绑 TOTP → Realm="B" 查不到 | `ErrorCode=CREDENTIAL_NOT_FOUND` |

**明确不做反向核对**（与第 1.2 节对应）：

| 不做项 | 核对方式 |
|--------|----------|
| 无 TOTP 类型导出到根包 | grep `identity\.TOTP` 或 `type TOTP` 在非 `internal/crypto/totp.go` → zero match |
| 无 usecase 代码修改 | diff `usecase/verify_credential.go` + `usecase/get_or_init_subject.go` → zero diff |
| 无 model.go / store.go / api.go / errors.go 修改 | diff 这些文件 → zero diff |
| 无 QR 码生成代码 | grep `image` / `qrcode` / `PNG` / `Encode` → zero match in non-test |
| 无 HOTP 实现 | grep `HOTP` / `hotp` / `counter` → zero match |

## 4. 架构升级

需在 acceptance 阶段归并到 `ARCHITECTURE.md`：

- §2 术语表：新增 TOTP / TOTP Secret / TOTP Code 条目
- §3 `internal/crypto`：新增 `totp.go` — TOTP 结构体（实现 CredentialVerifier）+ GenerateTOTPKey 密钥生成函数
- §5 约束：新增 TOTP 依赖 `github.com/pquerna/otp`
