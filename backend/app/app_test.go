package app_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/selfhostit/plasto/app"
)

func TestRoutes(t *testing.T) {
	setup := func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp()
		if err != nil {
			t.Fatal(err)
		}
		app.Register(testApp, app.Config{PublicDir: "testdata"})
		return testApp
	}

	scenarios := []tests.ApiScenario{
		{
			Name:            "info reports the version",
			Method:          http.MethodGet,
			URL:             "/api/app/info",
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"version":"dev"`},
			TestAppFactory:  setup,
		},
		{
			Name:            "unknown paths fall back to the web app",
			Method:          http.MethodGet,
			URL:             "/some/client/route",
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{"plasto test page"},
			TestAppFactory:  setup,
		},
		{
			Name:            "the web app only runs its own scripts",
			Method:          http.MethodGet,
			URL:             "/",
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{"plasto test page"},
			TestAppFactory:  setup,
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
				if csp := res.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'self';") {
					t.Errorf("Content-Security-Policy = %q", csp)
				}
			},
		},
		{
			Name:            "the API has no page policy",
			Method:          http.MethodGet,
			URL:             "/api/app/info",
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"version"`},
			TestAppFactory:  setup,
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
				if csp := res.Header.Get("Content-Security-Policy"); csp != "" {
					t.Errorf("Content-Security-Policy = %q", csp)
				}
			},
		},
	}

	for _, s := range scenarios {
		s.Test(t)
	}
}
