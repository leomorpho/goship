-- name: delete_sms_codes_by_profile
DELETE FROM phone_verification_codes
WHERE profile_id = ?;

-- name: insert_sms_code
INSERT INTO phone_verification_codes (
	created_at, updated_at, code, profile_id
) VALUES (?, ?, ?, ?);

-- name: find_latest_valid_sms_code
SELECT id, code
FROM phone_verification_codes
WHERE profile_id = ? AND created_at >= ?
ORDER BY created_at DESC
LIMIT 1;

-- name: delete_sms_code_by_id
DELETE FROM phone_verification_codes
WHERE id = ?;

