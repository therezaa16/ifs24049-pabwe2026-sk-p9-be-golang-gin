// Package response menstandarkan bentuk JSON { status, message, data? }.
package response

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Body adalah amplop respons standar.
type Body struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Success membalas 2xx dengan status "success".
func Success(c *gin.Context, code int, message string, data any) {
	c.JSON(code, Body{Status: "success", Message: message, Data: data})
}

// Fail membalas 4xx dengan status "fail".
func Fail(c *gin.Context, code int, message string) {
	c.JSON(code, Body{Status: "fail", Message: message})
}

// Error membalas 5xx dengan status "error".
func Error(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, Body{Status: "error", Message: message})
}

// AbortFail seperti Fail namun menghentikan rantai middleware.
func AbortFail(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, Body{Status: "fail", Message: message})
}

// AbortError seperti Error namun menghentikan rantai middleware.
func AbortError(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, Body{Status: "error", Message: message})
}

// BindMessage menerjemahkan galat binding/validasi menjadi pesan yang ramah.
func BindMessage(err error) string {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		parts := make([]string, 0, len(verrs))
		for _, fe := range verrs {
			parts = append(parts, "field '"+strings.ToLower(fe.Field())+"' tidak valid ("+fe.Tag()+")")
		}
		return "Validasi gagal: " + strings.Join(parts, ", ")
	}
	return "Format permintaan tidak valid"
}
