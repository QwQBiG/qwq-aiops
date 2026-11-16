package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUnknownResourceSubpathsCannotReadOrMutateRecords(t *testing.T) {
	oldSites, oldUsers, oldRoles := websitesStore.Websites, usersStore.Users, rolesStore.Roles
	t.Cleanup(func() {
		websitesStore.Websites, usersStore.Users, rolesStore.Roles = oldSites, oldUsers, oldRoles
	})
	tests := []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/api/websites/41/unknown", handleWebsiteDetail},
		{"/api/websites/41/ssl/apply/unknown", handleWebsiteDetail},
		{"/api/users/41/unknown", handleUserDetail},
		{"/api/users/41/permissions/unknown", handleUserDetail},
		{"/api/roles/41/unknown", handleRoleDetail},
	}
	for _, test := range tests {
		for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
			t.Run(method+test.path, func(t *testing.T) {
				websitesStore.Websites = []Website{{ID: 41, Domain: "original.test"}}
				usersStore.Users = []User{{ID: 41, Username: "original"}}
				rolesStore.Roles = []Role{{ID: 41, Name: "original"}}
				response := httptest.NewRecorder()
				test.handler(response, httptest.NewRequest(method, test.path, strings.NewReader(`{"enabled":false,"name":"changed","username":"changed","backend_url":"changed"}`)))
				if response.Code != http.StatusNotFound {
					t.Errorf("unknown resource path returned %d, want 404", response.Code)
				}
				if len(websitesStore.Websites) != 1 || len(usersStore.Users) != 1 || len(rolesStore.Roles) != 1 {
					t.Fatal("unknown resource path deleted a record")
				}
				if websitesStore.Websites[0].BackendURL != "" || usersStore.Users[0].Username != "original" || rolesStore.Roles[0].Name != "original" {
					t.Fatal("unknown resource path modified a record")
				}
			})
		}
	}
}
