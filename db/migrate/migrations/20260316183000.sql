ALTER TABLE profiles ADD COLUMN preferred_language TEXT;

ALTER TABLE profiles DROP COLUMN preferred_language;
