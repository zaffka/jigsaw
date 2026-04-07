-- Categories
CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        TEXT UNIQUE NOT NULL,
    name        JSONB NOT NULL,
    icon        TEXT NOT NULL,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed categories
INSERT INTO categories (slug, name, icon, sort_order) VALUES
    ('animals',   '{"ru":"Животные","en":"Animals"}',    '🐾', 10),
    ('food',      '{"ru":"Еда","en":"Food"}',             '🍎', 20),
    ('transport', '{"ru":"Транспорт","en":"Transport"}',  '🚗', 30),
    ('people',    '{"ru":"Люди","en":"People"}',          '👨‍👩‍👧', 40),
    ('nature',    '{"ru":"Природа","en":"Nature"}',       '🌿', 50),
    ('toys',      '{"ru":"Игрушки","en":"Toys"}',         '🧸', 60),
    ('clothes',   '{"ru":"Одежда","en":"Clothes"}',       '👕', 70),
    ('home',      '{"ru":"Дом","en":"Home"}',             '🏠', 80),
    ('letters',   '{"ru":"Буквы","en":"Letters"}',        '🔤', 90),
    ('numbers',   '{"ru":"Цифры","en":"Numbers"}',        '🔢', 100),
    ('colors',    '{"ru":"Цвета","en":"Colors"}',         '🎨', 110),
    ('shapes',    '{"ru":"Фигуры","en":"Shapes"}',        '🔷', 120),
    ('emotions',  '{"ru":"Эмоции","en":"Emotions"}',      '😊', 130),
    ('actions',   '{"ru":"Действия","en":"Actions"}',     '🏃', 140);

-- Extend puzzles table
ALTER TABLE puzzles ADD COLUMN category_id UUID REFERENCES categories(id);
ALTER TABLE puzzles ADD COLUMN difficulty  TEXT;
ALTER TABLE puzzles ADD COLUMN visibility  TEXT NOT NULL DEFAULT 'private';
ALTER TABLE puzzles ADD COLUMN owner_type  TEXT NOT NULL DEFAULT 'parent';

-- Puzzle layers (replaces rewards concept for multi-layer rewards)
CREATE TABLE puzzle_layers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    puzzle_id   UUID NOT NULL REFERENCES puzzles(id) ON DELETE CASCADE,
    sort_order  INT NOT NULL DEFAULT 0,
    type        TEXT NOT NULL,
    text        TEXT,
    audio_key   TEXT,
    tts_key     TEXT,
    video_key   TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_puzzle_layers_puzzle_id ON puzzle_layers(puzzle_id);

-- Drop rewards table replaced by puzzle_layers
DROP TABLE IF EXISTS rewards;

-- Child sessions (PIN-based login)
CREATE TABLE child_sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    child_id    UUID NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    token       TEXT UNIQUE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_child_sessions_token ON child_sessions(token);
CREATE INDEX idx_child_sessions_child_id ON child_sessions(child_id);

-- Extend children table
ALTER TABLE children ADD COLUMN pin_hash     TEXT;
ALTER TABLE children ADD COLUMN avatar_emoji TEXT NOT NULL DEFAULT '🧒';

-- Extend catalog_submissions table
ALTER TABLE catalog_submissions ADD COLUMN admin_comment TEXT;
ALTER TABLE catalog_submissions ADD COLUMN notified_at   TIMESTAMPTZ;

-- Seed default admin user (password: changeme, bcrypt cost 10)
INSERT INTO users (email, password_hash, role)
VALUES ('admin@jigsaw.local', '$2a$10$rUj4dQ5FUODVTuViUMeT4.SLImatYONIhg3UnJ5NFI1jGqzsJelJu', 'admin')
ON CONFLICT DO NOTHING;
