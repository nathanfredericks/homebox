package postgres

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestRBACMigrationBackfillPostgres(t *testing.T) {
	dsn := os.Getenv("HBX_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("HBX_PG_TEST_DSN not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}

	if err := goose.UpTo(db, ".", 20260512130001); err != nil {
		t.Fatalf("pre-RBAC migrations failed: %v", err)
	}

	seed := `
		INSERT INTO groups (id, created_at, updated_at, name, currency)
		VALUES ('10000000-0000-0000-0000-000000000001', NOW(), NOW(), 'Seed Home', 'usd');

		INSERT INTO users (id, created_at, updated_at, name, email, password, is_superuser, superuser, default_group_id, settings)
		VALUES
		  ('20000000-0000-0000-0000-000000000001', NOW(), NOW(), 'Owner', 'owner@example.com', 'x', false, false, '10000000-0000-0000-0000-000000000001', '{}'),
		  ('20000000-0000-0000-0000-000000000002', NOW(), NOW(), 'Member', 'member@example.com', 'x', false, false, '10000000-0000-0000-0000-000000000001', '{}'),
		  ('20000000-0000-0000-0000-000000000003', NOW(), NOW(), 'Legacy Super', 'super@example.com', 'x', true, false, '10000000-0000-0000-0000-000000000001', '{}');

		INSERT INTO user_groups (user_id, group_id, role)
		VALUES
		  ('20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'owner'),
		  ('20000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000001', 'user');
	`
	if _, err := db.Exec(seed); err != nil {
		t.Fatalf("seeding failed: %v", err)
	}

	if err := goose.Up(db, "."); err != nil {
		t.Fatalf("RBAC migration failed: %v", err)
	}

	count := func(q string, args ...any) int {
		var n int
		if err := db.QueryRow(q, args...).Scan(&n); err != nil {
			t.Fatalf("query %q failed: %v", q, err)
		}
		return n
	}

	if n := count(`SELECT COUNT(*) FROM roles WHERE is_super_admin = true`); n != 1 {
		t.Fatalf("expected 1 super admin role, got %d", n)
	}
	superAdmins := `SELECT COUNT(*) FROM user_roles ur JOIN roles r ON r.id = ur.role_id WHERE r.is_super_admin = true AND ur.user_id = $1`
	if n := count(superAdmins, "20000000-0000-0000-0000-000000000001"); n != 1 {
		t.Error("owner should be super admin")
	}
	if n := count(superAdmins, "20000000-0000-0000-0000-000000000003"); n != 1 {
		t.Error("legacy superuser should be super admin")
	}
	if n := count(superAdmins, "20000000-0000-0000-0000-000000000002"); n != 0 {
		t.Error("member should not be super admin")
	}
	if n := count(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('user_groups', 'group_invitation_tokens')`); n != 0 {
		t.Error("old tables should be dropped")
	}
	if n := count(`SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'users' AND column_name IN ('is_superuser', 'superuser')`); n != 0 {
		t.Error("old columns should be dropped")
	}
}
