package session

type Role string

const (
	RoleOwner     Role = "owner"
	RoleAdmin     Role = "admin"
	RoleDeveloper Role = "developer"
	RoleMember    Role = "member"
)

const (
	ContextAccountKey = "authenticated_account"
	ContextUserKey    = "authenticated_user"
)

type TokenClaims struct {
	UserID    string `json:"user_id"`
	AccountID string `json:"account_id"`
	Email     string `json:"email"`
	Role      Role   `json:"role"`
}
