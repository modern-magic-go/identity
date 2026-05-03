package identity

// VerifyInput 凭证校验入参
type VerifyInput struct {
	Realm        string       // 领域
	IdentityType IdentityType // 凭证类型
	Identifier   string       // 标识符（手机号 / 用户名）
	InputData    string       // 用户输入的验证物（明文密码）
}

// VerifyOutput 校验结果
type VerifyOutput struct {
	Success   bool      // 校验是否通过
	SubjectID SubjectID // 仅在 Success=true 时有效
	ErrorCode string    // ACCOUNT_LOCKED / INVALID_CREDENTIAL / CREDENTIAL_NOT_FOUND
	ErrorMsg  string    // 人类可读描述
}

// GetOrInitSubjectInput 静默解析入参
type GetOrInitSubjectInput struct {
	Realm        string       // 领域
	IdentityType IdentityType // 凭证类型
	Identifier   string       // 标识符
}

// GetOrInitSubjectOutput 解析结果
type GetOrInitSubjectOutput struct {
	SubjectID SubjectID // 已有或新创建的 subject_id
	IsNewUser bool      // 是否为新注册
}

// BindCredentialInput 绑定凭证入参
type BindCredentialInput struct {
	SubjectID      SubjectID    // 目标 subject
	Realm          string       // 领域
	IdentityType   IdentityType // 凭证类型
	Identifier     string       // 标识符
	CredentialData string       // 需加密存储的凭证数据，第三方登录可为空
}

// ListCredentialsInput 列出凭证入参
type ListCredentialsInput struct {
	SubjectID SubjectID // 目标 subject
	Realm     string    // 领域
}


