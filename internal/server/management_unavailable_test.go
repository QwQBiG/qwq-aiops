package server

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestUnavailableCertificateActionsDoNotChangeWebsite(t *testing.T) {
	old := websitesStore.Websites
	t.Cleanup(func() { websitesStore.Websites = old })
	websitesStore.Websites = []Website{{ID: 41, Domain: "example.test", SSLEnabled: false}}
	before := websitesStore.Websites[0]
	for _, action := range []string{"apply", "renew"} {
		w := httptest.NewRecorder()
		handleWebsiteSSL(w, httptest.NewRequest(http.MethodPost, "/", nil), 41, action)
		if w.Code != http.StatusNotImplemented || !strings.Contains(w.Body.String(), `"error"`) {
			t.Fatalf("%s: %d %s", action, w.Code, w.Body.String())
		}
		if websitesStore.Websites[0] != before {
			t.Fatal("unsupported certificate action changed website state")
		}
	}
}

func TestUnavailableUserPermissionsDoNotClaimAuthorization(t *testing.T) {
	old := usersStore.Users
	t.Cleanup(func() { usersStore.Users = old })
	usersStore.Users = []User{{ID: 42, Username: "operator", Roles: []string{"viewer"}}}
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		w := httptest.NewRecorder()
		handleUserPermissions(w, httptest.NewRequest(method, "/", strings.NewReader(`{"permissions":["users:write"]}`)), 42)
		if w.Code != http.StatusNotImplemented || !strings.Contains(w.Body.String(), `"error"`) {
			t.Fatalf("%s: %d %s", method, w.Code, w.Body.String())
		}
		if !reflect.DeepEqual(usersStore.Users[0].Roles, []string{"viewer"}) {
			t.Fatal("unsupported permission action changed user roles")
		}
	}
}

func TestRuntimeRepairIsExplicitlyUnavailable(t *testing.T) {
	w := httptest.NewRecorder()
	handleDeploymentRepair(w, httptest.NewRequest(http.MethodPost, "/api/deployment/repair", nil))
	if w.Code != http.StatusNotImplemented || !strings.Contains(w.Body.String(), `"error"`) {
		t.Fatalf("unconnected repair returned %d %s", w.Code, w.Body.String())
	}
}

func TestConcurrentEmptyWebsiteListingDoesNotMutateStore(t *testing.T) {
	old := websitesStore.Websites
	t.Cleanup(func() { websitesStore.Websites = old })
	websitesStore.Websites = nil
	var readers sync.WaitGroup
	for i := 0; i < 20; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			w := httptest.NewRecorder()
			handleWebsites(w, httptest.NewRequest(http.MethodGet, "/api/websites", nil))
			if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != "[]" {
				t.Errorf("empty website listing returned %d %s", w.Code, w.Body.String())
			}
		}()
	}
	readers.Wait()
	if websitesStore.Websites != nil {
		t.Fatal("read-only listing initialized shared website state")
	}
}
