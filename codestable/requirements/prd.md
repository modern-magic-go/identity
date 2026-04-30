# 通用身份核 (Identity Core) 基础架构设计文档 V1.0

## 1. 架构定位与设计原则

本模块定义为系统的 **“通用身份核（Identity Core）”**，是一个纯粹的、Headless（无头）的底层 RPC/API 服务。

### 1.1 核心边界（DOs & DON'Ts）

- 【做】身份映射：解决“外部标识（如手机号、微信、账号）对应系统内哪个实体”的问题。
- 【做】凭证校验：验证提供的“证据（密码、TOTP、第三方授权）”是否合法。
- 【不做】Token/Session 管理：模块验证成功后仅返回 subject_id 和验证状态，Token 签发及 Redis 会话维持由上层业务网关或 Auth Service 负责。
- 【不做】真实数据库写入：本模块只负责数据建模和接口契约，不直接落库。
- 【做】仓储分离：真实存储实现由项目内的仓储层负责，和数据库访问解耦。
- 【不做】用户画像（Profile）：模块内坚决不存储昵称、头像、性别等业务属性。
- 【不做】流程拦截：本模块不决定“某个端必须用什么登录”，仅提供原子化的校验能力，流程编排交由调用方实现。

### 1.2 核心隔离机制：Realm（领域）

放弃传统的 TenantID 或 AppID 命名，采用 Realm（领域/界）作为账号池的物理隔离单位。

- 定义：一个 Realm 代表一个独立的账号命名空间（如 `c_users`、`b_admins`）。
- 特性：不同的 Realm 之间数据完全物理隔离。同一个手机号在 `c_users` 和 `b_admins` 中注册，会生成两个截然不同的全局 subject_id。

## 2. 核心领域模型（原子凭证模型）

我们采用 **“凭证平权化（Atomic Credentials）”** 设计。无论是密码、微信 OpenID 还是 TOTP 密钥，在底层看来都是附着在 subject_id 上的一种“凭证（Credential）”。

- subject_id（全局唯一标识）：自然人在系统内的唯一代号。
- Realm（领域）：凭证生效的边界。
- Identity Type（标识类型）：凭证的类别（如：PASSWORD、WECHAT_OPENID、WECHAT_UNIONID、TOTP_SECRET）。
- Identifier（标识符）：外部可读的账号名（如：具体的手机号、普通的 Username、或者微信返回的字串）。
- Credential Data（凭证数据）：用于验证的加密物（如：Bcrypt Hash 后的密码、TOTP 的加密 Secret）。

## 3. 数据模型设计（MySQL / PostgreSQL）

本章节仅描述数据模型与字段契约，不表示本模块直接负责真实数据库写入。

系统仅需两张核心模型即可支撑所有复杂的身份验证流。

### 3.1 全局用户主表 `user_subject`

用于证明实体的存在，并控制全局维度的账号状态。真实持久化由项目内仓储层实现。

```sql
CREATE TABLE `user_subject` (
  `id` bigint(20) NOT NULL COMMENT '全局唯一标识 (由雪花算法生成)',
  `status` tinyint(4) NOT NULL DEFAULT '1' COMMENT '全局状态: 1-正常, 2-冻结 (冻结后所有Realm下的凭证全部失效)',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户主体';
```

### 3.2 统一凭证表 `identity_credential`

本系统的绝对核心。记录所有的登录方式和二次验证因子。真实持久化由项目内仓储层实现。

```sql
CREATE TABLE `identity_credential` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `subject_id` bigint(20) NOT NULL COMMENT '关联 user_subject.id',

  -- 领域隔离
  `realm` varchar(32) NOT NULL DEFAULT 'default' COMMENT '命名空间/领域 (例: users, admins, global)',

  -- 凭证标识定义
  `identity_type` varchar(32) NOT NULL COMMENT '类型: PASSWORD, SMS, WECHAT_OPENID, WECHAT_UNIONID, EMAIL, TOTP',
  `identifier` varchar(128) NOT NULL COMMENT '外部标识: 138xxx, oX99xxx, admin_name',

  -- 校验数据
  `credential_data` varchar(255) DEFAULT NULL COMMENT '敏感凭证数据 (Hash密码、TOTP密钥等)。第三方登录可为空',

  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  PRIMARY KEY (`id`),
  -- 核心防重约束：同一个 Realm 下，同一种类型的标识符绝对不能重复
  UNIQUE KEY `uk_realm_type_identifier` (`realm`, `identity_type`, `identifier`),
  -- 辅助索引：快速查询某个 subject_id 名下挂载了哪些凭证 (用于业务侧判断是否需要触发 2FA)
  KEY `idx_subject_id_realm` (`subject_id`, `realm`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='原子化用户凭证表';
```

## 4. 核心 API 接口契约定义

接口设计遵循“无状态”和“原子化”原则，所有接口均不涉及会话上下文。

### 4.1 `VerifyCredential`（通用凭证比对）

职能：验证调用方提供的标识和凭证是否匹配。

请求参数：

- `Realm` (string) - 领域
- `IdentityType` (string) - 凭证类型
- `Identifier` (string) - 标识符
- `InputData` (string) - 用户输入的验证物（明文密码、TOTP Code。如果是三方授权扫码，此处为空）

返回结果：

- `Success` (bool)
- `subject_id` (int64) - 仅在 `Success` 为 `true` 时返回
- `ErrorCode` / `ErrorMsg` - 如 `ACCOUNT_LOCKED`、`INVALID_CREDENTIAL`

### 4.2 `GetOrInitializeSubjectID`（静默解析与发号）

职能：专供“免密登录”（如微信扫码、上层已校验完的手机验证码）使用。有则返回对应 subject_id，无则自动生成 subject_id 并静默注册该凭证。

请求参数：

- `Realm` (string)
- `IdentityType` (string)
- `Identifier` (string)

返回结果：

- `subject_id` (int64)
- `IsNewUser` (bool) - 告知上层是否是新注册

### 4.3 `BindCredential`（挂载新凭证）

职能：为已存在的 subject_id 增加新的认证方式（如：登录后强制绑定手机，或开启 TOTP）。

请求参数：

- `subject_id` (int64)
- `Realm` (string)
- `IdentityType` (string)
- `Identifier` (string)
- `CredentialData` (string) - 需底层加密存储的数据

返回结果：`Success` (bool) 或唯一性冲突报错。

> **⚠️ 已知缺口：缺少逆操作**  
> 当前 `IdentityStore` 接口未定义 `RemoveCredential` 方法，无法解除已有凭证的绑定。  
> 这导致**换绑场景**（如用户更换手机号）无法直接支持——必须先删除旧凭证再绑定新凭证。  
> 当前需调用方自行在仓储层实现删除逻辑，或等待后续迭代在 `IdentityStore` 中增加该接口。

### 4.4 `ListCredentials`（获取可用验证因子）

职能：查询某个 subject_id 在当前 Realm 下拥有哪些凭证（抹除敏感数据），供上层业务决定是否需要发起 2FA（如发现有 `type=TOTP`，则要求用户进行二次验证）。

请求参数：`subject_id`、`Realm`

返回结果：`[]CredentialSummary {Type, Identifier}`

### 4.5 `LookupSubject` — 纯查询 SubjectID（🚧 待支持）

职能：仅根据凭证三元组查询 SubjectID，**不自动创建**。区别于 `GetOrInitializeSubjectID` 的"不存在就注册"语义。

请求参数：

- `Realm` (string)
- `IdentityType` (string)
- `Identifier` (string)

返回结果：

- `subject_id` (int64) — 找到时返回
- 未找到时返回 `ErrSubjectNotFound`

> **⚠️ 已知缺口：缺少纯查询接口**  
> 当前所有凭证 → SubjectID 的查询路径只有 `GetOrInitializeSubjectID`（自动创建）和 `VerifyCredential`（需 InputData，语义是验证）。  
> 上层无法做到"先查这个微信有没有绑定过用户，有则登录，无则引导用户选择注册或绑定已有账号"。  
> 缺少 `LookupSubject` 导致无法优雅处理"一人多凭证 vs 一人多 Subject"的账号关联场景。

## 5. 典型业务流程编排指南（致上层调用方）

本底座只提供武器，怎么打仗由上层（Auth Service）编排。

### 5.1 场景：账号密码 + TOTP 双因素登录

1. 上层收到用户请求，调用底座 `VerifyCredential(realm="admins", type="PASSWORD", identifier="admin", input="123456")`。
2. 底座返回验证成功，并返回 `subject_id: 888`。
3. 上层调用底座 `ListCredentials(subject_id=888, realm="admins")`。
4. 上层发现该 subject_id 列表中存在 `type="TOTP"`。
5. 上层挂起当前登录流程，向前端下发挑战："请输入验证器代码"。
6. 前端提交代码，上层再次调用底座 `VerifyCredential(realm="admins", type="TOTP", identifier="totp_device_1", input="837261")`。
7. 验证通过，上层签发 Token。

### 5.2 场景：微信登录（兼容 OpenID 与 UnionID）

微信体系复杂，UnionID 具有跨 Realm 的潜质，建议做如下编排：

1. 客户端拿到微信 Code，上层服务去向微信换取 OpenID 和 UnionID。
2. 上层优先按 UnionID 查找：调用底座 `VerifyCredential(realm="global", type="WECHAT_UNIONID", identifier="u_xxx")`。
3. 若找到 subject_id：登录成功。上层可异步调用 `BindCredential` 顺手把当前小程序的 OpenID 绑定到该 subject_id 下的指定 Realm。
4. 若没找到 UnionID：降级使用 OpenID 调用 `GetOrInitializeSubjectID(realm="c_users", type="WECHAT_OPENID", identifier="o_xxx")` 获取/生成 subject_id，随后异步将 UnionID 绑定至该 subject_id。

### 5.3 场景：换绑手机号（🚧 待支持）

当前底座缺少 `RemoveCredential` 能力，换绑手机号需要上层自行在仓储层实现。

预期完整流程：

1. 上层调用底座 `VerifyCredential(realm, TypeSMS, oldPhone, code)` 验证旧手机。
2. 验证通过后，上层在仓储层删除旧凭证（`DELETE FROM identity_credential WHERE realm=? AND identity_type='SMS' AND identifier=?`）。
3. 上层调用底座 `BindCredential(subjectID, realm, TypeSMS, newPhone, "")` 绑定新手机号。

> 当前 `BindCredential` 不会自动解除旧凭证，直接绑定新手机号会导致该 subject 同时持有两个手机号，不符合业务预期。

### 5.4 场景：微信静默登录后强制绑手机号 — Subject 分裂风险（🚧 待支持）

**问题：** 新用户通过微信静默登录（`GetOrInitializeSubjectID`）创建了 Subject **A**。随后业务强制绑定手机号，但该手机号已在 Subject **B** 下注册过 → `BindCredential` 返回 `ErrDuplicateCredential`。一个真实用户对应两个 Subject，模块内无法合并。

**根因：** 缺少 `LookupSubject` 纯查询接口，上层无法在静默注册前先判断该凭证是否已关联到其他 Subject。

**预期正确流程（依赖 `LookupSubject`）：**

1. 收到微信静默请求，先调 `LookupSubject(realm, TypeWECHAT_OPENID, openID)`。
2. 若返回 SubjectID → 直接登录，**不执行手机号强制绑定**。
3. 若返回 `ErrSubjectNotFound` → 调 `GetOrInitializeSubjectID` 创建新 Subject。
4. 随后强制绑手机时，先调 `LookupSubject(realm, TypeSMS, phone)`。
5. 若手机号已被其他 Subject 绑定 → 返回冲突，由上层决定是否引导用户走"账号关联"流程。
6. 若无冲突 → 执行 `BindCredential` 绑定手机号。

> 缺少 `LookupSubject` 的后果：上层在面对"凭证已存在但自己不知道"的场景时，只能盲目地调 `GetOrInitializeSubjectID`，导致 Subject 分裂。

## 6. 技术栈与安全规范建议（Go 语言生态）

为了保证底座的极简与高效，建议开发团队遵循以下规范：

- subject_id 生成：强制使用 Snowflake 算法（推荐库：`github.com/bwmarrin/snowflake`）。切勿使用数据库自增 ID，防止业务数据量被暴露。
- 密码存储：强制使用 bcrypt（推荐库：`golang.org/x/crypto/bcrypt`）。此算法自带随机 Salt（盐），并可通过调整 cost 参数控制计算耗时，有效防止彩虹表与暴力破解。
- TOTP 实现：推荐使用 `github.com/pquerna/otp/totp` 库处理 2FA 密钥生成与验证。
