package usecase

import (
	"strings"
	"time"

	"github.com/modern-magic-go/identity"
)

func (s *Service) now() time.Time {
	if s.clock == nil {
		return time.Now().UTC()
	}
	return s.clock.Now().UTC()
}

func (s *Service) buildSubjectView(subject *identity.Subject) SubjectView {
	if subject == nil {
		return SubjectView{}
	}
	return SubjectView{
		SubjectID:   subject.ID,
		SubjectNo:   subject.SubjectNo,
		SubjectType: subject.SubjectType,
		Realm:       subject.Realm,
	}
}

func maxDurationSeconds(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	return int64(duration.Seconds())
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
