package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/domain"
	"github.com/luli-tech/twilio-Boss/pkg/logger"
)

const (
	ContextAccountKey = "authenticated_account"
	ContextUserKey    = "authenticated_user"
)

// Auth authenticates requests using Twilio-style HTTP Basic Auth (AccountSID:AuthToken) or Bearer Token.
func Auth(accountRepo domain.AccountRepository, authServices ...domain.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		username, password, hasBasic := c.Request.BasicAuth()
		var accountSID, authToken string

		if hasBasic {
			accountSID = username
			authToken = password
		} else {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
				if len(authServices) > 0 && authServices[0] != nil {
					claims, err := authServices[0].ValidateToken(c.Request.Context(), tokenString)
					if err != nil {
						c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
							"code":    "INVALID_TOKEN",
							"message": "Invalid or expired bearer token",
						})
						return
					}

					acc, err := accountRepo.GetByID(c.Request.Context(), claims.AccountID)
					if err != nil || acc == nil || !acc.IsActive() {
						c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
							"code":    "INVALID_CREDENTIALS",
							"message": "Invalid bearer token account",
						})
						return
					}

					c.Set(ContextAccountKey, acc)
					c.Set(ContextUserKey, claims)
					c.Set(string(logger.AccountSIDKey), acc.SID)
					c.Next()
					return
				}
				authToken = tokenString
			}
		}

		if accountSID == "" && authToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Authentication credentials were not provided",
			})
			return
		}

		var acc *domain.Account
		var err error

		if accountSID != "" {
			acc, err = accountRepo.GetBySID(c.Request.Context(), accountSID)
		}

		if err != nil || acc == nil {
			if errors.Is(err, domain.ErrAccountNotFound) || acc == nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"code":    "INVALID_CREDENTIALS",
					"message": "Invalid Account SID or Auth Token",
				})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Authentication service failure",
			})
			return
		}

		// Verify AuthToken
		if acc.AuthToken != authToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "INVALID_CREDENTIALS",
				"message": "Invalid Account SID or Auth Token",
			})
			return
		}

		if !acc.IsActive() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    "ACCOUNT_SUSPENDED",
				"message": "Account has been suspended or closed",
			})
			return
		}

		// Inject into gin context
		c.Set(ContextAccountKey, acc)
		c.Set(string(logger.AccountSIDKey), acc.SID)

		c.Next()
	}
}

// GetTokenClaims helper retrieves authenticated JWT user claims from gin context.
func GetTokenClaims(c *gin.Context) *domain.TokenClaims {
	val, exists := c.Get(ContextUserKey)
	if !exists {
		return nil
	}
	claims, ok := val.(*domain.TokenClaims)
	if !ok {
		return nil
	}
	return claims
}

// GetAccount helper retrieves authenticated Account from gin context.
func GetAccount(c *gin.Context) *domain.Account {
	val, exists := c.Get(ContextAccountKey)
	if !exists {
		return nil
	}
	acc, ok := val.(*domain.Account)
	if !ok {
		return nil
	}
	return acc
}
