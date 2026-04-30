# core-api 验收报告

> 阶段：阶段 3（验收闭环）
> 验收日期：2026-04-30
> 关联方案 doc：codestable/features/2026-04-30-core-api/core-api-design.md

## 1. 接口契约核对

对照方案第 2.1 节名词层逐一核查：

**接口示例逐项核对**：
- [x] NewIdentityCore 构造（`core/core.go:18`）：`NewIdentityCore(store) → *IdentityCore` — 代码实际行为：一致。内建 `TypePassword → Bcrypt` + `TypeTOTP → TOTP`
- [x] VerifyCredential（`core/core.go:29`）：`core.VerifyCredential(ctx, VerifyInput{...})` → 成功/失败 — 代码实际行为：一致。委托 `usecase.VerifyCredential`
- [x] GetOrInitializeSubjectID（`core/core.go:34`）：`core.GetOrInitializeSubjectID(ctx, GetOrInitSubjectInput{...})` → 新/已有 subject — 代码实际行为：一致。委托 `usecase.GetOrInitializeSubjectID`
- [x] BindCredential（`core/core.go:39`）：`core.BindCredential(ctx, BindCredentialInput{...}) → nil / error` — 代码实际行为：一致。委托 `usecase.BindCredential`
- [x] ListCredentials（`core/core.go:44`）：`core.ListCredentials(ctx, ListCredentialsInput{...}) → []CredentialSummary, nil` — 代码实际行为：一致。委托 `usecase.ListCredentials`

**名词层"现状 → 变化"逐项核对**：
- [x] `IdentityCore` struct（`core/core.go:12`）：聚合 `store` + `verifiers` map — 代码改动一致
- [x] 4 个方法签名：均与方案一致，比 usecase 函数少 store 参数

**流程图核对**（第 2.2 节 mermaid 图）：
- [x] `CALLER → NewIdentityCore → IdentityCore` — `core/core.go:18` 落点
- [x] `IdentityCore → usecase.VerifyCredential` 等 4 条委托链 — `core/core.go:29-48` 4 方法落点
- [x] `usecase → store` — usecase 内部实现，本 feature 不碰

**偏差记录**：placement 从方案初稿的根包 `identity.go` 改为 `core/core.go`——方案 doc 已同步更新（§1.1、§2.1、§2.3、§4 均已反映 `core/` 路径）。原因：Go import cycle 约束。

## 2. 行为与决策核对

对照方案第 1 节 + 第 2.2 节：

**需求摘要逐项验证**：
- [x] IdentityCore struct 聚合 store + verifier map：`core/core.go:12-15` 实现
- [x] NewIdentityCore 构造函数：`core/core.go:18-25` 实现，内建 Bcrypt + TOTP
- [x] 4 个方法委托 usecase：各方法直接 `return usecase.Xxx(...)`，无额外逻辑

**明确不做逐项核对**：
- [x] 无新增 usecase 编排：`diff usecase/` zero change
- [x] 无新增 store 接口/方法：`diff store.go` zero change
- [x] 无新增 crypto：`diff internal/crypto/` zero change
- [x] 不暴露 CredentialVerifier：接口定义仍仅在 `internal/crypto/verifier.go`

**关键决策落地**：
- [x] D1 IDGenerator 不存于 IdentityCore：`core/core.go:13` 仅 `store` + `verifiers`，无 idgen 字段
- [x] D2 Verifier map 内建默认值：`core/core.go:22-24` `Bcrypt{}` + `TOTP{}`
- [x] D3 方法签名一致：4 方法签名与方案完全一致
- [x] D4 纯委托：每个方法仅一行 `return usecase.Xxx(...)`

**编排层"现状 → 变化"逐项核对**：
- [x] 新增 IdentityCore struct：`core/core.go:12-15`
- [x] 新增 NewIdentityCore：`core/core.go:18-25`
- [x] 新增 4 个方法：`core/core.go:29-48`

**跨层纪律核对**：无新增纪律——IdentityCore 不引入新的错误语义、幂等性约定、并发约束。

**挂载点反向核对（可卸载性）**：
- [x] M1 `core/` — IdentityCore struct：代码落点 `core/core.go:12` ✓
- [x] M2 `core/` — NewIdentityCore：代码落点 `core/core.go:18` ✓
- [x] M3 `core/` — 4 个方法：代码落点 `core/core.go:29-48` ✓
- [x] **反向核查**（grep `IdentityCore`）：15 处命中全部在 `core/core.go` 或 `core/core_test.go` 内，落在清单内
- [x] **拔除沙盘推演**：删除 `core/` 包 → feature 完全消失，无残留

## 3. 验收场景核对

对照方案第 3 节关键场景清单，逐条可观察证据验证：

- [x] **C1**：编译通过无 panic
  - 证据来源：`go build ./...` + `go vet ./...` 通过
  - 结果：通过

- [x] **C2**：VerifyCredential 正常路径
  - 证据来源：单测 `TestVerifyCredentialSuccess` @ `core/core_test.go:25`
  - 结果：通过

- [x] **C3**：GetOrInitializeSubjectID 正常路径
  - 证据来源：单测 `TestGetOrInitializeSubjectIDNew` @ `core/core_test.go:116`
  - 结果：通过

- [x] **C4**：BindCredential 正常路径
  - 证据来源：单测 `TestBindCredentialSuccess` @ `core/core_test.go:152`
  - 结果：通过

- [x] **C5**：ListCredentials 正常路径
  - 证据来源：单测 `TestListCredentialsHasItems` @ `core/core_test.go:217`
  - 结果：通过

- [x] **C6**：VerifyCredential 密码错误
  - 证据来源：单测 `TestVerifyCredentialWrongPassword` @ `core/core_test.go:64`
  - 结果：通过

- [x] **C7**：VerifyCredential 凭证未找到
  - 证据来源：单测 `TestVerifyCredentialNotFound` @ `core/core_test.go:98`
  - 结果：通过

- [x] **C8**：BindCredential 重复绑定
  - 证据来源：单测 `TestBindCredentialDuplicate` @ `core/core_test.go:185`
  - 结果：通过

- [x] **C9**：BindCredential subject 不存在
  - 证据来源：单测 `TestBindCredentialSubjectNotFound` @ `core/core_test.go:203`
  - 结果：通过

- [x] **C10**：ListCredentials 无凭证/不存在
  - 证据来源：单测 `TestListCredentialsEmpty` + `TestListCredentialsSubjectNotFound` @ `core/core_test.go:243,266`
  - 结果：通过

- [x] **C11**：PRD 5.1 双因素集成场景
  - 证据来源：单测 `TestTwoFactorAuthEndToEnd` @ `core/core_test.go:284`
  - 结果：通过（创建 subject → 绑密码 → 绑 TOTP → Verify PASSWORD 成功 → ListCredentials 发现 TOTP → Verify TOTP 成功）

**无前端改动**，跳过浏览器验证。

## 4. 术语一致性

对照方案第 0 节 + 第 2.1 节命名 grep 代码：

- `IdentityCore`：`core/core.go:11-12,17-18,28-48` 全部一致 ✓
- `NewIdentityCore`：`core/core.go:18` 构造函数 ✓
- `IdentityStore`：`core/core.go:13` 字段类型 ✓
- `CredentialVerifier`：`core/core.go:14` 仅在 `core/` 包引用 internal/crypto，未导出 ✓
- 防冲突：`grep "UpdateCredential|DeleteCredential|RemoveCredential" *.go` → zero match ✓
- 防冲突：`grep "IdentityCore" *.go` 仅在 `core/` 出现 ✓

## 5. 架构归并

对照方案第 4 节：

- [x] **名词归并**：`ARCHITECTURE.md` 术语表追加 `IdentityCore` + `NewIdentityCore` — 已写入
- [x] **模块归并**：`ARCHITECTURE.md` 新增 `core/` 模块条目 — 已写入
- [x] **实现状态**：`ARCHITECTURE.md` 状态表 `core-api` 行：`⬜ planned` → `✅ done` — 已更新
- [x] **状态行**：更新为 "core-api 完成，5/5 子 feature 全部闭环" — 已更新
- [x] **关键架构决定**：#5 更新为 `core.IdentityCore` 包裹 — 已更新
- [x] **roadmap 行**：架构索引中 roadmap 描述更新为"5 条子 feature 全部完成" — 已更新

## 6. requirement 回写

对照方案 frontmatter `requirement: prd`：

- [x] 有对应 req（PRD）且本次实现完全遵循 PRD 已定义的 IdentityCore 组装模式，PRD §4.7 的核心价值（"将所有能力暴露为方法"）已验证落地，未改边界/用户故事 → **req-prd 未变，无需更新**

## 7. roadmap 回写

对照方案 frontmatter `roadmap: identity-core` / `roadmap_item: core-api`：

- [x] `identity-core-items.yaml`：`core-api` `status: done` + `feature: 2026-04-30-core-api` — 已更新，`validate-yaml.py` 校验通过
- [x] `identity-core-roadmap.md`：第 5 节子 feature 清单 `core-api` 行标记 `✓ done` — 已同步

## 8. AGENTS.md / CLAUDE.md 候选盘点

- [x] 有候选：**Go import cycle 与根包布局约束**。当根包同时 import 自身子包（如 `usecase`）而子包又 import 根包时会产生 import cycle。本 feature 的解法是把入口 struct 放到同级 `core/` 包。但这个经验与项目强耦合（本库的 usecase 和 crypto 都 import 根包的类型），下个 feature 的 AI 大概率不会再遇到——不记入 AGENTS.md。

## 9. 遗留

- 后续优化点：无
- 已知限制：`NewIdentityCore` 不接受自定义 verifier，调用方无法扩展额外凭证类型——按方案明确不做（D2），未来 feature 可追加 `SetVerifier` 方法
- 实现阶段"顺手发现"列表：无
