package httpapi

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func ParseUUIDParam(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, errors.New("invalid uuid")
	}
	return id, nil
}

func ParsePaging(c *gin.Context) (int, int, error) {
	limit := 50
	offset := 0
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, errors.New("invalid limit")
		}
		limit = v
	}
	if raw := c.Query("offset"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, errors.New("invalid offset")
		}
		offset = v
	}
	return limit, offset, nil
}

func ParsePositiveInt(raw string) (int, error) {
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return 0, errors.New("invalid number")
	}
	return v, nil
}

func ParseOptionalRFC3339(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, errors.New("invalid time format, expected RFC3339")
	}
	return &ts, nil
}
