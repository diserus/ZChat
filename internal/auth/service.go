package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"zchat/internal/apperror"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}

type TokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	GetByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

type TokenGenerator interface {
	GenerateTokenPair(user *User) (*TokenPair, error)
	NewRefreshToken(userID uuid.UUID, tokenStr string) *RefreshToken
	ValidateAccessToken(tokenStr string) (*Claims, error)
	ValidateRefreshToken(tokenStr string) (*Claims, error)
	HashToken(token string) string
}

type RegisterInput struct {
	Email    string
	Name     string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type RefreshInput struct {
	RefreshToken string
}

type Output struct {
	AccessToken  string
	RefreshToken string
	User         *User
}

type Service struct {
	users  UserRepository
	tokens TokenRepository
	jwt    TokenGenerator
	log    *zap.Logger
}

func NewService(users UserRepository, tokens TokenRepository, jwt TokenGenerator, log *zap.Logger) *Service {
	return &Service{users: users, tokens: tokens, jwt: jwt, log: log}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*Output, error) {
	existing, err := s.users.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		return nil, apperror.ErrUserAlreadyExists
	}

	user := &User{Email: input.Email, Name: input.Name}
	if err = user.Validate(); err != nil {
		return nil, err
	}
	if err = user.SetPassword(input.Password); err != nil {
		return nil, err
	}
	if err = s.users.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return s.createSession(ctx, user)
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*Output, error) {
	user, err := s.users.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil || !user.CheckPassword(input.Password) {
		return nil, apperror.ErrInvalidCredentials
	}
	return s.createSession(ctx, user)
}

func (s *Service) Refresh(ctx context.Context, input RefreshInput) (*Output, error) {
	claims, err := s.jwt.ValidateRefreshToken(input.RefreshToken)
	if err != nil {
		return nil, apperror.ErrInvalidToken
	}
	token, err := s.tokens.GetByHash(ctx, s.jwt.HashToken(input.RefreshToken))
	if err != nil || token == nil || token.Revoked {
		return nil, apperror.ErrInvalidToken
	}
	if err = s.tokens.Revoke(ctx, token.ID); err != nil {
		return nil, fmt.Errorf("revoke token: %w", err)
	}
	user, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil || user == nil {
		return nil, apperror.ErrInvalidToken
	}
	return s.createSession(ctx, user)
}

func (s *Service) Logout(ctx context.Context, input RefreshInput) error {
	if _, err := s.jwt.ValidateRefreshToken(input.RefreshToken); err != nil {
		return apperror.ErrInvalidToken
	}
	token, err := s.tokens.GetByHash(ctx, s.jwt.HashToken(input.RefreshToken))
	if err != nil || token == nil {
		return apperror.ErrInvalidToken
	}
	return s.tokens.Revoke(ctx, token.ID)
}

// UserExists is the cross-context query used by chat/group services to verify
// that a target user is real before creating relationships with them.
func (s *Service) UserExists(ctx context.Context, id uuid.UUID) (bool, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	return user != nil, nil
}

func (s *Service) createSession(ctx context.Context, user *User) (*Output, error) {
	pair, err := s.jwt.GenerateTokenPair(user)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}
	rt := s.jwt.NewRefreshToken(user.ID, pair.RefreshToken)
	if err = s.tokens.Create(ctx, rt); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}
	return &Output{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		User:         user,
	}, nil
}
