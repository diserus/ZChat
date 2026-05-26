package auth

// RegisterRequest (note: renamed to public)
type RegisterRequest struct {
	Email    string `json:"email"    binding:"required,email" example:"user@example.com" description:"User email address"`
	Name     string `json:"name"     binding:"required,min=2" example:"John Doe" description:"Display name (min 2 chars)"`
	Password string `json:"password" binding:"required,min=8" example:"secret123" description:"Password (min 8 chars)" format:"password"`
}

// LoginRequest
type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"secret123" format:"password"`
}

// RefreshRequest
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// AuthResponse for login/refresh
type AuthResponse struct {
	AccessToken  string  `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string  `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	User         UserDTO `json:"user"`
}

// UserDTO
type UserDTO struct {
	ID    string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email string `json:"email" example:"user@example.com"`
	Name  string `json:"name" example:"John Doe"`
}

func toAuthResponse(out *Output) AuthResponse {
	return AuthResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		User: UserDTO{
			ID:    out.User.ID.String(),
			Email: out.User.Email,
			Name:  out.User.Name,
		},
	}
}
