package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// The owner manages everyone else from the People screen: they see their
// email addresses and can set a new password without knowing the old one.
// For their own account the owner still needs their current password.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		users.ManageRule = types.Pointer(`@request.auth.role = "owner" && id != @request.auth.id`)
		return app.Save(users)
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		users.ManageRule = nil
		return app.Save(users)
	})
}
