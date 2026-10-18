package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUserHandlersRejectPasswordsOver72Bytes(t *testing.T) {
	previousMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(previousMode) })

	for _, password := range []struct {
		name  string
		value string
	}{
		{"ASCII", strings.Repeat("a", 73)},
		{"UTF8", strings.Repeat("中", 25)},
	} {
		for _, handler := range []struct {
			name string
			call gin.HandlerFunc
		}{
			{"signup", SignUpHandler},
			{"login", LoginHandler},
		} {
			t.Run(handler.name+"/"+password.name, func(t *testing.T) {
				body, err := json.Marshal(map[string]string{
					"username":   "password_limit_test",
					"password":   password.value,
					"repassword": password.value,
				})
				if err != nil {
					t.Fatal(err)
				}
				recorder := httptest.NewRecorder()
				context, _ := gin.CreateTestContext(recorder)
				context.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
				context.Request.Header.Set("Content-Type", "application/json")

				handler.call(context)

				var response struct{ Code ResCode }
				if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.Code != CodeInvalidParam {
					t.Fatalf("code = %d, want %d", response.Code, CodeInvalidParam)
				}
			})
		}
	}
}
