package auth

type registerRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Name     string `json:"name"     binding:"required,min=2"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	User         UserDTO `json:"user"`
}

type UserDTO struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
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
