package queries

import "testing"

func TestGetAuthQueriesLoaded(t *testing.T) {
	t.Parallel()
	names := []string{
		"get_auth_user_record_by_email_postgres",
		"get_auth_user_record_by_email_sqlite",
		"get_auth_identity_by_user_id_postgres",
		"get_auth_identity_by_user_id_sqlite",
		"get_user_display_name_by_user_id_postgres",
		"get_user_display_name_by_user_id_sqlite",
		"insert_last_seen_online_postgres",
		"insert_last_seen_online_sqlite",
		"update_user_password_hash_by_user_id_postgres",
		"update_user_password_hash_by_user_id_sqlite",
		"update_user_display_name_by_user_id_postgres",
		"update_user_display_name_by_user_id_sqlite",
		"mark_user_verified_by_user_id_postgres",
		"mark_user_verified_by_user_id_sqlite",
		"insert_password_token_postgres",
		"insert_password_token_sqlite",
		"get_password_token_hash_postgres",
		"get_password_token_hash_sqlite",
		"delete_password_tokens_by_user_id_postgres",
		"delete_password_tokens_by_user_id_sqlite",
	}

	for _, name := range names {
		query, err := Get(name)
		if err != nil {
			t.Fatalf("Get(%s): %v", name, err)
		}
		if query == "" {
			t.Fatalf("Get(%s): empty query", name)
		}
	}
}
