-- +goose Up
-- Remove duplicate quality profiles created by repeated Radarr imports.
-- For each group of profiles sharing the same name, keep the one with the
-- smallest id (first-created by UUID insertion order); redirect all FK
-- references before deleting the extras.

-- Step 1: Re-point movies that reference a duplicate to the survivor.
UPDATE movies
SET quality_profile_id = (
    SELECT MIN(qp_inner.id)
    FROM quality_profiles qp_inner
    WHERE qp_inner.name = (
        SELECT qp_outer.name
        FROM quality_profiles qp_outer
        WHERE qp_outer.id = movies.quality_profile_id
    )
)
WHERE quality_profile_id IN (
    SELECT id FROM quality_profiles
    WHERE id NOT IN (SELECT MIN(id) FROM quality_profiles GROUP BY name)
);

-- Step 2: Re-point libraries that reference a duplicate to the survivor.
UPDATE libraries
SET default_quality_profile_id = (
    SELECT MIN(qp_inner.id)
    FROM quality_profiles qp_inner
    WHERE qp_inner.name = (
        SELECT qp_outer.name
        FROM quality_profiles qp_outer
        WHERE qp_outer.id = libraries.default_quality_profile_id
    )
)
WHERE default_quality_profile_id IN (
    SELECT id FROM quality_profiles
    WHERE id NOT IN (SELECT MIN(id) FROM quality_profiles GROUP BY name)
);

-- Step 3: Delete the duplicate rows (those that are not the MIN(id) survivor).
DELETE FROM quality_profiles
WHERE id NOT IN (SELECT MIN(id) FROM quality_profiles GROUP BY name);

-- Step 4: Add a unique index so the same name cannot be inserted twice.
CREATE UNIQUE INDEX IF NOT EXISTS quality_profiles_name_unique ON quality_profiles(name);

-- +goose Down
DROP INDEX IF EXISTS quality_profiles_name_unique;
