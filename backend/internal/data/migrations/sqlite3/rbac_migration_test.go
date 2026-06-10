package sqlite3

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	_ "github.com/sysadminsmedia/homebox/backend/pkgs/cgofreesqlite"
)

// TestRBACMigrationBackfill seeds a pre-RBAC database with a group owner, a
// plain member and a legacy superuser, applies the RBAC migration, and
// verifies the backfill: owners and superusers become Super Admins, members
// get no roles, and the dropped tables/columns are gone.
func TestRBACMigrationBackfill(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "rbac-migration.db")

	db, err := sql.Open("sqlite3", dbPath+"?_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}

	// Migrate to the state just before the RBAC migration.
	if err := goose.UpTo(db, ".", 20260512130000); err != nil {
		t.Fatalf("pre-RBAC migrations failed: %v", err)
	}

	seed := `
		INSERT INTO groups (id, created_at, updated_at, name, currency)
		VALUES ('10000000-0000-0000-0000-000000000001', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'Seed Home', 'usd');

		INSERT INTO users (id, created_at, updated_at, name, email, password, is_superuser, superuser, default_group_id, settings)
		VALUES
		  ('20000000-0000-0000-0000-000000000001', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'Owner', 'owner@example.com', 'x', false, false, '10000000-0000-0000-0000-000000000001', '{}'),
		  ('20000000-0000-0000-0000-000000000002', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'Member', 'member@example.com', 'x', false, false, '10000000-0000-0000-0000-000000000001', '{}'),
		  ('20000000-0000-0000-0000-000000000003', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'Legacy Super', 'super@example.com', 'x', true, false, '10000000-0000-0000-0000-000000000001', '{}');

		INSERT INTO user_groups (user_id, group_id, role)
		VALUES
		  ('20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'owner'),
		  ('20000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000001', 'user');
	`
	if _, err := db.Exec(seed); err != nil {
		t.Fatalf("seeding pre-RBAC data failed: %v", err)
	}

	// Apply the RBAC migration (and anything after it).
	if err := goose.Up(db, "."); err != nil {
		t.Fatalf("RBAC migration failed: %v", err)
	}

	countQuery := func(q string, args ...any) int {
		var n int
		if err := db.QueryRow(q, args...).Scan(&n); err != nil {
			t.Fatalf("query %q failed: %v", q, err)
		}
		return n
	}

	// Super Admin role seeded exactly once.
	if n := countQuery(`SELECT COUNT(*) FROM roles WHERE is_super_admin = true`); n != 1 {
		t.Fatalf("expected exactly 1 super admin role, got %d", n)
	}

	// Owner and legacy superuser backfilled; plain member not.
	superAdmins := `
		SELECT COUNT(*) FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE r.is_super_admin = true AND ur.user_id = ?`
	if n := countQuery(superAdmins, "20000000-0000-0000-0000-000000000001"); n != 1 {
		t.Error("group owner should have been granted Super Admin")
	}
	if n := countQuery(superAdmins, "20000000-0000-0000-0000-000000000003"); n != 1 {
		t.Error("legacy superuser should have been granted Super Admin")
	}
	if n := countQuery(superAdmins, "20000000-0000-0000-0000-000000000002"); n != 0 {
		t.Error("plain member should not have been granted Super Admin")
	}

	// Old structures gone.
	if n := countQuery(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name IN ('user_groups', 'group_invitation_tokens')`); n != 0 {
		t.Error("user_groups / group_invitation_tokens should be dropped")
	}
	if n := countQuery(`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name IN ('is_superuser', 'superuser')`); n != 0 {
		t.Error("users.is_superuser / users.superuser should be dropped")
	}

	// Users survived the table rebuild.
	if n := countQuery(`SELECT COUNT(*) FROM users`); n != 3 {
		t.Fatalf("expected 3 users after rebuild, got %d", n)
	}
}
