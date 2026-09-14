package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/sms"
	"github.com/GnEveLynn/FormTally/server/internal/wechat"
)

var mainlandPhone = regexp.MustCompile(`^\+861[3-9][0-9]{9}$`)

type Service struct {
	store  *PostgresStore
	sender sms.Sender
	weChat wechat.Client
	now    func() time.Time
}

func NewService(store *PostgresStore, sender sms.Sender, weChatClients ...wechat.Client) *Service {
	service := &Service{store: store, sender: sender, now: time.Now}
	if len(weChatClients) > 0 {
		service.weChat = weChatClients[0]
	}
	return service
}

func validMainlandPhone(phone string) bool { return mainlandPhone.MatchString(phone) }

func (s *Service) RequestCode(ctx context.Context, input RequestCodeInput) (Verification, error) {
	return s.RequestCodeForUser(ctx, "", input)
}

func (s *Service) RequestCodeForUser(ctx context.Context, userID string, input RequestCodeInput) (Verification, error) {
	if !validMainlandPhone(input.Phone) {
		return Verification{}, &Error{Code: "PHONE_UNSUPPORTED", Message: "请输入有效的中国大陆手机号", Status: http.StatusUnprocessableEntity}
	}
	if input.Purpose != PurposeLogin && input.Purpose != PurposeDeleteAccount {
		return Verification{}, &Error{Code: "VALIDATION_FAILED", Message: "验证码用途无效", Status: http.StatusUnprocessableEntity}
	}
	if input.Purpose == PurposeDeleteAccount {
		if userID == "" {
			return Verification{}, unauthenticated()
		}
		owned, err := s.store.UserOwnsPhone(ctx, userID, input.Phone)
		if err != nil {
			return Verification{}, err
		}
		if !owned {
			return Verification{}, &Error{Code: "PHONE_MISMATCH", Message: "手机号与当前账户不一致", Status: http.StatusUnprocessableEntity}
		}
	}
	now := s.now().UTC()
	if retryAt, ok, err := s.store.RetryAt(ctx, input.Phone, input.Purpose); err != nil {
		return Verification{}, err
	} else if ok && retryAt.After(now) {
		seconds := int(retryAt.Sub(now).Seconds())
		if seconds < 1 {
			seconds = 1
		}
		return Verification{}, &Error{Code: "RATE_LIMITED", Message: "请求过于频繁", Status: http.StatusTooManyRequests, RetryAfter: seconds}
	}
	code, err := randomCode()
	if err != nil {
		return Verification{}, err
	}
	id := "verify_" + rand.Text()
	if err := s.store.InsertCode(ctx, id, input.Phone, input.Purpose, codeHash(id, code), now.Add(5*time.Minute), now.Add(time.Minute)); err != nil {
		return Verification{}, err
	}
	if err := s.sender.Send(ctx, input.Phone, string(input.Purpose), code); err != nil {
		return Verification{}, err
	}
	return Verification{RequestID: id, ExpiresInSeconds: 300, RetryAfterSeconds: 60}, nil
}

func (s *Service) CreateSession(ctx context.Context, input CreateSessionInput) (SessionResult, string, error) {
	if input.TermsVersion != CurrentTermsVersion || input.PrivacyVersion != CurrentPrivacyVersion {
		return SessionResult{}, "", &Error{Code: "AGREEMENT_VERSION_OUTDATED", Message: "协议版本已更新，请重新确认", Status: http.StatusConflict}
	}
	if !validMainlandPhone(input.Phone) || len(input.Code) != 6 || input.VerificationRequestID == "" {
		return SessionResult{}, "", &Error{Code: "VERIFICATION_CODE_INVALID", Message: "验证码错误", Status: http.StatusUnprocessableEntity}
	}
	token, hash, now, expires := s.newSession()
	result, err := s.store.ConsumeCodeAndCreateSession(ctx, input, codeHash(input.VerificationRequestID, input.Code), hash, now, expires)
	return result, token, err
}

func (s *Service) GetSession(ctx context.Context, token string) (SessionResult, error) {
	if token == "" {
		return SessionResult{}, unauthenticated()
	}
	return s.store.Session(ctx, tokenHash(token), s.now().UTC())
}

func (s *Service) RevokeSession(ctx context.Context, token string) error {
	if token == "" {
		return unauthenticated()
	}
	return s.store.RevokeSession(ctx, tokenHash(token), s.now().UTC())
}

func VerificationCodeHash(id, code string) []byte {
	sum := sha256.Sum256([]byte(id + "\x00" + code))
	return sum[:]
}
func codeHash(id, code string) []byte { return VerificationCodeHash(id, code) }
func tokenHash(token string) []byte   { sum := sha256.Sum256([]byte(token)); return sum[:] }

func (s *Service) newSession() (string, []byte, time.Time, time.Time) {
	token := "sess_" + rand.Text() + rand.Text()
	now := s.now().UTC()
	return token, tokenHash(token), now, now.Add(30 * 24 * time.Hour)
}
func unauthenticated() error {
	return &Error{Code: "UNAUTHENTICATED", Message: "请先登录", Status: http.StatusUnauthorized}
}

func randomCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
