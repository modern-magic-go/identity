---
doc_type: feature-acceptance
feature: 2026-04-30-password-verify
status: completed
summary: password-verify 验收通过——接口契约 0 偏差、12 项验收场景全部通过、架构已归并、roadmap 已回写。最小闭环：创建主体→绑定密码→校验凭证可端到端演示
tags: [identity, password, verify, usecase, mock]
---

# password-verify 验收报告

> 阶段：阶段 3（验收闭环）
> 验收日期：2026-04-30
> 关联方案 doc：`codestable/features/2026-04-30-password-verify/password-verify-design.md`

## 1. 接口契约核对

对照方案第 2.1 节名词层逐一核查：

**名词层"现状 → 变化"逐项核对**：
- [x] `VerifyInput` struct（4 字段） → `api.go:4` ✓
- [x] `VerifyOutput` struct（4 字段） → `api.go:12` ✓
- [x] `GetOrInitSubjectInput` struct（3 字段） → `api.go:20` ✓
- [x] `GetOrInitSubjectOutput` struct（2 字段） → `api.go:27` ✓
- [x] `MockStore` struct（idGen + subjects + creds） → `internal/store/mock.go:12-16` ✓
- [x] `NewMockStore(idGen) *MockStore` → `mock.go:19-24` ✓
- [x] `FindByRealmTypeIdentifier` → 返回 `ErrCredentialNotFound` → `mock.go:32-37` ✓
- [x] `CreateSubject` → 调 `idGen.Generate()` → `mock.go:41-44` ✓
- [x] `BindCredential` → 检查唯一性 + subject 存在性 → `mock.go:48-59` ✓
- [x] `ListBySubjectRealm` → 按 subjectID+realm 筛选 → `mock.go:63-73` ✓

**D1 函数签名核对**：
- [x] `VerifyCredential(ctx, store, verifiers map, input) (output, error)` → `verify_credential.go:12-17` ✓
- [x] `GetOrInitializeSubjectID(ctx, store, input) (output, error)` → `get_or_init_subject.go:11-15` ✓

**流程图核对**（设计 §2.2 mermaid）：
- [x] VerifyCredential: Find→未找到 CREDENTIAL_NOT_FOUND ✓ (`verify_credential.go:20-25`)
- [x] VerifyCredential: 找到→verifier.Verify→false INVALID_CREDENTIAL ✓ (`verify_credential.go:43-48`)
- [x] VerifyCredential: 找到→verifier.Verify→true Success ✓ (`verify_credential.go:51-54`)
- [x] GetOrInit: Find→找到 IsNewUser=false ✓ (`get_or_init_subject.go:17-21`)
- [x] GetOrInit: Find→未找到→CreateSubject→BindCredential→IsNewUser=true ✓ (`get_or_init_subject.go:24-46`)

**结论**：0 偏差。

## 2. 行为与决策核对

**明确不做逐项核对**：
- [x] 无 IdentityCore struct — REV1 grep zero match ✓
- [x] 无 TOTP 实现代码 — REV3 grep 仅 `model.go:17` TypeTOTP 常量 ✓
- [x] 无 BindCredential / ListCredentials usecase 编排 — REV2 grep zero match ✓
- [x] 无真实 DB 连接 — REV4 grep zero match ✓
- [x] 不处理 subject 冻结状态 — 代码中无 AccountLocked 判断 ✓

**关键决策落地**：
- [x] D1 纯函数注入 — usecase 两个函数均接收 store + 其他依赖为参数，非 struct 方法 ✓
- [x] D2 CreateSubject 不改签名 — usecase 调用 `store.CreateSubject(ctx)`，不直接调 IDGenerator ✓
- [x] D3 Verifier map 注入 — `VerifyCredential` 接收 `map[IdentityType]crypto.CredentialVerifier` ✓

**跨层纪律核对**：
- [x] 凭证未找到返回 `Success=false` + ErrorCode（非 error） ✓
- [x] 密码不匹配返回 `Success=false` + ErrorCode（非 error） ✓
- [x] store 层错误（非 `ErrCredentialNotFound`）向上传播 ✓
- [x] Verifier 未注册时返回 `UNSUPPORTED_TYPE`（防止 nil map panic） ✓

**挂载点反向核对（可卸载性）**：
- [x] M1 `api.go` — 4 个 API 类型 ✓
- [x] M2 `usecase/verify_credential.go` — VerifyCredential 函数 ✓
- [x] M3 `usecase/get_or_init_subject.go` — GetOrInitializeSubjectID 函数 ✓
- [x] M4 `internal/store/mock.go` — MockStore ✓
- [x] **反向 grep**：`VerifyCredential` / `GetOrInitializeSubjectID` 外部引用 zero；`MockStore` 仅 usecase_test.go 引用；API 类型仅 api.go + usecase/ 引用。全在清单内 ✓
- [x] **拔除沙盘推演**：删 M1-M4 → `api.go` + `usecase/` + `internal/store/mock.go` 消失。无残留 ✓

## 3. 验收场景核对

- [x] **C1**：`VerifyCredential` 正确密码 → `Success=true` + SubjectID 非零
  - 证据：`usecase/usecase_test.go` `TestVerifyCredentialSuccess`
  - 结果：通过 ✓

- [x] **C2**：`GetOrInitializeSubjectID` 新标识 → `IsNewUser=true` + SubjectID 非零
  - 证据：`usecase/usecase_test.go` `TestGetOrInitSubjectNewUser`
  - 结果：通过 ✓

- [x] **C3**：同一标识再次调用 → `IsNewUser=false` + 相同 SubjectID
  - 证据：`usecase/usecase_test.go` `TestGetOrInitSubjectExistingUser`
  - 结果：通过 ✓

- [x] **C4**：创建 subject 并绑定密码后校验 → `Success=true` + SubjectID 匹配
  - 证据：`usecase/usecase_test.go` `TestEndToEndCreateAndVerify`
  - 结果：通过 ✓（implement 期间修正：Password 类型不适合 GetOrInit 静默注册，改用 CreateSubject + BindCredential 演示完整闭环）

- [x] **C5**：凭证未找到 → `Success=false` + `CREDENTIAL_NOT_FOUND`
  - 证据：`usecase/usecase_test.go` `TestVerifyCredentialNotFound`
  - 结果：通过 ✓

- [x] **C6**：密码错误 → `Success=false` + `INVALID_CREDENTIAL`
  - 证据：`usecase/usecase_test.go` `TestVerifyCredentialWrongPassword`
  - 结果：通过 ✓

- [x] **C7**：重复绑定 → `ErrDuplicateCredential`
  - 证据：`internal/store/mock_test.go` `TestMockStoreBindDuplicate`
  - 结果：通过 ✓

- [x] **C8**：SubjectID 不存在 → `ErrSubjectNotFound`
  - 证据：`internal/store/mock_test.go` `TestMockStoreBindMissingSubject`
  - 结果：通过 ✓

**反向核对**：
- [x] REV1-REV4 全部 zero match ✓

## 4. 术语一致性

对照方案第 0 节术语表 + 第 2.1 节命名：
- [x] `VerifyInput` — `api.go:4` + usecase 引用 ✓
- [x] `VerifyOutput` — `api.go:12` + usecase 引用 ✓
- [x] `GetOrInitSubjectInput` — `api.go:20` + usecase 引用 ✓
- [x] `GetOrInitSubjectOutput` — `api.go:27` + usecase 引用 ✓
- [x] `MockStore` — `internal/store/mock.go:12` ✓
- [x] `NewMockStore` — `internal/store/mock.go:19` ✓
- [x] 防冲突：无新增术语与现有类型冲突 ✓

## 5. 架构归并

对照方案第 4 节：

- [x] `ARCHITECTURE.md` §2 术语表：新增 VerifyInput/Output、GetOrInitSubjectInput/Output、MockStore 条目 ✓
- [x] `ARCHITECTURE.md` §3 根包：新增 `api.go` 条目 ✓
- [x] `ARCHITECTURE.md` §3 新增 `usecase/` 模块条目（verify_credential.go + get_or_init_subject.go） ✓
- [x] `ARCHITECTURE.md` §3 新增 `internal/store` 模块条目（mock.go） ✓
- [x] `ARCHITECTURE.md` §4 架构决定：新增"纯函数编排"（D1） ✓
- [x] `ARCHITECTURE.md` §5 约束：新增 usecase 不直接调 IDGenerator、Verifier map 注入 ✓
- [x] `ARCHITECTURE.md` §6 实现状态：password-verify 标 ✅ done ✓

## 6. requirement 回写

- 方案 frontmatter `requirement: prd` 已填
- PRD（`codestable/requirements/prd.md`）已覆盖密码校验和静默注册能力
- 本次实现完全遵循 PRD §4 API 契约，未改边界
- [x] req-prd 未变，无需更新

## 7. roadmap 回写

- [x] `identity-core-items.yaml` — `password-verify` 条目 `status: done` ✓
- [x] `identity-core-roadmap.md` — 第 5 节清单 `password-verify` 已标 ✓ done ✓
- [x] items.yaml 通过 `validate-yaml.py` 校验 ✓

## 8. AGENTS.md / CLAUDE.md 候选盘点

- 候选 1（与 f1 同）：go 版本自动升级问题。建议 AGENTS.md "已知坑" 补充
- 无新增候选

## 9. 遗留

- 实现阶段 C4 修正：Password 类型不适合 GetOrInitializeSubjectID 静默注册（该 API 设计用于免密登录场景）。端到端测试改为直接 CreateSubject + BindCredential 演示。不改变 API 契约，无需上层修改
- 无其他遗留
