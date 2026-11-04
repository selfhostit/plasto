package app_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/selfhostit/plasto/app"
	_ "github.com/selfhostit/plasto/migrations"
)

// newApp starts an empty app with the plasto migrations and routes.
func newApp(t testing.TB) *tests.TestApp {
	testApp, err := tests.NewTestAppWithConfig(core.BaseAppConfig{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	app.Register(testApp, app.Config{PublicDir: "testdata"})
	return testApp
}

// addUser saves a user with the given role and returns an auth token for them.
func addUser(t testing.TB, a core.App, email, role string) (*core.Record, string) {
	users, err := a.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	u := core.NewRecord(users)
	u.SetEmail(email)
	u.SetPassword("password123")
	u.Set("role", role)
	if err := a.Save(u); err != nil {
		t.Fatal(err)
	}
	token, err := u.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	return u, token
}

// newSetupApp starts an empty app whose setup code is ABCD-EFGH.
func newSetupApp(t testing.TB) *tests.TestApp {
	a := newApp(t)
	if err := os.WriteFile(filepath.Join(a.DataDir(), app.SetupCodeFile), []byte("ABCD-EFGH\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestSetup(t *testing.T) {
	scenarios := []tests.ApiScenario{
		{
			Name:            "a new install needs setup",
			Method:          http.MethodGet,
			URL:             "/api/app/setup",
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"needed":true`},
			TestAppFactory:  newApp,
		},
		{
			Name:           "setup creates the owner and an admin account",
			Method:         http.MethodPost,
			URL:            "/api/app/setup",
			Body:           strings.NewReader(`{"name":" Ada ","email":"Ada@Example.com","password":"password123","code":"abcd efgh"}`),
			ExpectedStatus: http.StatusNoContent,
			TestAppFactory: newSetupApp,
			AfterTestFunc: func(t testing.TB, a *tests.TestApp, _ *http.Response) {
				if _, err := os.Stat(filepath.Join(a.DataDir(), app.SetupCodeFile)); !os.IsNotExist(err) {
					t.Errorf("the setup code is still there: %v", err)
				}
				owner, err := a.FindAuthRecordByEmail("users", "ada@example.com")
				if err != nil {
					t.Fatal(err)
				}
				if owner.GetString("name") != "Ada" || owner.GetString("role") != "owner" || !owner.Verified() {
					t.Errorf("owner = %v", owner)
				}
				if _, err := a.FindAuthRecordByEmail(core.CollectionNameSuperusers, "ada@example.com"); err != nil {
					t.Errorf("no admin account: %v", err)
				}
			},
		},
		{
			Name:            "setup refuses a wrong code",
			Method:          http.MethodPost,
			URL:             "/api/app/setup",
			Body:            strings.NewReader(`{"name":"Eve","email":"eve@example.com","password":"password123","code":"ABCD-EFGX"}`),
			ExpectedStatus:  http.StatusForbidden,
			ExpectedContent: []string{"setup code is wrong"},
			TestAppFactory:  newSetupApp,
			AfterTestFunc: func(t testing.TB, a *tests.TestApp, _ *http.Response) {
				if n, _ := a.CountRecords("users"); n != 0 {
					t.Errorf("%d users after a wrong code", n)
				}
			},
		},
		{
			Name:            "setup refuses a missing code",
			Method:          http.MethodPost,
			URL:             "/api/app/setup",
			Body:            strings.NewReader(`{"name":"Eve","email":"eve@example.com","password":"password123"}`),
			ExpectedStatus:  http.StatusForbidden,
			ExpectedContent: []string{"setup code is wrong"},
			TestAppFactory:  newApp,
		},
		{
			Name:            "setup refuses a short password",
			Method:          http.MethodPost,
			URL:             "/api/app/setup",
			Body:            strings.NewReader(`{"name":"Ada","email":"ada@example.com","password":"short"}`),
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{"at least 8 characters"},
			TestAppFactory:  newApp,
		},
		{
			Name:            "setup only runs once",
			Method:          http.MethodPost,
			URL:             "/api/app/setup",
			Body:            strings.NewReader(`{"name":"Eve","email":"eve@example.com","password":"password123"}`),
			ExpectedStatus:  http.StatusForbidden,
			ExpectedContent: []string{"already set up"},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				a := newApp(t)
				addUser(t, a, "ada@example.com", "owner")
				return a
			},
		},
	}
	for _, s := range scenarios {
		s.Test(t)
	}
}

func TestUserRules(t *testing.T) {
	signup := `{"email":"new@example.com","password":"password123","passwordConfirm":"password123","role":"member"}`
	scenarios := []struct {
		name, as, method, url, body string
		status                      int
		content                     string
	}{
		{"nobody can sign up", "", http.MethodPost, "/api/collections/users/records", signup, http.StatusBadRequest, ""},
		{"members cannot add users", "member", http.MethodPost, "/api/collections/users/records", signup, http.StatusBadRequest, ""},
		{"the owner adds users", "owner", http.MethodPost, "/api/collections/users/records", signup, http.StatusOK, `"role":"member"`},
		{"members rename themselves", "member", http.MethodPatch, "/api/collections/users/records/{member}", `{"name":"Bob"}`, http.StatusOK, `"name":"Bob"`},
		{"members cannot change their role", "member", http.MethodPatch, "/api/collections/users/records/{member}", `{"role":"owner"}`, http.StatusNotFound, ""},
		{"members cannot remove anyone", "member", http.MethodDelete, "/api/collections/users/records/{member}", "", http.StatusNotFound, ""},
		{"members see each other", "member", http.MethodGet, "/api/collections/users/records", "", http.StatusOK, `"totalItems":2`},
		{"signed-out visitors see nobody", "", http.MethodGet, "/api/collections/users/records", "", http.StatusOK, `"totalItems":0`},
	}

	for _, sc := range scenarios {
		a := newApp(t)
		_, ownerToken := addUser(t, a, "owner@example.com", "owner")
		member, memberToken := addUser(t, a, "member@example.com", "member")
		headers := map[string]string{}
		switch sc.as {
		case "owner":
			headers["Authorization"] = ownerToken
		case "member":
			headers["Authorization"] = memberToken
		}
		s := tests.ApiScenario{
			Name:           sc.name,
			Method:         sc.method,
			URL:            strings.ReplaceAll(sc.url, "{member}", member.Id),
			Body:           strings.NewReader(sc.body),
			Headers:        headers,
			ExpectedStatus: sc.status,
			TestAppFactory: func(testing.TB) *tests.TestApp { return a },
		}
		if sc.content != "" {
			s.ExpectedContent = []string{sc.content}
		} else {
			s.NotExpectedContent = []string{`"id":`}
		}
		s.Test(t)
	}
}

func TestNewInstallMakesASetupCode(t *testing.T) {
	a := newApp(t)
	s := tests.ApiScenario{
		Method:          http.MethodGet,
		URL:             "/api/app/setup",
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"needed":true`},
		TestAppFactory:  func(testing.TB) *tests.TestApp { return a },
		AfterTestFunc: func(t testing.TB, a *tests.TestApp, _ *http.Response) {
			b, err := os.ReadFile(filepath.Join(a.DataDir(), app.SetupCodeFile))
			if err != nil {
				t.Fatal(err)
			}
			if code := strings.TrimSpace(string(b)); len(code) != 9 || code[4] != '-' {
				t.Errorf("setup code = %q", code)
			}
		},
	}
	s.Test(t)
}
