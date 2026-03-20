package commands

import "testing"

func TestGooseTarget(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		dbURL      string
		wantDriver string
		wantConn   string
		wantErr    bool
	}{
		{
			name:       "postgres",
			dbURL:      "postgres://user:pass@localhost:5432/app?sslmode=disable",
			wantDriver: "postgres",
			wantConn:   "postgres://user:pass@localhost:5432/app?sslmode=disable",
		},
		{
			name:       "mysql",
			dbURL:      "mysql://user:pass@localhost:3306/app",
			wantDriver: "mysql",
			wantConn:   "mysql://user:pass@localhost:3306/app",
		},
		{
			name:       "sqlite prefix",
			dbURL:      "sqlite://file:dev.db?_fk=1",
			wantDriver: "sqlite3",
			wantConn:   "file:dev.db?_fk=1",
		},
		{
			name:       "sqlite3 prefix",
			dbURL:      "sqlite3://dev.db",
			wantDriver: "sqlite3",
			wantConn:   "dev.db",
		},
		{
			name:    "unsupported scheme",
			dbURL:   "sqlserver://localhost",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			driver, conn, err := gooseTarget(tc.dbURL)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if driver != tc.wantDriver {
				t.Fatalf("driver = %q, want %q", driver, tc.wantDriver)
			}
			if conn != tc.wantConn {
				t.Fatalf("conn = %q, want %q", conn, tc.wantConn)
			}
		})
	}
}
