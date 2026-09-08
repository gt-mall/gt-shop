package auth

import (
	"strings"

	"github.com/gin-gonic/gin"

	"mall/admin_core/internal/model"
	"mall/admin_core/pkg/response"
)

const contextUserKey = "authenticated_user"

func (s *Service) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := requestToken(c)
		if token == "" {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}
		user, err := s.Authenticate(c.Request.Context(), token)
		if err != nil {
			response.Unauthorized(c, err.Error())
			c.Abort()
			return
		}
		c.Set(contextUserKey, user)
		c.Next()
	}
}

func (s *Service) Permission(permission string) gin.HandlerFunc {
	return s.PermissionAny(permission)
}

func (s *Service) PermissionAny(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}
		for _, permission := range permissions {
			allowed, err := s.Can(c.Request.Context(), user.ID, permission)
			if err != nil {
				response.Error(c, err.Error())
				c.Abort()
				return
			}
			if allowed {
				c.Next()
				return
			}
		}
		response.Error(c, "permission denied")
		c.Abort()
	}
}

func (s *Service) PermissionAll(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}
		for _, permission := range permissions {
			allowed, err := s.Can(c.Request.Context(), user.ID, permission)
			if err != nil {
				response.Error(c, err.Error())
				c.Abort()
				return
			}
			if !allowed {
				response.Error(c, "permission denied")
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (model.SystemUser, bool) {
	value, ok := c.Get(contextUserKey)
	if !ok {
		return model.SystemUser{}, false
	}
	user, ok := value.(model.SystemUser)
	return user, ok
}

func requestToken(c *gin.Context) string {
	authorization := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
		return strings.TrimSpace(authorization[7:])
	}
	return strings.TrimSpace(c.GetHeader("token"))
}
