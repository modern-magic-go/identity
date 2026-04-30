# credential-crud 验收报告

> 阶段：阶段 3（验收闭环）
> 验收日期：2026-04-30
> 关联方案 doc：codestable/features/2026-04-30-credential-crud/credential-crud-design.md

## 1. 接口契约核对

对照方案第 2.1 节名词层逐一核查：

**接口示例逐项核对**：
- [x] BindCredential 正常路径（`usecase/bind_credential.go:10` BindCredential）：示例输入→输出 → 代码实际行为：一致。BindCredentialInput → 构造 *Credential → store.BindCredential → nil
- [x] BindCredential 重复（`usecase/bind_credential.go:10`）：示例 → 代码实际返回 `ErrDuplicateCredential`，`errors.Is` 匹配 → 一致
- [x] BindCredential subject 不存在（`usecase/bind_credential.go:10`）：示例 → 代码实际返回 `ErrSubjectNotFound`，`errors.Is` 匹配 → 一致
- [x] ListCredentials 有凭证（`usecase/list_credentials.go:10` ListCredentials）：示例 → 代码实际返回 `[]CredentialSummary`，含 `Type` + `Identifier`，不含 `CredentialData` → 一致
- [x] ListCredentials 空结果（`usecase/list_credentials.go:10`）：示例 → 代码实际返回空切片 `[]`，nil error → 一致

**名词层"现状 → 变化"逐项核对**：
- [x] `BindCredentialInput` struct（`api.go:33`）：声称的变化 → 代码改动：一致，5 字段匹配
- [x] `ListCredentialsInput` struct（`api.go:42`）：声称的变化 → 代码改动：一致，2 字段匹配

**流程图核对**（第 2.2 节开头 mermaid 图）：
- [x] 图中 BindCredential 节点：BindCredentialInput → 构造 Credential → store.BindCredential → 返回 nil/ErrDup/ErrNotFound — `usecase/bind_credential.go:15-22` 实际落点一致
- [x] 图中 ListCredentials 节点：ListCredentialsInput → store.ListBySubjectRealm → 返回结果 — `usecase/list_credentials.go:14` 实际落点一致
- [x] 图中 GetOrInitializeSubjectID → BindCredential 虚线 — `usecase/get_or_init_subject.go:33` 实际落点一致

## 2. 行为与决策核对

对照方案第 1 节 + 第 2.2 节：

**需求摘要逐项验证**：
- [x] BindCredential 为已存在 subject 绑定新凭证：`usecase/bind_credential.go` 实现，`TestBindCredentialSuccess` PASS
- [x] ListCredentials 列出 subject 在 Realm 下所有凭证（脱敏）：`usecase/list_credentials.go` 实现，`TestListCredentialsHasCredentials` PASS（验证不含 CredentialData）

**明确不做逐项核对**（第 3 节反向核对项）：
- [x] 无 IdentityCore struct：`grep "IdentityCore" *.go` → zero match
- [x] 无新增 store 方法/接口变动：`git diff --stat` 显示 `store.go` 无变化
- [x] 无 credential 更新/删除：`grep "UpdateCredential|DeleteCredential|RemoveCredential" *.go` → zero match
- [x] 无真实 DB 连接：无 MySQL/Redis 连接代码

**关键决策落地**：
- [x] D1 纯函数形态：`BindCredential(ctx, store, BindCredentialInput) error` — 代码签名一致
- [x] D2 旁路纠正：`get_or_init_subject.go:33` 已从 `store.BindCredential` 切为 `BindCredential`
- [x] D3 ListCredentials 空结果返回空切片：`usecase/list_credentials.go:14` 直透 store 返回值，MockStore 对不存在 subject 返回空切片
- [x] D4 错误直透：`usecase/bind_credential.go:22` 直接 `return store.BindCredential(ctx, cred)`，不做错误翻译

**编排层"现状 → 变化"逐项核对**：
- [x] 新增 BindCredential 编排：`usecase/bind_credential.go` 新增文件，构造 Credential → 委托 store
- [x] 新增 ListCredentials 编排：`usecase/list_credentials.go` 新增文件，委托 store.ListBySubjectRealm
- [x] 重构 GetOrInitializeSubjectID：`usecase/get_or_init_subject.go:33-38` 已改为调用 `BindCredential`

**跨层纪律核对**：
- [x] 错误语义 store 哨兵直透：代码中 `bind_credential.go:22` 直返 store 错误，无 wrap
- [x] BindCredential 非幂等：MockStore 内部 `ErrDuplicateCredential` 保证，测试 `TestBindCredentialDuplicate` PASS
- [x] 纯函数无共享状态：两个函数均为无副作用纯编排
- [x] 可观测点无：与 f1-f3 一致，无明显日志/埋点

**挂载点反向核对（可卸载性）**：
- [x] M1 `usecase/bind_credential.go` BindCredential 函数 → 代码落点一致
- [x] M2 `usecase/list_credentials.go` ListCredentials 函数 → 代码落点一致
- [x] M3 `api.go` BindCredentialInput / ListCredentialsInput 类型 → 代码落点一致
- [x] M4 `usecase/get_or_init_subject.go` 内部调用切换 → 代码落点一致
- [x] **反向核查**（grep `BindCredential|ListCredentials`）：所有代码引用均已纳入清单——`usecase/bind_credential.go`、`usecase/list_credentials.go`、`usecase/get_or_init_subject.go`、`api.go`。`store.go` / `internal/store/mock.go` 的 `BindCredential` 引用为 store 层接口/实现，属 f1-f2 产物，非本 feature 引入。
- [x] **拔除沙盘推演**：删除上述 4 个挂载点（3 处新增 + 1 处修改回退）→ feature 完全消失，无残留

## 3. 验收场景核对

对照方案第 3 节关键场景清单，逐条可观察证据验证：

- [x] **C1**：BindCredential 成功绑定 → 返回 nil；ListBySubjectRealm 可查到
  - 证据来源：单测 `TestBindCredentialSuccess` @ `usecase/usecase_test.go:459`
  - 结果：通过

- [x] **C2**：同 subject 绑定多类型凭证（PASSWORD + TOTP）→ 两次均返回 nil；ListCredentials 返回 2 条
  - 证据来源：单测 `TestListCredentialsHasCredentials` @ `usecase/usecase_test.go:545`
  - 结果：通过

- [x] **C3**：ListCredentials 返回脱敏列表 → 含 Type + Identifier，不含 CredentialData
  - 证据来源：单测 `TestListCredentialsHasCredentials` + CredentialSummary 类型定义（`model.go:31` 不含 CredentialData 字段）
  - 结果：通过

- [x] **C4**：BindCredential 重复绑定 → 返回 ErrDuplicateCredential
  - 证据来源：单测 `TestBindCredentialDuplicate` @ `usecase/usecase_test.go:492`
  - 结果：通过

- [x] **C5**：BindCredential SubjectID 不存在 → 返回 ErrSubjectNotFound
  - 证据来源：单测 `TestBindCredentialSubjectNotFound` @ `usecase/usecase_test.go:525`
  - 结果：通过

- [x] **C6**：ListCredentials target subject 无凭证 → 返回空切片，nil error
  - 证据来源：单测 `TestListCredentialsEmpty` @ `usecase/usecase_test.go:580`
  - 结果：通过

- [x] **C7**：ListCredentials target subject 不存在 → 返回空切片，nil error
  - 证据来源：单测 `TestListCredentialsSubjectNotFound` @ `usecase/usecase_test.go:602`
  - 结果：通过

- [x] **C8**：GetOrInit 新用户后 ListCredentials 可见 → 1 条凭证摘要
  - 证据来源：已有单测 `TestGetOrInitSubjectNewUser` @ `usecase/usecase_test.go:111` PASS（调用链经 BindCredential）
  - 结果：通过

- [x] **C9**：已有 GetOrInit 测试全部通过 → TestGetOrInitSubjectNewUser / TestGetOrInitSubjectExistingUser PASS
  - 证据来源：`go test ./usecase/... -run "TestGetOrInit"` 全部 PASS
  - 结果：通过

**无前端改动**，跳过浏览器验证。

## 4. 术语一致性

对照方案第 0 节 + 第 2.1 节命名 grep 代码：

- `BindCredentialInput`：`api.go` 定义 + `bind_credential.go` 引用 + `get_or_init_subject.go` 引用 → 全部一致
- `ListCredentialsInput`：`api.go` 定义 + `list_credentials.go` 引用 → 全部一致
- `BindCredential`：`bind_credential.go` 函数名 + `get_or_init_subject.go` 调用点 → 与 store 接口方法 `BindCredential` 同名但包路径不同（`usecase.BindCredential` vs `(MockStore).BindCredential`），方案第 0 节声明"无新术语"，无冲突
- `ListCredentials`：`list_credentials.go` 函数名 → 与 store 接口方法 `ListBySubjectRealm` 命名不同，不存在同名歧义
- 防冲突：`grep "IdentityCore\|UpdateCredential\|DeleteCredential"` → zero match

## 5. 架构归并

对照方案第 4 节：

- [x] **名词归并**：`ARCHITECTURE.md` 根包条目追加 `BindCredentialInput` / `ListCredentialsInput` 两个 API 类型 — 已写入
- [x] **动词骨架归并**：`ARCHITECTURE.md` usecase 模块条目追加 `bind_credential.go` + `list_credentials.go` — 已写入
- [x] **跨层纪律归并**：本 feature 无新增跨层纪律（错误语义、幂等性等均沿用 f1-f3 约定）
- [x] **实现状态**：`ARCHITECTURE.md` 状态表 `credential-crud` 行：`⬜ planned` → `✅ done` — 已更新
- [x] **架构总入口状态行**：更新为 `credential-crud 完成` — 已更新

## 6. requirement 回写

对照方案 frontmatter `requirement: prd`：

- [x] 有对应 req（PRD）且本次实现完全遵循 PRD §4.3/§4.4 已定义的 BindCredential / ListCredentials 契约，未改边界/用户故事 → **req-prd 未变，无需更新**

## 7. roadmap 回写

对照方案 frontmatter `roadmap: identity-core` / `roadmap_item: credential-crud`：

- [x] `identity-core-items.yaml`：`credential-crud` `status: done` + `feature: 2026-04-30-credential-crud` — 已更新，`validate-yaml.py` 校验通过
- [x] `identity-core-roadmap.md`：第 5 节子 feature 清单 `credential-crud` 行标记 `✓ done` — 已同步

## 8. AGENTS.md / CLAUDE.md 候选盘点

- [x] 无候选：本 feature 未暴露需要补入 AGENTS.md / CLAUDE.md 的内容。这是纯 Go 库项目，无环境/工具/工作流类的新踩坑。

## 9. 遗留

- 后续优化点：无
- 已知限制：MockStore.ListBySubjectRealm 遍历 map 导致返回顺序不确定——store 层问题，非 usecase 层问题，不阻塞本 feature
- 实现阶段"顺手发现"列表：无
