package app

import (
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// First-run setup: until a user exists, the web app shows a setup screen
// instead of the sign-in form, so nobody needs a terminal or the admin UI to
// get started. That first user becomes the owner and also gets an admin UI
// account. Once anyone exists the endpoint refuses, so it only ever runs once.
//
// Setup asks for a one-time code that the server prints to its log and keeps
// in pb_data/setup_code, where deploy/install.sh reads it. Only someone who
// can see the machine can set it up, so a stranger who finds a fresh install
// on the internet can't make themselves its owner.

type setupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Code     string `json:"code"`
}

// SetupCodeFile holds the setup code inside the data folder until setup is done.
const SetupCodeFile = "setup_code"

// letters and digits that can't be mistaken for each other
const setupCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// setupCode returns the code that first-run setup asks for, making it the
// first time.
func setupCode(app core.App) (string, error) {
	path := filepath.Join(app.DataDir(), SetupCodeFile)
	if b, err := os.ReadFile(path); err == nil {
		if code := strings.TrimSpace(string(b)); code != "" {
			return code, nil
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = setupCodeAlphabet[int(b[i])%len(setupCodeAlphabet)]
	}
	code := string(b[:4]) + "-" + string(b[4:])
	return code, os.WriteFile(path, []byte(code+"\n"), 0o600)
}

// sameSetupCode compares codes the way people type them: any case, with or
// without the dash or spaces.
func sameSetupCode(typed, code string) bool {
	norm := func(s string) []byte {
		return []byte(strings.NewReplacer("-", "", " ", "").Replace(strings.ToUpper(s)))
	}
	return subtle.ConstantTimeCompare(norm(typed), norm(code)) == 1
}

var setupMu sync.Mutex

func setupNeeded(app core.App) (bool, error) {
	n, err := app.CountRecords("users")
	return n == 0, err
}

func registerSetupRoutes(se *core.ServeEvent) error {
	if needed, err := setupNeeded(se.App); err != nil {
		return err
	} else if needed {
		code, err := setupCode(se.App)
		if err != nil {
			return err
		}
		// on stdout, so `docker compose logs` shows it
		fmt.Printf("\nFirst-run setup code: %s\nThe setup screen asks for it when you create the first account.\n\n", code)
	}

	se.Router.GET("/api/app/setup", func(e *core.RequestEvent) error {
		needed, err := setupNeeded(e.App)
		if err != nil {
			return e.InternalServerError("", err)
		}
		return e.JSON(http.StatusOK, map[string]bool{"needed": needed})
	})

	se.Router.POST("/api/app/setup", func(e *core.RequestEvent) error {
		var body setupRequest
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Fill in your name, email and password.", err)
		}
		body.Name = strings.TrimSpace(body.Name)
		body.Email = strings.ToLower(strings.TrimSpace(body.Email))
		if body.Name == "" || body.Email == "" {
			return e.BadRequestError("Fill in your name and email.", nil)
		}
		if len(body.Password) < 8 {
			return e.BadRequestError("The password needs at least 8 characters.", nil)
		}

		setupMu.Lock()
		defer setupMu.Unlock()
		needed, err := setupNeeded(e.App)
		if err != nil {
			return e.InternalServerError("", err)
		}
		if !needed {
			return e.ForbiddenError("This app is already set up.", nil)
		}
		code, err := setupCode(e.App)
		if err != nil {
			return e.InternalServerError("", err)
		}
		if !sameSetupCode(body.Code, code) {
			return e.ForbiddenError("That setup code is wrong. The installer shows it, and so does the app's log (docker compose logs app).", nil)
		}

		if err := e.App.RunInTransaction(func(tx core.App) error { return createOwner(tx, body) }); err != nil {
			return e.BadRequestError("Could not save: "+err.Error(), err)
		}
		e.App.Logger().Info("First-run setup done", "owner", body.Email)
		_ = os.Remove(filepath.Join(e.App.DataDir(), SetupCodeFile))
		return e.NoContent(http.StatusNoContent)
	})
	return nil
}

// createOwner adds the owner and, unless a real one exists already, an admin
// UI account with the same email and password.
func createOwner(tx core.App, req setupRequest) error {
	users, err := tx.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	owner := core.NewRecord(users)
	owner.Set("name", req.Name)
	owner.Set("role", "owner")
	owner.SetEmail(req.Email)
	owner.SetPassword(req.Password)
	owner.SetVerified(true)
	if err := tx.Save(owner); err != nil {
		return err
	}

	// PocketBase keeps a placeholder installer account until a real admin
	// exists; saving one removes it.
	admins, err := tx.CountRecords(core.CollectionNameSuperusers,
		dbx.Not(dbx.HashExp{"email": core.DefaultInstallerEmail}))
	if err != nil || admins > 0 {
		return err
	}
	superusers, err := tx.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		return err
	}
	admin := core.NewRecord(superusers)
	admin.SetEmail(req.Email)
	admin.SetPassword(req.Password)
	return tx.Save(admin)
}
