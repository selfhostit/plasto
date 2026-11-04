// Package app holds everything that makes this PocketBase server your app:
// routes, hooks and scheduled jobs. Register wires them up.
package app

import (
	"net/http"
	"os"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/selfhostit/plasto/push"
)

// Version is set at build time with -ldflags "-X github.com/selfhostit/plasto/app.Version=...".
var Version = "dev"

// Config holds the settings that come from command-line flags.
type Config struct {
	// PublicDir is the directory with the built web app.
	PublicDir string
}

// Register adds the app's routes and hooks to a PocketBase instance.
func Register(app core.App, cfg Config) {
	// scheduled jobs and backups follow the server's time zone (TZ)
	app.Cron().SetTimezone(time.Local)

	registerDemo(app) // DEMO

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/app/info", info)
		if err := registerSetupRoutes(se); err != nil {
			return err
		}
		registerWorkerRoutes(se)
		push.RegisterRoutes(se)

		// Serve the web app, falling back to index.html for client-side routes.
		if !se.Router.HasRoute(http.MethodGet, "/{path...}") {
			se.Router.GET("/{path...}", apis.Static(os.DirFS(cfg.PublicDir), true)).BindFunc(secureHeaders)
		}
		return se.Next()
	})
}

// info tells the web app which build it is talking to.
func info(e *core.RequestEvent) error {
	return e.JSON(http.StatusOK, map[string]string{"version": Version})
}
