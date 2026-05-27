package auth_test

import (
	"testing"
	"zchat/internal/auth"

	"github.com/stretchr/testify/require"
)

func TestUser_Validate_EmptyEmail(t *testing.T) {
	user := &auth.User{
		Name: "homelander",
	}
	err := user.Validate()
	require.ErrorContains(t, err, "email is required")
}

func TestUser_Validate_EmptyName(t *testing.T) {
	user := &auth.User{
		Email: "homelander@gmail.com",
	}
	err := user.Validate()
	require.ErrorContains(t, err, "name is required")
}

func TestValidate_Success(t *testing.T) {
	user := &auth.User{
		Name:  "homelander",
		Email: "homelander@gmail.com",
	}
	err := user.Validate()
	require.NoError(t, err)
}

func TestSetPassword_ShortPassword(t *testing.T) {
	user := &auth.User{}
	err := user.SetPassword("123")
	require.ErrorContains(t, err, "password must be at least 6 characters")
}

func TestSetPassword_Success(t *testing.T) {
	user := &auth.User{}
	err := user.SetPassword("1234567")
	require.NoError(t, err)
	require.NotEmpty(t, user.EncryptedPassword)
	require.NotEqual(t, "1234567", user.EncryptedPassword)
	require.True(t, user.CheckPassword("1234567"))
	require.False(t, user.CheckPassword("wrong-password"))
}

func TestUser_CheckPassword(t *testing.T) {
	user := &auth.User{}
	require.NoError(t, user.SetPassword("secret123"))

	require.True(t, user.CheckPassword("secret123"))
	require.False(t, user.CheckPassword("wrong-password"))
}
