package httpapi

import (
	"errors"
	"net/http"

	"zchat/internal/apperror"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func ErrorJSON(err error) ErrorResponse {
	return ErrorResponse{Error: err.Error()}
}

// RespondError maps a domain error to the appropriate HTTP status using
// the sentinels in the apperror package.
func RespondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, apperror.ErrValidation):
		c.JSON(http.StatusBadRequest, ErrorJSON(err))
	case errors.Is(err, apperror.ErrForbidden):
		c.JSON(http.StatusForbidden, ErrorJSON(err))
	case errors.Is(err, apperror.ErrNotFound):
		c.JSON(http.StatusNotFound, ErrorJSON(err))
	default:
		c.JSON(http.StatusInternalServerError, ErrorJSON(err))
	}
}
