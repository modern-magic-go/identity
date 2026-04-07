package usecase

// SubjectView 是用例返回的主体摘要。
type SubjectView struct {
	SubjectID   int64
	SubjectNo   string
	SubjectType string
	Realm       string
}

// LoginResult 是登录成功返回。
type LoginResult struct {
	Subject SubjectView
}

// PasswordLoginInput 定义密码登录输入。
type PasswordLoginInput struct {
	Realm      string
	Identifier string
	Password   string
	Platform   string
	DeviceID   string
	DeviceInfo string
	IP         string
	UserAgent  string
	TraceID    string
}
