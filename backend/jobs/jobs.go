// Package jobs runs scheduled work, like PocketBase's cron, with two extras:
// every run is recorded in the "jobs" collection (visible in the admin UI),
// and a job whose time came while the server was off runs once when it starts
// again, so a daily job is never silently skipped.
//
// From app.Register:
//
//	jobs.MustAdd(app, "daily-digest", "0 7 * * *", func(app core.App) error {
//		return sendDigest(app)
//	})
//
// Schedules use the server's time zone (the TZ environment variable).
package jobs

import (
	"fmt"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/cron"
)

// Collection records the last run of each job. Only superusers can read it.
const Collection = "jobs"

// Func does the work; a returned error is logged and recorded.
type Func func(app core.App) error

// how far back a missed run is looked for
const lookback = 366 * 24 * time.Hour

var running sync.Map // job name -> *sync.Mutex

// MustAdd schedules fn with a cron expression such as "0 3 * * *" (03:00 every
// day). It panics if the expression is invalid.
func MustAdd(app core.App, name, cronExpr string, fn Func) {
	schedule, err := cron.NewSchedule(cronExpr)
	if err != nil {
		panic(fmt.Sprintf("job %q: %v", name, err))
	}
	runs := &background{}
	app.Cron().MustAdd(name, cronExpr, func() {
		runs.do(func() { Run(app, name, fn) })
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		now := time.Now()
		go runs.do(func() { catchUp(se.App, name, schedule, fn, now) })
		return se.Next()
	})

	// the app waits for a running job before it closes its database, so a run
	// never writes to a closed app
	app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		runs.stop()
		return e.Next()
	})
}

// background tracks one job's runs so the app can wait for them when it stops.
// Once stopped, new runs are skipped.
type background struct {
	mu      sync.Mutex
	wg      sync.WaitGroup
	stopped bool
}

func (b *background) do(f func()) {
	b.mu.Lock()
	if b.stopped {
		b.mu.Unlock()
		return
	}
	b.wg.Add(1)
	b.mu.Unlock()
	defer b.wg.Done()
	f()
}

func (b *background) stop() {
	b.mu.Lock()
	b.stopped = true
	b.mu.Unlock()
	b.wg.Wait()
}

// Run runs a job now and records the result. A run is skipped while the
// previous one is still going.
func Run(app core.App, name string, fn Func) {
	mu, _ := running.LoadOrStore(name, &sync.Mutex{})
	if !mu.(*sync.Mutex).TryLock() {
		app.Logger().Warn("Job still running, skipping this run", "job", name)
		return
	}
	defer mu.(*sync.Mutex).Unlock()

	start := time.Now()
	err := safely(app, fn)
	if err != nil {
		app.Logger().Error("Job failed", "job", name, "error", err)
	} else {
		app.Logger().Info("Job done", "job", name, "took", time.Since(start).Round(time.Millisecond))
	}
	if err := record(app, name, start, time.Since(start), err); err != nil {
		app.Logger().Error("Could not record job run", "job", name, "error", err)
	}
}

func safely(app core.App, fn Func) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return fn(app)
}

func record(app core.App, name string, start time.Time, took time.Duration, runErr error) error {
	rec, err := app.FindFirstRecordByData(Collection, "name", name)
	if err != nil {
		col, err := app.FindCollectionByNameOrId(Collection)
		if err != nil {
			return err
		}
		rec = core.NewRecord(col)
		rec.Set("name", name)
	}
	rec.Set("last_run", start)
	rec.Set("last_duration_ms", took.Milliseconds())
	rec.Set("last_error", "")
	if runErr != nil {
		rec.Set("last_error", runErr.Error())
	}
	return app.Save(rec)
}

// catchUp runs the job if a scheduled time passed since its last run. A job
// seen for the first time is only recorded, so adding one doesn't run it.
func catchUp(app core.App, name string, schedule *cron.Schedule, fn Func, now time.Time) {
	rec, err := app.FindFirstRecordByData(Collection, "name", name)
	if err != nil {
		col, err := app.FindCollectionByNameOrId(Collection)
		if err != nil {
			app.Logger().Error("Jobs collection missing", "error", err)
			return
		}
		rec = core.NewRecord(col)
		rec.Set("name", name)
		rec.Set("last_run", now)
		if err := app.Save(rec); err != nil {
			app.Logger().Error("Could not record new job", "job", name, "error", err)
		}
		return
	}
	last := rec.GetDateTime("last_run").Time()
	if missed(schedule, last, now) {
		app.Logger().Info("Catching up on a missed job run", "job", name, "last_run", last)
		Run(app, name, fn)
	}
}

// missed reports whether the schedule was due at any minute after last, up to
// and including now, in the server's time zone.
func missed(schedule *cron.Schedule, last, now time.Time) bool {
	if last.IsZero() || now.Sub(last) > lookback {
		last = now.Add(-lookback)
	}
	for t := last.Truncate(time.Minute).Add(time.Minute); !t.After(now); t = t.Add(time.Minute) {
		if schedule.IsDue(cron.NewMoment(t.In(time.Local))) {
			return true
		}
	}
	return false
}
