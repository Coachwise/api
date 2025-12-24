package views

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func parsePagination(c *gin.Context, defaultLimit int, maxLimit int) (int, int) {
	limit := defaultLimit
	offset := 0
	if v, ok := c.GetQuery("limit"); ok {
		if parsed, err := strconv.Atoi(v); err == nil {
			limit = parsed
		}
	}
	if v, ok := c.GetQuery("offset"); ok {
		if parsed, err := strconv.Atoi(v); err == nil {
			offset = parsed
		}
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
