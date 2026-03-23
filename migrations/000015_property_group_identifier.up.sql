-- Add unique identifier slug to property_groups.
ALTER TABLE property_groups ADD COLUMN identifier VARCHAR(255);

-- Backfill existing rows: slugify from the first translation name.
-- Handles duplicates by appending a numeric suffix.
WITH ranked AS (
    SELECT
        pg.id,
        LOWER(
            REGEXP_REPLACE(
                REGEXP_REPLACE(
                    TRANSLATE(
                        (SELECT pgt.name FROM property_group_translations pgt WHERE pgt.property_group_id = pg.id ORDER BY pgt.locale LIMIT 1),
                        'äöüÄÖÜß',
                        'aouAOUs'
                    ),
                    '[^a-zA-Z0-9]+', '-', 'g'
                ),
                '(^-+|-+$)', '', 'g'
            )
        ) AS base_slug,
        ROW_NUMBER() OVER (
            PARTITION BY LOWER(
                REGEXP_REPLACE(
                    REGEXP_REPLACE(
                        TRANSLATE(
                            (SELECT pgt.name FROM property_group_translations pgt WHERE pgt.property_group_id = pg.id ORDER BY pgt.locale LIMIT 1),
                            'äöüÄÖÜß',
                            'aouAOUs'
                        ),
                        '[^a-zA-Z0-9]+', '-', 'g'
                    ),
                    '(^-+|-+$)', '', 'g'
                )
            )
            ORDER BY pg.created_at
        ) AS rn
    FROM property_groups pg
)
UPDATE property_groups
SET identifier = CASE
    WHEN ranked.rn = 1 THEN ranked.base_slug
    ELSE ranked.base_slug || '-' || ranked.rn
END
FROM ranked
WHERE property_groups.id = ranked.id;

-- Fallback: any rows without a translation get their UUID as identifier.
UPDATE property_groups SET identifier = id::text WHERE identifier IS NULL OR identifier = '';

ALTER TABLE property_groups ALTER COLUMN identifier SET NOT NULL;
CREATE UNIQUE INDEX property_groups_identifier_key ON property_groups (identifier);
