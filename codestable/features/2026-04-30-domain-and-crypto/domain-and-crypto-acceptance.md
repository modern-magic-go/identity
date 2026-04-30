---
doc_type: feature-acceptance
feature: 2026-04-30-domain-and-crypto
status: completed
summary: domain-and-crypto 验收通过——接口契约 0 偏差、14 项验收场景全部通过、架构已归并、roadmap 已回写
tags: [identity, domain-model, crypto, foundation]
---

# domain-and-crypto 验收报告

> 阶段：阶段 3（验收闭环）
> 验收日期：2026-04-30
> 关联方案 doc：`codestable/features/2026-04-30-domain-and-crypto/domain-and-crypto-design.md`

## 1. 接口契约核对

对照方案第 2.1 节名词层逐一核查：

**名词层"现状 → 变化"逐项核对**：
- [x] `SubjectID = int64` → `model.go:4` ✓
- [x] `Realm = string` → `model.go:7` ✓
- [x] `IdentityType string` → `model.go:10` ✓
- [x] 6 个 IdentityType 常量 → `model.go:13-18` 全部到位 ✓
- [x] `Credential` struct（5 字段） → `model.go:22-28` ✓
- [x] `CredentialSummary` struct（2 字段） → `model.go:31-34` ✓
- [x] 5 个错误哨兵 → `errors.go:6-10` ✓
- [x] `IdentityStore` 接口（4 方法签名一致） → `store.go:6-17` ✓
- [x] `IDGenerator` 接口 → `internal/idgen/idgen.go:4-5` ✓
- [x] `CredentialVerifier` 接口 → `internal/crypto/verifier.go:6-8` ✓

**流程图核对**（第 2.2 节 mermaid 图）：
- [x] Snowflake 节点 → `internal/idgen/snowflake.go` `New` + `Generate` ✓
- [x] Bcrypt Hash/Verify 节点 → `internal/crypto/bcrypt.go` `Hash` + `Verify` ✓

**结论**：0 偏差。

## 2. 行为与决策核对

对照方案第 1 节 + 第 2.2 节：

**明确不做逐项核对**：
- [x] 无业务编排（VerifyCredential / GetOrInitializeSubjectID）→ REV1 grep zero match ✓
- [x] 无 TOTP 实现 → REV2 grep 仅 `model.go:17` TypeTOTP 常量声明，无实现代码 ✓
- [x] 无 MySQL/Redis 存储连接 → REV3 grep zero match（errors_test.go 假阳性已核实） ✓
- [x] 无并发处理代码 → bcrypt 纯函数，Snowflake 库自带线程安全 ✓

**关键决策落地**：
- [x] bcrypt DefaultCost = 10 → `internal/crypto/bcrypt.go:9` ✓
- [x] SubjectID 使用类型别名 `= int64` → `model.go:4` ✓
- [x] IdentityType 用 `string` 类型别名 + 常量枚举 → `model.go:10` ✓
- [x] Snowflake nodeID 范围 0-1023，New 构造函数校验 → `internal/idgen/snowflake.go:11` ✓

**跨层纪律核对**：
- [x] 哨兵错误用 `errors.New()`，调用方 `errors.Is` 匹配 → `errors_test.go` ✓
- [x] `Bcrypt.Verify` 不匹配时返回 `(false, nil)`（非 error） → `internal/crypto/bcrypt.go:22-24` ✓
- [x] 独立 `Verify` 函数返回原始 bcrypt error（供调用方判断 `ErrMismatchedHashAndPassword`） → `internal/crypto/bcrypt.go:40` ✓
- [x] `Hash` 透传 bcrypt 库错误 → `internal/crypto/bcrypt.go:31` ✓

**挂载点反向核对（可卸载性）**：
- [x] 挂载点 M1-M6 全部在代码有准确落点（§1 已逐条核对）
- [x] **反向 grep**：`identity.IdentityType` 引用仅限 `internal/crypto`（挂载点 5 内）；无外部未登记引用
- [x] **拔除沙盘推演**：删除 6 个挂载点后无残留代码——`internal/` 子包全部随挂载点消失，根包仅剩 go.mod

## 3. 验收场景核对

对照方案第 3 节关键场景清单：

- [x] **C1**：Snowflake 连续 1000 个 ID 全唯一且为正 int64
  - 证据：`internal/idgen/snowflake_test.go` `TestGenerateUniqueness`
  - 结果：通过 ✓

- [x] **C2**：BcryptHash 返回 `$2a$10$` 开头 60 字符 hash
  - 证据：`internal/crypto/bcrypt_test.go` `TestHashAndVerify` `strings.HasPrefix` 断言
  - 结果：通过 ✓

- [x] **C3**：Hash→Verify 正确密码返回 nil
  - 证据：`internal/crypto/bcrypt_test.go` `TestHashAndVerify`
  - 结果：通过 ✓

- [x] **C4**：错误密码 Verify 返回 `bcrypt.ErrMismatchedHashAndPassword`
  - 证据：`internal/crypto/bcrypt_test.go` `TestVerifyWrongPassword`
  - 结果：通过 ✓

- [x] **C5**：`go build ./...` 编译通过无循环引用
  - 证据：`go build ./...` 静默退出
  - 结果：通过 ✓

- [x] **C6**：`errors.Is(哨兵, 哨兵)` 为 true
  - 证据：`errors_test.go` `TestErrorSentinelsAreDistinct` 自反性
  - 结果：通过 ✓

- [x] **C7**：不同哨兵 `errors.Is` 互相为 false
  - 证据：同测试——互斥性
  - 结果：通过 ✓

- [x] **C8**：空密码 Hash 允许通过
  - 证据：`internal/crypto/bcrypt_test.go` `TestHashEmptyPassword`
  - 结果：通过 ✓

- [x] **C9**：cost<4 静默升至 MinCost（库行为）
  - 证据：`internal/crypto/bcrypt_test.go` `TestHashLowCost`
  - 结果：通过 ✓（implement 期间同步更新了 design C9）

- [x] **C10**：nodeID<0 返回 error
  - 证据：`internal/idgen/snowflake_test.go` `TestNewInvalidNodeID`
  - 结果：通过 ✓

- [x] **C11**：非法 hash 格式 Verify 返回 error
  - 证据：`internal/crypto/bcrypt_test.go` `TestVerifyBogusHash`
  - 结果：通过 ✓

**反向核对**：
- [x] REV1：grep `VerifyCredential|GetOrInit` → zero match ✓
- [x] REV2：grep `TOTP` → 仅 TypeTOTP 常量 ✓
- [x] REV3：无 MySQL/Redis 连接代码 ✓

## 4. 术语一致性

对照方案第 0 节术语表 grep 代码：
- [x] SubjectID — `model.go:4` ✓
- [x] Realm — `model.go:7,24` ✓
- [x] IdentityType — `model.go:10,26,32` + `store.go:8` ✓
- [x] Credential — `model.go:22` + `store.go:8,14` ✓
- [x] CredentialSummary — `model.go:31` + `store.go:17` ✓
- [x] IdentityStore — `store.go:6` ✓
- [x] IDGenerator — `internal/idgen/idgen.go:4` ✓
- [x] CredentialVerifier — `internal/crypto/verifier.go:6` ✓
- [x] 防冲突检查——仓库内无同名概念 ✓

## 5. 架构归并

对照方案第 4 节：

- [x] `ARCHITECTURE.md`：从空骨架填充为完整架构入口——含项目简介、术语表、模块索引、架构决定、约束、实现状态。已写入 ✓
- [x] 根包 `identity` 三文件已在模块索引有条目
- [x] `internal/idgen` 已在模块索引有条目
- [x] `internal/crypto` 已在模块索引有条目
- [x] roadmap 规划已在 §3 列出

## 6. requirement 回写

- 方案 frontmatter 原 `requirement` 字段为空
- 本次新增了用户可感能力（身份核库基础层 + 密码学适配）
- 对应 requirement 已存在：`codestable/requirements/prd.md`（完整定义了整体 Identity Core 系统）
- [x] 已回填方案 frontmatter `requirement: prd`
- 无需触发 `cs-req`（PRD 已覆盖当前能力范围）

## 7. roadmap 回写

对照方案 frontmatter `roadmap: identity-core` + `roadmap_item: domain-and-crypto`：
- [x] `identity-core-items.yaml` — `domain-and-crypto` 条目 `status: done` 已更新 ✓
- [x] `identity-core-roadmap.md` — 第 5 节子 feature 清单 `domain-and-crypto` 已标 ✓ done
- [x] items.yaml 通过 `validate-yaml.py` 校验 ✓

## 8. AGENTS.md / CLAUDE.md 候选盘点

- [x] 候选 1：**go 版本自动升级** — implement 期间 `go get golang.org/x/crypto` 自动将 go 版本从 1.21 升至 1.25.0。后续 feature 加依赖时可能再次触发版本升级。建议 AGENTS.md "已知坑" 补充。
- 无其他候选。

## 9. 遗留

- 后续优化点：无
- 已知限制：无
- 实现阶段"顺手发现"列表：无（implement 期间仅发现 C9 库行为偏差，已当场修 design + 代码）
