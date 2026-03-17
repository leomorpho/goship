-- name: get_profile_fully_onboarded_by_user_id_postgres
SELECT fully_onboarded
FROM profiles
WHERE user_profile = $1
LIMIT 1;

-- name: get_profile_fully_onboarded_by_user_id_sqlite
SELECT fully_onboarded
FROM profiles
WHERE user_profile = ?
LIMIT 1;

-- name: get_profile_thumbnail_object_key_by_user_id_postgres
SELECT fs.object_key
FROM profiles p
JOIN images i ON i.id = p.profile_profile_image
JOIN image_sizes sz ON sz.image_sizes = i.id
JOIN file_storages fs ON fs.id = sz.image_size_file
WHERE p.user_profile = $1
  AND lower(sz.size) = 'thumbnail'
ORDER BY sz.id DESC
LIMIT 1;

-- name: get_profile_thumbnail_object_key_by_user_id_sqlite
SELECT fs.object_key
FROM profiles p
JOIN images i ON i.id = p.profile_profile_image
JOIN image_sizes sz ON sz.image_sizes = i.id
JOIN file_storages fs ON fs.id = sz.image_size_file
WHERE p.user_profile = ?
  AND lower(sz.size) = 'thumbnail'
ORDER BY sz.id DESC
LIMIT 1;

-- name: get_profile_settings_by_id_postgres
SELECT id, bio, birthdate, country_code, phone_number_e164, phone_verified, fully_onboarded
FROM profiles
WHERE id = $1
LIMIT 1;

-- name: get_profile_settings_by_id_sqlite
SELECT id, bio, birthdate, country_code, phone_number_e164, phone_verified, fully_onboarded
FROM profiles
WHERE id = ?
LIMIT 1;

-- name: update_profile_bio_by_id_postgres
UPDATE profiles
SET bio = $2
WHERE id = $1;

-- name: update_profile_bio_by_id_sqlite
UPDATE profiles
SET bio = ?
WHERE id = ?;

-- name: update_profile_phone_by_id_postgres
UPDATE profiles
SET country_code = $2, phone_number_e164 = $3
WHERE id = $1;

-- name: update_profile_phone_by_id_sqlite
UPDATE profiles
SET country_code = ?, phone_number_e164 = ?
WHERE id = ?;

-- name: insert_profile_postgres
INSERT INTO profiles (
  created_at,
  updated_at,
  bio,
  birthdate,
  age,
  country_code,
  phone_number_e164,
  user_profile
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id;

-- name: insert_profile_sqlite
INSERT INTO profiles (
  created_at,
  updated_at,
  bio,
  birthdate,
  age,
  country_code,
  phone_number_e164,
  user_profile
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: update_profile_details_by_id_postgres
UPDATE profiles
SET
  bio = $2,
  birthdate = $3,
  age = $4,
  country_code = $5,
  phone_number_e164 = $6
WHERE id = $1;

-- name: update_profile_details_by_id_sqlite
UPDATE profiles
SET
  bio = ?,
  birthdate = ?,
  age = ?,
  country_code = ?,
  phone_number_e164 = ?
WHERE id = ?;

-- name: mark_profile_fully_onboarded_by_id_postgres
UPDATE profiles
SET fully_onboarded = true
WHERE id = $1;

-- name: mark_profile_fully_onboarded_by_id_sqlite
UPDATE profiles
SET fully_onboarded = 1
WHERE id = ?;

-- name: mark_profile_phone_verified_by_id_postgres
UPDATE profiles
SET phone_verified = true
WHERE id = $1;

-- name: mark_profile_phone_verified_by_id_sqlite
UPDATE profiles
SET phone_verified = 1
WHERE id = ?;

-- name: get_friends_by_profile_id_postgres
SELECT
  pf.friend_id,
  u.id AS user_id,
  u.name,
  p.age,
  p.bio,
  p.phone_number_e164,
  p.country_code
FROM profile_friends pf
JOIN profiles p ON p.id = pf.friend_id
JOIN users u ON u.id = p.user_profile
WHERE pf.profile_id = $1
ORDER BY pf.friend_id;

-- name: get_friends_by_profile_id_sqlite
SELECT
  pf.friend_id,
  u.id AS user_id,
  u.name,
  p.age,
  p.bio,
  p.phone_number_e164,
  p.country_code
FROM profile_friends pf
JOIN profiles p ON p.id = pf.friend_id
JOIN users u ON u.id = p.user_profile
WHERE pf.profile_id = ?
ORDER BY pf.friend_id;

-- name: are_profiles_friends_postgres
SELECT EXISTS(
  SELECT 1
  FROM profile_friends
  WHERE profile_id = $1 AND friend_id = $2
);

-- name: are_profiles_friends_sqlite
SELECT EXISTS(
  SELECT 1
  FROM profile_friends
  WHERE profile_id = ? AND friend_id = ?
);

-- name: get_profile_photos_by_profile_id_postgres
SELECT
  i.id AS image_id,
  sz.size,
  sz.width,
  sz.height,
  fs.object_key
FROM images i
JOIN image_sizes sz ON sz.image_sizes = i.id
JOIN file_storages fs ON fs.id = sz.image_size_file
WHERE i.profile_photos = $1
ORDER BY i.created_at DESC, sz.id ASC;

-- name: get_profile_photos_by_profile_id_sqlite
SELECT
  i.id AS image_id,
  sz.size,
  sz.width,
  sz.height,
  fs.object_key
FROM images i
JOIN image_sizes sz ON sz.image_sizes = i.id
JOIN file_storages fs ON fs.id = sz.image_size_file
WHERE i.profile_photos = ?
ORDER BY i.created_at DESC, sz.id ASC;

-- name: get_profile_core_by_id_postgres
SELECT
  p.id,
  u.name,
  p.age,
  p.bio,
  p.phone_number_e164,
  p.country_code
FROM profiles p
JOIN users u ON u.id = p.user_profile
WHERE p.id = $1
LIMIT 1;

-- name: get_profile_core_by_id_sqlite
SELECT
  p.id,
  u.name,
  p.age,
  p.bio,
  p.phone_number_e164,
  p.country_code
FROM profiles p
JOIN users u ON u.id = p.user_profile
WHERE p.id = ?
LIMIT 1;

-- name: get_profile_image_by_profile_id_postgres
SELECT
  i.id AS image_id,
  sz.size,
  sz.width,
  sz.height,
  fs.object_key
FROM profiles p
JOIN images i ON i.id = p.profile_profile_image
JOIN image_sizes sz ON sz.image_sizes = i.id
JOIN file_storages fs ON fs.id = sz.image_size_file
WHERE p.id = $1
ORDER BY sz.id ASC;

-- name: get_profile_image_by_profile_id_sqlite
SELECT
  i.id AS image_id,
  sz.size,
  sz.width,
  sz.height,
  fs.object_key
FROM profiles p
JOIN images i ON i.id = p.profile_profile_image
JOIN image_sizes sz ON sz.image_sizes = i.id
JOIN file_storages fs ON fs.id = sz.image_size_file
WHERE p.id = ?
ORDER BY sz.id ASC;

-- name: link_profiles_as_friends_postgres
INSERT INTO profile_friends (profile_id, friend_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: link_profiles_as_friends_sqlite
INSERT OR IGNORE INTO profile_friends (profile_id, friend_id)
VALUES (?, ?);

-- name: unlink_profiles_as_friends_postgres
DELETE FROM profile_friends
WHERE profile_id = $1 AND friend_id = $2;

-- name: unlink_profiles_as_friends_sqlite
DELETE FROM profile_friends
WHERE profile_id = ? AND friend_id = ?;

-- name: count_unseen_notifications_by_profile_id_postgres
SELECT COUNT(*)
FROM notifications
WHERE profile_notifications = $1 AND read = $2;

-- name: count_unseen_notifications_by_profile_id_sqlite
SELECT COUNT(*)
FROM notifications
WHERE profile_notifications = ? AND read = ?;

-- name: get_profile_image_id_by_profile_id_postgres
SELECT profile_profile_image
FROM profiles
WHERE id = $1
LIMIT 1;

-- name: get_profile_image_id_by_profile_id_sqlite
SELECT profile_profile_image
FROM profiles
WHERE id = ?
LIMIT 1;

-- name: insert_image_postgres
INSERT INTO images (created_at, updated_at, type)
VALUES ($1, $2, $3)
RETURNING id;

-- name: insert_image_sqlite
INSERT INTO images (created_at, updated_at, type)
VALUES (?, ?, ?);

-- name: insert_image_size_postgres
INSERT INTO image_sizes (created_at, updated_at, size, width, height, image_sizes, image_size_file)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: insert_image_size_sqlite
INSERT INTO image_sizes (created_at, updated_at, size, width, height, image_sizes, image_size_file)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: set_profile_image_id_postgres
UPDATE profiles
SET profile_profile_image = $2
WHERE id = $1;

-- name: set_profile_image_id_sqlite
UPDATE profiles
SET profile_profile_image = ?
WHERE id = ?;

-- name: attach_gallery_image_to_profile_postgres
UPDATE images
SET profile_photos = $2
WHERE id = $1;

-- name: attach_gallery_image_to_profile_sqlite
UPDATE images
SET profile_photos = ?
WHERE id = ?;

-- name: image_belongs_to_profile_gallery_postgres
SELECT EXISTS(
  SELECT 1
  FROM images
  WHERE id = $1 AND profile_photos = $2
);

-- name: image_belongs_to_profile_gallery_sqlite
SELECT EXISTS(
  SELECT 1
  FROM images
  WHERE id = ? AND profile_photos = ?
);

-- name: get_image_storage_objects_by_image_id_postgres
SELECT
  i.id AS image_id,
  sz.size,
  sz.width,
  sz.height,
  fs.object_key
FROM images i
JOIN image_sizes sz ON sz.image_sizes = i.id
JOIN file_storages fs ON fs.id = sz.image_size_file
WHERE i.id = $1
ORDER BY sz.id ASC;

-- name: get_image_storage_objects_by_image_id_sqlite
SELECT
  i.id AS image_id,
  sz.size,
  sz.width,
  sz.height,
  fs.object_key
FROM images i
JOIN image_sizes sz ON sz.image_sizes = i.id
JOIN file_storages fs ON fs.id = sz.image_size_file
WHERE i.id = ?
ORDER BY sz.id ASC;

-- name: clear_profile_image_by_image_id_postgres
UPDATE profiles
SET profile_profile_image = NULL
WHERE profile_profile_image = $1;

-- name: clear_profile_image_by_image_id_sqlite
UPDATE profiles
SET profile_profile_image = NULL
WHERE profile_profile_image = ?;

-- name: delete_image_by_id_postgres
DELETE FROM images
WHERE id = $1;

-- name: delete_image_by_id_sqlite
DELETE FROM images
WHERE id = ?;

-- name: get_subscription_for_benefactor_by_profile_id_postgres
SELECT ms.id, ms.paying_profile_id
FROM monthly_subscriptions ms
JOIN monthly_subscription_benefactors msb ON msb.monthly_subscription_id = ms.id
WHERE msb.profile_id = $1
LIMIT 1;

-- name: get_subscription_for_benefactor_by_profile_id_sqlite
SELECT ms.id, ms.paying_profile_id
FROM monthly_subscriptions ms
JOIN monthly_subscription_benefactors msb ON msb.monthly_subscription_id = ms.id
WHERE msb.profile_id = ?
LIMIT 1;

-- name: count_subscription_benefactors_by_subscription_id_postgres
SELECT COUNT(*)
FROM monthly_subscription_benefactors
WHERE monthly_subscription_id = $1;

-- name: count_subscription_benefactors_by_subscription_id_sqlite
SELECT COUNT(*)
FROM monthly_subscription_benefactors
WHERE monthly_subscription_id = ?;

-- name: remove_subscription_benefactor_by_subscription_and_profile_postgres
DELETE FROM monthly_subscription_benefactors
WHERE monthly_subscription_id = $1
  AND profile_id = $2;

-- name: remove_subscription_benefactor_by_subscription_and_profile_sqlite
DELETE FROM monthly_subscription_benefactors
WHERE monthly_subscription_id = ?
  AND profile_id = ?;

-- name: delete_subscription_by_id_postgres
DELETE FROM monthly_subscriptions
WHERE id = $1;

-- name: delete_subscription_by_id_sqlite
DELETE FROM monthly_subscriptions
WHERE id = ?;

-- name: delete_user_by_profile_id_postgres
DELETE FROM users
WHERE id = (
  SELECT user_profile
  FROM profiles
  WHERE id = $1
  LIMIT 1
);

-- name: delete_user_by_profile_id_sqlite
DELETE FROM users
WHERE id = (
  SELECT user_profile
  FROM profiles
  WHERE id = ?
  LIMIT 1
);
