package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"durg-voter-api/pkg/models"
)

func TestAuthMiddlewareAdminRole(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUserFromContext(r.Context())
		if !ok {
			t.Fatal("expected user in context")
		}
		if user.Role != models.RoleAdmin {
			t.Fatalf("expected admin role, got %s", user.Role)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/v1/admin/sql", nil)
	req.Header.Set("Authorization", "Bearer admin-token-secret-key-12345")
	rr := httptest.NewRecorder()

	handler := Authenticate(nextHandler)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}
}

func TestAuthMiddlewareGuestRole(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUserFromContext(r.Context())
		if !ok {
			t.Fatal("expected user in context")
		}
		if user.Role != models.RoleGuest {
			t.Fatalf("expected guest role, got %s", user.Role)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/v1/voters", nil)
	req.Header.Set("Authorization", "Bearer guest-token-secret-key-67890")
	rr := httptest.NewRecorder()

	handler := Authenticate(nextHandler)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}
}

func TestRequireRoleAdminDeniedForGuest(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/v1/admin/sql", nil)
	req.Header.Set("Authorization", "Bearer guest-token-secret-key-67890")
	rr := httptest.NewRecorder()

	handler := Authenticate(RequireRole(models.RoleAdmin)(nextHandler))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden for guest user on admin route, got %d", rr.Code)
	}
}
