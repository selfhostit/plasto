package app

import "github.com/pocketbase/pocketbase/core"

// contentSecurityPolicy lets the web app run only its own scripts, so a script
// that gets into a page (through a stored note, say) can't run or send data away.
// Styles may be inline because Svelte sets some; images may come from blob: and
// data: URLs for previews made in the browser.
const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' blob: data:; media-src 'self' blob:; connect-src 'self'; worker-src 'self'; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// secureHeaders adds browser protections to the web app's files. The API and
// PocketBase's admin UI (/_/) are left as they are.
func secureHeaders(e *core.RequestEvent) error {
	h := e.Response.Header()
	h.Set("Content-Security-Policy", contentSecurityPolicy)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "same-origin")
	return e.Next()
}
