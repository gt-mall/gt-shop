package auth

import (
	"strings"

	"github.com/gin-gonic/gin"

	"mall/admin_core/internal/model"
	"mall/admin_core/pkg/response"
)

const ContextMemberKey = "authenticated_member"

func (s *Service) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := requestToken(c)
		if token == "" {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}
		member, err := s.Authenticate(c.Request.Context(), token)
		if err != nil {
			response.Unauthorized(c, err.Error())
			c.Abort()
			return
		}
		c.Set(ContextMemberKey, member)
		c.Next()
	}
}

func CurrentMember(c *gin.Context) (model.Member, bool) {
	value, ok := c.Get(ContextMemberKey)
	if !ok {
		return model.Member{}, false
	}
	member, ok := value.(model.Member)
	return member, ok
}

func requestToken(c *gin.Context) string {
	authorization := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
		return strings.TrimSpace(authorization[7:])
	}
	return strings.TrimSpace(c.GetHeader("token"))
}
