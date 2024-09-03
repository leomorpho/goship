-- Modify "profiles" table
ALTER TABLE "profiles" DROP COLUMN "min_interested_age", DROP COLUMN "max_interested_age", DROP COLUMN "latitude", DROP COLUMN "longitude", DROP COLUMN "radius", DROP COLUMN "num_matches";
-- Drop "profile_disliked_profiles" table
DROP TABLE "profile_disliked_profiles";
-- Drop "profile_liked_profiles" table
DROP TABLE "profile_liked_profiles";
-- Drop "profile_matches" table
DROP TABLE "profile_matches";
