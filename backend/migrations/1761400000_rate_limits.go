package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Rate limits on signing in and on first-run setup, so a password can't be
// guessed by trying many (PocketBase ships with rate limits off). They count
// per address; behind a reverse proxy or the Tailscale sidecar every visitor
// shares the proxy's address, so the limits then hold for everyone together,
// which still stops guessing. Change them in the admin UI under Settings >
// Rate limits; installs that turned them on already are left alone.
func init() {
	m.Register(func(app core.App) error {
		settings := app.Settings()
		if settings.RateLimits.Enabled {
			return nil
		}
		settings.RateLimits.Enabled = true
		settings.RateLimits.Rules = []core.RateLimitRule{
			// password sign-ins of users, workers and the admin UI
			{Label: "*:auth", MaxRequests: 10, Duration: 60},
			{Label: "POST /api/app/setup", MaxRequests: 5, Duration: 60},
		}
		return app.Save(settings)
	}, nil)
}
