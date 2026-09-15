package auth

import "time"

const (
	CurrentTermsVersion             = "2026-09-10"
	CurrentPrivacyVersion           = "2026-09-10"
	CurrentAIImageProcessingVersion = "2026-09-10"
	SessionCookieName               = "formtally_session"
)

type ClientType string

const (
	ClientWeb               ClientType = "web"
	ClientWeChatMiniProgram ClientType = "wechat_miniprogram"
)

type Purpose string

const (
	PurposeLogin         Purpose = "login"
	PurposeDeleteAccount Purpose = "delete_account"
)

type RequestCodeInput struct {
	Phone   string  `json:"phone"`
	Purpose Purpose `json:"purpose"`
}

type Verification struct {
	RequestID         string `json:"requestId"`
	ExpiresInSeconds  int    `json:"expiresInSeconds"`
	RetryAfterSeconds int    `json:"retryAfterSeconds"`
}

type CreateSessionInput struct {
	Phone                 string
	Code                  string
	VerificationRequestID string
	TermsVersion          string
	PrivacyVersion        string
}

type WeChatSessionInput struct {
	LoginCode      string
	TermsVersion   string
	PrivacyVersion string
}

type WeChatSessionResult struct {
	Session SessionResult
	Token   string
}

type WeChatPhoneBindingInput struct {
	BindingTicket  string
	PhoneCode      string
	TermsVersion   string
	PrivacyVersion string
}

type SessionView struct {
	ExpiresAt time.Time `json:"expiresAt"`
}

type UserView struct {
	ID               string  `json:"id"`
	PhoneMasked      *string `json:"phoneMasked"`
	OnboardingStatus string  `json:"onboardingStatus"`
}

type ConsentsView struct {
	TermsVersion                    string  `json:"termsVersion"`
	PrivacyVersion                  string  `json:"privacyVersion"`
	AIImageProcessingVersion        *string `json:"aiImageProcessingVersion"`
	CurrentAIImageProcessingVersion string  `json:"currentAiImageProcessingVersion"`
}

type SessionResult struct {
	Session  SessionView  `json:"session"`
	User     UserView     `json:"user"`
	Consents ConsentsView `json:"consents"`
}

type Error struct {
	Code       string
	Message    string
	Status     int
	RetryAfter int
}

func (e *Error) Error() string { return e.Code }

func ErrorCode(err error) string {
	if apiError, ok := err.(*Error); ok {
		return apiError.Code
	}
	return ""
}
