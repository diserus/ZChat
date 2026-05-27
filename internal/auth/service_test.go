package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"zchat/internal/apperror"
	"zchat/internal/auth"
	authmocks "zchat/internal/auth/mocks"
)

type serviceFixture struct {
	ctx    context.Context
	users  *authmocks.MockUserRepository
	tokens *authmocks.MockTokenRepository
	jwt    *authmocks.MockTokenGenerator
	svc    *auth.Service
}

func newServiceFixture(t *testing.T) *serviceFixture {
	t.Helper()

	users := authmocks.NewMockUserRepository(t)
	tokens := authmocks.NewMockTokenRepository(t)
	jwt := authmocks.NewMockTokenGenerator(t)

	return &serviceFixture{
		ctx:    context.Background(),
		users:  users,
		tokens: tokens,
		jwt:    jwt,
		svc:    auth.NewService(users, tokens, jwt, zap.NewNop()),
	}
}

func newUser(t *testing.T, email, password string) *auth.User {
	t.Helper()

	user := &auth.User{
		ID:    uuid.New(),
		Email: email,
		Name:  "Neo",
	}
	require.NoError(t, user.SetPassword(password))

	return user
}

func expectSession(f *serviceFixture, user interface{}) *auth.TokenPair {
	pair := &auth.TokenPair{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}
	refreshToken := &auth.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: "refresh-token-hash",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	f.jwt.EXPECT().
		GenerateTokenPair(user).
		Return(pair, nil)
	f.jwt.EXPECT().
		NewRefreshToken(mock.AnythingOfType("uuid.UUID"), pair.RefreshToken).
		Return(refreshToken)
	f.tokens.EXPECT().
		Create(f.ctx, refreshToken).
		Return(nil)

	return pair
}

func expectSessionError(f *serviceFixture, user interface{}, err error) {
	f.jwt.EXPECT().
		GenerateTokenPair(user).
		Return(nil, err)
}

func TestService_Register_Success(t *testing.T) {
	f := newServiceFixture(t)
	input := auth.RegisterInput{
		Email:    "neo@example.com",
		Name:     "Neo",
		Password: "secret123",
	}

	f.users.EXPECT().GetByEmail(f.ctx, input.Email).Return(nil, nil)
	f.users.EXPECT().
		Create(f.ctx, mock.MatchedBy(func(user *auth.User) bool {
			return user.Email == input.Email &&
				user.Name == input.Name &&
				user.EncryptedPassword != "" &&
				user.CheckPassword(input.Password)
		})).
		Return(nil)
	pair := expectSession(f, mock.AnythingOfType("*auth.User"))

	out, err := f.svc.Register(f.ctx, input)

	require.NoError(t, err)
	require.Equal(t, pair.AccessToken, out.AccessToken)
	require.Equal(t, pair.RefreshToken, out.RefreshToken)
	require.Equal(t, input.Email, out.User.Email)
	require.Equal(t, input.Name, out.User.Name)
}

func TestService_Register_EmailAlreadyExists(t *testing.T) {
	f := newServiceFixture(t)
	input := auth.RegisterInput{Email: "neo@example.com", Name: "Neo", Password: "secret123"}

	f.users.EXPECT().
		GetByEmail(f.ctx, input.Email).
		Return(&auth.User{ID: uuid.New(), Email: input.Email}, nil)

	out, err := f.svc.Register(f.ctx, input)

	require.ErrorIs(t, err, apperror.ErrUserAlreadyExists)
	require.Nil(t, out)
}

func TestService_Register_GetByEmailError(t *testing.T) {
	f := newServiceFixture(t)
	input := auth.RegisterInput{Email: "neo@example.com", Name: "Neo", Password: "secret123"}
	repoErr := errors.New("database is down")

	f.users.EXPECT().GetByEmail(f.ctx, input.Email).Return(nil, repoErr)

	out, err := f.svc.Register(f.ctx, input)

	require.ErrorIs(t, err, repoErr)
	require.ErrorContains(t, err, "check email")
	require.Nil(t, out)
}

func TestService_Register_InvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		input     auth.RegisterInput
		wantError string
	}{
		{
			name:      "missing name",
			input:     auth.RegisterInput{Email: "neo@example.com", Password: "secret123"},
			wantError: "name is required",
		},
		{
			name:      "weak password",
			input:     auth.RegisterInput{Email: "neo@example.com", Name: "Neo", Password: "s"},
			wantError: "password must",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newServiceFixture(t)

			f.users.EXPECT().GetByEmail(f.ctx, tt.input.Email).Return(nil, nil)

			out, err := f.svc.Register(f.ctx, tt.input)

			require.ErrorContains(t, err, tt.wantError)
			require.Nil(t, out)
		})
	}
}

func TestService_Register_CreateUserError(t *testing.T) {
	f := newServiceFixture(t)
	input := auth.RegisterInput{Email: "neo@example.com", Name: "Neo", Password: "secret123"}
	createErr := errors.New("database is down")

	f.users.EXPECT().GetByEmail(f.ctx, input.Email).Return(nil, nil)
	f.users.EXPECT().
		Create(f.ctx, mock.MatchedBy(func(user *auth.User) bool {
			return user.Email == input.Email && user.CheckPassword(input.Password)
		})).
		Return(createErr)

	out, err := f.svc.Register(f.ctx, input)

	require.ErrorIs(t, err, createErr)
	require.ErrorContains(t, err, "create user")
	require.Nil(t, out)
}

func TestService_Register_GenerateTokenPairError(t *testing.T) {
	f := newServiceFixture(t)
	input := auth.RegisterInput{Email: "neo@example.com", Name: "Neo", Password: "secret123"}
	tokenErr := errors.New("token error")

	f.users.EXPECT().GetByEmail(f.ctx, input.Email).Return(nil, nil)
	f.users.EXPECT().Create(f.ctx, mock.AnythingOfType("*auth.User")).Return(nil)
	expectSessionError(f, mock.AnythingOfType("*auth.User"), tokenErr)

	out, err := f.svc.Register(f.ctx, input)

	require.ErrorIs(t, err, tokenErr)
	require.ErrorContains(t, err, "generate tokens")
	require.Nil(t, out)
}

func TestService_Register_SaveRefreshTokenError(t *testing.T) {
	f := newServiceFixture(t)
	input := auth.RegisterInput{Email: "neo@example.com", Name: "Neo", Password: "secret123"}
	tokenPair := &auth.TokenPair{AccessToken: "access", RefreshToken: "refresh"}
	refreshToken := &auth.RefreshToken{ID: uuid.New(), UserID: uuid.New(), TokenHash: "hash"}
	saveErr := errors.New("refresh error")

	f.users.EXPECT().GetByEmail(f.ctx, input.Email).Return(nil, nil)
	f.users.EXPECT().Create(f.ctx, mock.AnythingOfType("*auth.User")).Return(nil)
	f.jwt.EXPECT().GenerateTokenPair(mock.AnythingOfType("*auth.User")).Return(tokenPair, nil)
	f.jwt.EXPECT().NewRefreshToken(mock.AnythingOfType("uuid.UUID"), tokenPair.RefreshToken).Return(refreshToken)
	f.tokens.EXPECT().Create(f.ctx, refreshToken).Return(saveErr)

	out, err := f.svc.Register(f.ctx, input)

	require.ErrorIs(t, err, saveErr)
	require.ErrorContains(t, err, "save refresh token")
	require.Nil(t, out)
}

func TestService_Login_Success(t *testing.T) {
	f := newServiceFixture(t)
	input := auth.LoginInput{Email: "neo@example.com", Password: "secret123"}
	user := newUser(t, input.Email, input.Password)

	f.users.EXPECT().GetByEmail(f.ctx, input.Email).Return(user, nil)
	pair := expectSession(f, user)

	out, err := f.svc.Login(f.ctx, input)

	require.NoError(t, err)
	require.Equal(t, pair.AccessToken, out.AccessToken)
	require.Equal(t, pair.RefreshToken, out.RefreshToken)
	require.Equal(t, user, out.User)
}

func TestService_Login_GetUserError(t *testing.T) {
	f := newServiceFixture(t)
	input := auth.LoginInput{Email: "neo@example.com", Password: "secret123"}
	repoErr := errors.New("database is down")

	f.users.EXPECT().GetByEmail(f.ctx, input.Email).Return(nil, repoErr)

	out, err := f.svc.Login(f.ctx, input)

	require.ErrorIs(t, err, repoErr)
	require.ErrorContains(t, err, "get user")
	require.Nil(t, out)
}

func TestService_Login_InvalidCredentials(t *testing.T) {
	tests := []struct {
		name string
		user *auth.User
	}{
		{name: "user not found", user: nil},
		{name: "wrong password", user: newUser(t, "neo@example.com", "right-password")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newServiceFixture(t)
			input := auth.LoginInput{Email: "neo@example.com", Password: "wrong-password"}

			f.users.EXPECT().GetByEmail(f.ctx, input.Email).Return(tt.user, nil)

			out, err := f.svc.Login(f.ctx, input)

			require.ErrorIs(t, err, apperror.ErrInvalidCredentials)
			require.Nil(t, out)
		})
	}
}

func TestService_Login_CreateSessionError(t *testing.T) {
	f := newServiceFixture(t)
	input := auth.LoginInput{Email: "neo@example.com", Password: "secret123"}
	user := newUser(t, input.Email, input.Password)
	tokenErr := errors.New("token error")

	f.users.EXPECT().GetByEmail(f.ctx, input.Email).Return(user, nil)
	expectSessionError(f, user, tokenErr)

	out, err := f.svc.Login(f.ctx, input)

	require.ErrorIs(t, err, tokenErr)
	require.ErrorContains(t, err, "generate tokens")
	require.Nil(t, out)
}

func TestService_Refresh_Success(t *testing.T) {
	f := newServiceFixture(t)
	input := auth.RefreshInput{RefreshToken: "refresh-token"}
	user := newUser(t, "neo@example.com", "secret123")
	token := &auth.RefreshToken{ID: uuid.New(), UserID: user.ID, TokenHash: "refresh-token-hash"}

	f.jwt.EXPECT().ValidateRefreshToken(input.RefreshToken).Return(&auth.Claims{UserID: user.ID}, nil)
	f.jwt.EXPECT().HashToken(input.RefreshToken).Return(token.TokenHash)
	f.tokens.EXPECT().GetByHash(f.ctx, token.TokenHash).Return(token, nil)
	f.tokens.EXPECT().Revoke(f.ctx, token.ID).Return(nil)
	f.users.EXPECT().GetByID(f.ctx, user.ID).Return(user, nil)
	pair := expectSession(f, user)

	out, err := f.svc.Refresh(f.ctx, input)

	require.NoError(t, err)
	require.Equal(t, pair.AccessToken, out.AccessToken)
	require.Equal(t, pair.RefreshToken, out.RefreshToken)
	require.Equal(t, user, out.User)
}

func TestService_Refresh_InvalidToken(t *testing.T) {
	f := newServiceFixture(t)
	input := auth.RefreshInput{RefreshToken: "bad-refresh-token"}

	f.jwt.EXPECT().
		ValidateRefreshToken(input.RefreshToken).
		Return(nil, errors.New("invalid token"))

	out, err := f.svc.Refresh(f.ctx, input)

	require.ErrorIs(t, err, apperror.ErrInvalidToken)
	require.Nil(t, out)
}

func TestService_Refresh_StoredTokenInvalid(t *testing.T) {
	tests := []struct {
		name  string
		token *auth.RefreshToken
		err   error
	}{
		{name: "lookup error", err: errors.New("database is down")},
		{name: "not found"},
		{name: "revoked", token: &auth.RefreshToken{ID: uuid.New(), Revoked: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newServiceFixture(t)
			input := auth.RefreshInput{RefreshToken: "refresh-token"}
			tokenHash := "refresh-token-hash"

			f.jwt.EXPECT().ValidateRefreshToken(input.RefreshToken).Return(&auth.Claims{UserID: uuid.New()}, nil)
			f.jwt.EXPECT().HashToken(input.RefreshToken).Return(tokenHash)
			f.tokens.EXPECT().GetByHash(f.ctx, tokenHash).Return(tt.token, tt.err)

			out, err := f.svc.Refresh(f.ctx, input)

			require.ErrorIs(t, err, apperror.ErrInvalidToken)
			require.Nil(t, out)
		})
	}
}

func TestService_Refresh_RevokeError(t *testing.T) {
	f := newServiceFixture(t)
	input := auth.RefreshInput{RefreshToken: "refresh-token"}
	token := &auth.RefreshToken{ID: uuid.New(), TokenHash: "refresh-token-hash"}
	revokeErr := errors.New("database is down")

	f.jwt.EXPECT().ValidateRefreshToken(input.RefreshToken).Return(&auth.Claims{UserID: uuid.New()}, nil)
	f.jwt.EXPECT().HashToken(input.RefreshToken).Return(token.TokenHash)
	f.tokens.EXPECT().GetByHash(f.ctx, token.TokenHash).Return(token, nil)
	f.tokens.EXPECT().Revoke(f.ctx, token.ID).Return(revokeErr)

	out, err := f.svc.Refresh(f.ctx, input)

	require.ErrorIs(t, err, revokeErr)
	require.ErrorContains(t, err, "revoke token")
	require.Nil(t, out)
}

func TestService_Refresh_UserInvalid(t *testing.T) {
	tests := []struct {
		name string
		user *auth.User
		err  error
	}{
		{name: "lookup error", err: errors.New("database is down")},
		{name: "not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newServiceFixture(t)
			input := auth.RefreshInput{RefreshToken: "refresh-token"}
			userID := uuid.New()
			token := &auth.RefreshToken{ID: uuid.New(), TokenHash: "refresh-token-hash"}

			f.jwt.EXPECT().ValidateRefreshToken(input.RefreshToken).Return(&auth.Claims{UserID: userID}, nil)
			f.jwt.EXPECT().HashToken(input.RefreshToken).Return(token.TokenHash)
			f.tokens.EXPECT().GetByHash(f.ctx, token.TokenHash).Return(token, nil)
			f.tokens.EXPECT().Revoke(f.ctx, token.ID).Return(nil)
			f.users.EXPECT().GetByID(f.ctx, userID).Return(tt.user, tt.err)

			out, err := f.svc.Refresh(f.ctx, input)

			require.ErrorIs(t, err, apperror.ErrInvalidToken)
			require.Nil(t, out)
		})
	}
}

func TestService_Logout_Success(t *testing.T) {
	f := newServiceFixture(t)
	input := auth.RefreshInput{RefreshToken: "refresh-token"}
	token := &auth.RefreshToken{ID: uuid.New(), TokenHash: "refresh-token-hash"}

	f.jwt.EXPECT().ValidateRefreshToken(input.RefreshToken).Return(&auth.Claims{UserID: uuid.New()}, nil)
	f.jwt.EXPECT().HashToken(input.RefreshToken).Return(token.TokenHash)
	f.tokens.EXPECT().GetByHash(f.ctx, token.TokenHash).Return(token, nil)
	f.tokens.EXPECT().Revoke(f.ctx, token.ID).Return(nil)

	err := f.svc.Logout(f.ctx, input)

	require.NoError(t, err)
}

func TestService_Logout_InvalidToken(t *testing.T) {
	f := newServiceFixture(t)
	input := auth.RefreshInput{RefreshToken: "bad-refresh-token"}

	f.jwt.EXPECT().
		ValidateRefreshToken(input.RefreshToken).
		Return(nil, errors.New("invalid token"))

	err := f.svc.Logout(f.ctx, input)

	require.ErrorIs(t, err, apperror.ErrInvalidToken)
}

func TestService_Logout_StoredTokenInvalid(t *testing.T) {
	tests := []struct {
		name  string
		token *auth.RefreshToken
		err   error
	}{
		{name: "lookup error", err: errors.New("database is down")},
		{name: "not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newServiceFixture(t)
			input := auth.RefreshInput{RefreshToken: "refresh-token"}
			tokenHash := "refresh-token-hash"

			f.jwt.EXPECT().ValidateRefreshToken(input.RefreshToken).Return(&auth.Claims{UserID: uuid.New()}, nil)
			f.jwt.EXPECT().HashToken(input.RefreshToken).Return(tokenHash)
			f.tokens.EXPECT().GetByHash(f.ctx, tokenHash).Return(tt.token, tt.err)

			err := f.svc.Logout(f.ctx, input)

			require.ErrorIs(t, err, apperror.ErrInvalidToken)
		})
	}
}

func TestService_Logout_RevokeError(t *testing.T) {
	f := newServiceFixture(t)
	input := auth.RefreshInput{RefreshToken: "refresh-token"}
	token := &auth.RefreshToken{ID: uuid.New(), TokenHash: "refresh-token-hash"}
	revokeErr := errors.New("database is down")

	f.jwt.EXPECT().ValidateRefreshToken(input.RefreshToken).Return(&auth.Claims{UserID: uuid.New()}, nil)
	f.jwt.EXPECT().HashToken(input.RefreshToken).Return(token.TokenHash)
	f.tokens.EXPECT().GetByHash(f.ctx, token.TokenHash).Return(token, nil)
	f.tokens.EXPECT().Revoke(f.ctx, token.ID).Return(revokeErr)

	err := f.svc.Logout(f.ctx, input)

	require.ErrorIs(t, err, revokeErr)
}

func TestService_UserExists(t *testing.T) {
	tests := []struct {
		name string
		user *auth.User
		err  error
		want bool
	}{
		{name: "found", user: &auth.User{ID: uuid.New()}, want: true},
		{name: "not found", want: false},
		{name: "repo error", err: errors.New("database is down"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newServiceFixture(t)
			userID := uuid.New()

			f.users.EXPECT().GetByID(f.ctx, userID).Return(tt.user, tt.err)

			exists, err := f.svc.UserExists(f.ctx, userID)

			require.Equal(t, tt.want, exists)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
