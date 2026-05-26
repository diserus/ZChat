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

func TestService_Register_Success(t *testing.T) {
	ctx := context.Background()

	users := authmocks.NewMockUserRepository(t)
	tokens := authmocks.NewMockTokenRepository(t)
	jwt := authmocks.NewMockTokenGenerator(t)
	svc := auth.NewService(users, tokens, jwt, zap.NewNop())

	input := auth.RegisterInput{
		Email:    "neo@example.com",
		Name:     "Neo",
		Password: "secret123",
	}
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

	users.EXPECT().
		GetByEmail(ctx, input.Email).
		Return(nil, nil)

	users.EXPECT().
		Create(ctx, mock.MatchedBy(func(user *auth.User) bool {
			require.Equal(t, input.Email, user.Email)
			require.Equal(t, input.Name, user.Name)
			require.NotEmpty(t, user.EncryptedPassword)
			require.True(t, user.CheckPassword(input.Password))
			return true
		})).
		Return(nil)

	jwt.EXPECT().
		GenerateTokenPair(mock.AnythingOfType("*auth.User")).
		Return(pair, nil)

	jwt.EXPECT().
		NewRefreshToken(mock.AnythingOfType("uuid.UUID"), pair.RefreshToken).
		Return(refreshToken)

	tokens.EXPECT().
		Create(ctx, refreshToken).
		Return(nil)

	out, err := svc.Register(ctx, input)

	require.NoError(t, err)
	require.Equal(t, pair.AccessToken, out.AccessToken)
	require.Equal(t, pair.RefreshToken, out.RefreshToken)
	require.Equal(t, input.Email, out.User.Email)
	require.Equal(t, input.Name, out.User.Name)
}

func TestService_Register_EmailAlreadyExists(t *testing.T) {
	ctx := context.Background()

	users := authmocks.NewMockUserRepository(t)
	tokens := authmocks.NewMockTokenRepository(t)
	jwt := authmocks.NewMockTokenGenerator(t)
	svc := auth.NewService(users, tokens, jwt, zap.NewNop())

	input := auth.RegisterInput{
		Email:    "neo@example.com",
		Name:     "Neo",
		Password: "secret123",
	}
	existingUser := &auth.User{
		ID:    uuid.New(),
		Email: input.Email,
		Name:  "Existing Neo",
	}

	users.EXPECT().
		GetByEmail(ctx, input.Email).
		Return(existingUser, nil)

	out, err := svc.Register(ctx, input)

	require.ErrorIs(t, err, apperror.ErrUserAlreadyExists)
	require.Nil(t, out)
}

func TestService_Register_GetByEmailError(t *testing.T) {
	ctx := context.Background()

	users := authmocks.NewMockUserRepository(t)
	tokens := authmocks.NewMockTokenRepository(t)
	jwt := authmocks.NewMockTokenGenerator(t)
	svc := auth.NewService(users, tokens, jwt, zap.NewNop())

	input := auth.RegisterInput{
		Email:    "neo@example.com",
		Name:     "Neo",
		Password: "secret123",
	}
	repoErr := errors.New("database is down")

	users.EXPECT().
		GetByEmail(ctx, input.Email).
		Return(nil, repoErr)
	out, err := svc.Register(ctx, input)
	require.ErrorIs(t, err, repoErr)
	require.ErrorContains(t, err, "check email")
	require.Nil(t, out)
}

func TestService_Register_InvalidUser(t *testing.T) {
	ctx := context.Background()

	users := authmocks.NewMockUserRepository(t)
	tokens := authmocks.NewMockTokenRepository(t)
	jwt := authmocks.NewMockTokenGenerator(t)
	svc := auth.NewService(users, tokens, jwt, zap.NewNop())

	input := auth.RegisterInput{
		Email:    "neo@example.com",
		Name:     "",
		Password: "secret123",
	}

	users.EXPECT().
		GetByEmail(ctx, input.Email).
		Return(nil, nil)

	out, err := svc.Register(ctx, input)

	require.ErrorContains(t, err, "name is required")
	require.Nil(t, out)
}

func TestService_Register_WeakPassword(t *testing.T) {
	ctx := context.Background()

	users := authmocks.NewMockUserRepository(t)
	tokens := authmocks.NewMockTokenRepository(t)
	jwt := authmocks.NewMockTokenGenerator(t)
	svc := auth.NewService(users, tokens, jwt, zap.NewNop())

	input := auth.RegisterInput{
		Email:    "neo@example.com",
		Name:     "Neo",
		Password: "s",
	}

	users.EXPECT().
		GetByEmail(ctx, input.Email).
		Return(nil, nil)

	out, err := svc.Register(ctx, input)
	require.ErrorContains(t, err, "password must")
	require.Nil(t, out)
}

func TestService_Register_CreateUserError(t *testing.T) {
	ctx := context.Background()

	users := authmocks.NewMockUserRepository(t)
	tokens := authmocks.NewMockTokenRepository(t)
	jwt := authmocks.NewMockTokenGenerator(t)
	svc := auth.NewService(users, tokens, jwt, zap.NewNop())

	input := auth.RegisterInput{
		Email:    "neo@example.com",
		Name:     "Neo",
		Password: "secret123",
	}

	users.EXPECT().
		GetByEmail(ctx, input.Email).
		Return(nil, nil)

	createErr := errors.New("database is down")

	users.EXPECT().
		Create(ctx, mock.MatchedBy(func(user *auth.User) bool {
			require.Equal(t, input.Email, user.Email)
			require.Equal(t, input.Name, user.Name)
			require.NotEmpty(t, user.EncryptedPassword)
			require.True(t, user.CheckPassword(input.Password))
			return true
		})).
		Return(createErr)
	out, err := svc.Register(ctx, input)

	require.ErrorIs(t, err, createErr)
	require.ErrorContains(t, err, "create user")
	require.Nil(t, out)
}

func TestService_Register_GenerateTokenPairError(t *testing.T) {
	ctx := context.Background()

	users := authmocks.NewMockUserRepository(t)
	tokens := authmocks.NewMockTokenRepository(t)
	jwt := authmocks.NewMockTokenGenerator(t)
	svc := auth.NewService(users, tokens, jwt, zap.NewNop())

	input := auth.RegisterInput{
		Email:    "neo@example.com",
		Name:     "Neo",
		Password: "secret123",
	}

	users.EXPECT().
		GetByEmail(ctx, input.Email).
		Return(nil, nil)
	users.EXPECT().
		Create(ctx, mock.MatchedBy(func(user *auth.User) bool {
			return user.Email == input.Email &&
				user.Name == input.Name &&
				user.CheckPassword(input.Password)
		})).
		Return(nil)
	tokenErr := errors.New("token error")
	jwt.EXPECT().GenerateTokenPair(mock.AnythingOfType("*auth.User")).Return(nil, tokenErr)

	out, err := svc.Register(ctx, input)

	require.ErrorIs(t, err, tokenErr)
	require.ErrorContains(t, err, "generate tokens")
	require.Nil(t, out)
}

func TestService_Register_SaveRefreshTokenError(t *testing.T) {
	ctx := context.Background()

	users := authmocks.NewMockUserRepository(t)
	tokens := authmocks.NewMockTokenRepository(t)
	jwt := authmocks.NewMockTokenGenerator(t)
	svc := auth.NewService(users, tokens, jwt, zap.NewNop())

	input := auth.RegisterInput{
		Email:    "neo@example.com",
		Name:     "Neo",
		Password: "secret123",
	}
	users.EXPECT().
		GetByEmail(ctx, input.Email).
		Return(nil, nil)
	users.EXPECT().
		Create(ctx, mock.MatchedBy(func(user *auth.User) bool {
			return user.Email == input.Email &&
				user.Name == input.Name &&
				user.CheckPassword(input.Password)
		})).
		Return(nil)

	tokenPair := &auth.TokenPair{
		AccessToken:  "access",
		RefreshToken: "refresh",
	}
	refreshToken := &auth.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: "hash",
		ExpiresAt: time.Now().Add(time.Hour),
		Revoked:   false,
		CreatedAt: time.Now(),
	}
	tokenErr := errors.New("refresh error")
	jwt.EXPECT().GenerateTokenPair(mock.AnythingOfType("*auth.User")).Return(tokenPair, nil)
	jwt.EXPECT().NewRefreshToken(mock.AnythingOfType("uuid.UUID"), tokenPair.RefreshToken).Return(refreshToken)
	tokens.EXPECT().Create(ctx, refreshToken).Return(tokenErr)

	out, err := svc.Register(ctx, input)
	require.ErrorIs(t, err, tokenErr)
	require.ErrorContains(t, err, "save refresh token")
	require.Nil(t, out)
}
