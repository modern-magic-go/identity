package usecase

import (
	"context"
	"fmt"

	"github.com/modern-magic-go/identity"
)

// LogoutCurrent 注销当前会话。
func (s *Service) LogoutCurrent(ctx context.Context, input LogoutCurrentInput) (*LogoutResult, error) {
	if input.AccessToken == "" {
		return nil, fmt.Errorf("access token 不能为空")
	}
	info, err := s.sessionManager.InspectAccessToken(ctx, input.AccessToken)
	if err != nil {
		return nil, err
	}
	if err := s.sessionManager.Revoke(ctx, info.AccessTokenID, info.RefreshTokenID); err != nil {
		return nil, err
	}
	if err := s.sessionRefs.RevokeByTokenID(ctx, info.AccessTokenID, s.now()); err != nil {
		return nil, err
	}
	return &LogoutResult{Success: true, RevokedAccessToken: true, RevokedRefreshToken: info.RefreshTokenID != ""}, nil
}

// LogoutAll 注销主体全部会话。
func (s *Service) LogoutAll(ctx context.Context, input LogoutAllInput) (*LogoutAllResult, error) {
	if input.SubjectID <= 0 {
		return nil, fmt.Errorf("subject id 无效")
	}
	revoked, err := s.sessionManager.RevokeAllBySubject(ctx, input.SubjectID)
	if err != nil {
		return nil, err
	}
	if _, err := s.sessionRefs.RevokeBySubjectID(ctx, input.SubjectID, s.now()); err != nil {
		return nil, err
	}
	return &LogoutAllResult{Success: true, RevokedSessions: int(revoked)}, nil
}

var _ identity.SessionRefStatus
