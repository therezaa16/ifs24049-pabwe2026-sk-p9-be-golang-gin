package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func ctx() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	var m map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &m))
	return m
}

func TestSuccess(t *testing.T) {
	c, w := ctx()
	Success(c, http.StatusCreated, "ok", map[string]string{"a": "b"})
	m := decode(t, w)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "success", m["status"])
	assert.NotNil(t, m["data"])
}

func TestSuccessWithoutData(t *testing.T) {
	c, w := ctx()
	Success(c, http.StatusOK, "ok", nil)
	_, has := decode(t, w)["data"]
	assert.False(t, has)
}

func TestFailAndError(t *testing.T) {
	c, w := ctx()
	Fail(c, http.StatusBadRequest, "bad")
	assert.Equal(t, "fail", decode(t, w)["status"])

	c, w = ctx()
	Error(c, "boom")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "error", decode(t, w)["status"])
}

func TestAbort(t *testing.T) {
	c, w := ctx()
	AbortFail(c, http.StatusUnauthorized, "no")
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	c, w = ctx()
	AbortError(c, "x")
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestBindMessage(t *testing.T) {
	type in struct {
		Topic string `validate:"required"`
	}
	err := validator.New().Struct(in{})
	assert.Contains(t, BindMessage(err), "topic")
	assert.Equal(t, "Format permintaan tidak valid", BindMessage(assert.AnError))
}
