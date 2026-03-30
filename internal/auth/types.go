package auth

// RegisterRequest is the payload for POST /auth/register.
type RegisterRequest struct {
	Name     string `json:"name"     validate:"required,min=1,max=255"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// RegisterResponse is returned on successful registration.
type RegisterResponse struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Score int    `json:"score"`
}
