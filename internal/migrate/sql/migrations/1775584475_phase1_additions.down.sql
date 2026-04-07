-- Remove seeded admin user
DELETE FROM users WHERE email = 'admin@jigsaw.local' AND role = 'admin';

-- Revert catalog_submissions columns
ALTER TABLE catalog_submissions DROP COLUMN IF EXISTS notified_at;
ALTER TABLE catalog_submissions DROP COLUMN IF EXISTS admin_comment;

-- Revert children columns
ALTER TABLE children DROP COLUMN IF EXISTS avatar_emoji;
ALTER TABLE children DROP COLUMN IF EXISTS pin_hash;

-- Drop child_sessions
DROP TABLE IF EXISTS child_sessions;

-- Drop puzzle_layers
DROP TABLE IF EXISTS puzzle_layers;

-- Revert puzzles columns
ALTER TABLE puzzles DROP COLUMN IF EXISTS owner_type;
ALTER TABLE puzzles DROP COLUMN IF EXISTS visibility;
ALTER TABLE puzzles DROP COLUMN IF EXISTS difficulty;
ALTER TABLE puzzles DROP COLUMN IF EXISTS category_id;

-- Drop categories (seed data removed with table)
DROP TABLE IF EXISTS categories;
