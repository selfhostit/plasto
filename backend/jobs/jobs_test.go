package jobs

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/cron"

	_ "github.com/selfhostit/plasto/migrations"
)

func mustSchedule(t *testing.T, expr string) *cron.Schedule {
	s, err := cron.NewSchedule(expr)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestMissed(t *testing.T) {
	time.Local = time.UTC
	daily := mustSchedule(t, "0 3 * * *")
	at := func(s string) time.Time {
		v, err := time.Parse("2006-01-02 15:04", s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}

	cases := []struct {
		name      string
		last, now string
		want      bool
	}{
		{"ran today, next run is tomorrow", "2026-10-19 03:00", "2026-10-19 20:00", false},
		{"server was off over 03:00", "2026-10-18 03:00", "2026-10-19 09:00", true},
		{"restarted just before the run", "2026-10-18 03:00", "2026-10-19 02:59", false},
		{"restarted at exactly the run time", "2026-10-18 03:00", "2026-10-19 03:00", true},
		{"off for over a year", "2024-01-01 00:00", "2026-10-19 09:00", true},
	}
	for _, c := range cases {
		if got := missed(daily, at(c.last), at(c.now)); got != c.want {
			t.Errorf("%s: missed = %v, want %v", c.name, got, c.want)
		}
	}
}

func newApp(t *testing.T) *tests.TestApp {
	a, err := tests.NewTestAppWithConfig(core.BaseAppConfig{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.Cleanup)
	return a
}

func lastRun(t *testing.T, a core.App, name string) *core.Record {
	rec, err := a.FindFirstRecordByData(Collection, "name", name)
	if err != nil {
		t.Fatalf("no record for %s: %v", name, err)
	}
	return rec
}

func TestCatchUp(t *testing.T) {
	time.Local = time.UTC
	a := newApp(t)
	daily := mustSchedule(t, "0 3 * * *")
	now := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	runs := 0
	fn := func(core.App) error { runs++; return nil }

	// a new job is only recorded
	catchUp(a, "digest", daily, fn, now)
	if runs != 0 {
		t.Fatalf("a new job ran %d times", runs)
	}
	lastRun(t, a, "digest")

	// the server was off at 03:00: the job runs once on start
	rec := lastRun(t, a, "digest")
	rec.Set("last_run", now.Add(-30*time.Hour))
	if err := a.Save(rec); err != nil {
		t.Fatal(err)
	}
	catchUp(a, "digest", daily, fn, now)
	if runs != 1 {
		t.Fatalf("missed job ran %d times, want 1", runs)
	}

	// it ran just now, so another start doesn't run it again
	catchUp(a, "digest", daily, fn, time.Now())
	if runs != 1 {
		t.Fatalf("job ran again after catching up (%d runs)", runs)
	}
}

func TestRunRecordsFailures(t *testing.T) {
	a := newApp(t)

	Run(a, "broken", func(core.App) error { return errors.New("feed unreachable") })
	if got := lastRun(t, a, "broken").GetString("last_error"); got != "feed unreachable" {
		t.Errorf("last_error = %q", got)
	}

	Run(a, "broken", func(core.App) error { panic("nil map") })
	if got := lastRun(t, a, "broken").GetString("last_error"); got != "panic: nil map" {
		t.Errorf("last_error after a panic = %q", got)
	}

	Run(a, "broken", func(core.App) error { return nil })
	if got := lastRun(t, a, "broken").GetString("last_error"); got != "" {
		t.Errorf("last_error after success = %q", got)
	}
}

func TestRunSkipsOverlappingRuns(t *testing.T) {
	a := newApp(t)
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		Run(a, "slow", func(core.App) error {
			close(started)
			<-release
			return nil
		})
		close(done)
	}()
	<-started
	ran := false
	Run(a, "slow", func(core.App) error { ran = true; return nil })
	close(release)
	<-done
	if ran {
		t.Error("a second run started while the first was still going")
	}
}

func TestBackupsAreOnByDefault(t *testing.T) {
	a := newApp(t)
	if b := a.Settings().Backups; b.Cron != "0 3 * * *" || b.CronMaxKeep != 7 {
		t.Errorf("backups = %q keeping %d", b.Cron, b.CronMaxKeep)
	}
}

func TestMustAddCatchesUpWhenTheServerStarts(t *testing.T) {
	ran := make(chan struct{}, 1)
	scenario := tests.ApiScenario{
		Method:          http.MethodGet,
		URL:             "/api/health",
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{"healthy"},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			a, err := tests.NewTestAppWithConfig(core.BaseAppConfig{DataDir: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			col, _ := a.FindCollectionByNameOrId(Collection)
			rec := core.NewRecord(col)
			rec.Set("name", "every-minute")
			rec.Set("last_run", time.Now().Add(-time.Hour))
			if err := a.Save(rec); err != nil {
				t.Fatal(err)
			}
			MustAdd(a, "every-minute", "* * * * *", func(core.App) error {
				ran <- struct{}{}
				return nil
			})
			return a
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, _ *http.Response) {
			select {
			case <-ran:
			case <-time.After(5 * time.Second):
				t.Error("the missed job did not run when the server started")
			}
		},
	}
	scenario.Test(t)
}

func TestSignInIsRateLimited(t *testing.T) {
	a := newApp(t)
	r := a.Settings().RateLimits
	rule, ok := r.FindRateLimitRule([]string{"_superusers:auth", "*:auth"})
	if !r.Enabled || !ok || rule.MaxRequests != 10 || rule.Duration != 60 {
		t.Errorf("rate limits = %+v", r)
	}
}
