-- Extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'parent' CHECK (role IN ('parent', 'admin')),
    locale      TEXT NOT NULL DEFAULT 'ru',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Sessions
CREATE TABLE sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Children profiles
CREATE TABLE children (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    avatar_key  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_children_user_id ON children(user_id);

-- Puzzles
CREATE TABLE puzzles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    child_id    UUID REFERENCES children(id) ON DELETE SET NULL,
    titles      JSONB NOT NULL DEFAULT '{}',
    image_key   TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'processing' CHECK (status IN ('processing', 'ready', 'failed')),
    config      JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_puzzles_user_id ON puzzles(user_id);
CREATE INDEX idx_puzzles_status ON puzzles(status);

-- Puzzle pieces
CREATE TABLE puzzle_pieces (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    puzzle_id   UUID NOT NULL REFERENCES puzzles(id) ON DELETE CASCADE,
    image_key   TEXT NOT NULL,
    path_svg    TEXT NOT NULL,
    grid_x      INT NOT NULL,
    grid_y      INT NOT NULL,
    bounds      JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_puzzle_pieces_puzzle_id ON puzzle_pieces(puzzle_id);

-- Rewards
CREATE TABLE rewards (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    puzzle_id   UUID NOT NULL UNIQUE REFERENCES puzzles(id) ON DELETE CASCADE,
    type        TEXT NOT NULL CHECK (type IN ('video', 'word', 'animation')),
    content_key TEXT,
    words       JSONB NOT NULL DEFAULT '{}',
    tts_keys    JSONB NOT NULL DEFAULT '{}'
);

-- Catalog submissions (moderation queue)
CREATE TABLE catalog_submissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    puzzle_id   UUID NOT NULL UNIQUE REFERENCES puzzles(id) ON DELETE CASCADE,
    status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    reviewer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reason      TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at TIMESTAMPTZ
);

CREATE INDEX idx_catalog_submissions_status ON catalog_submissions(status);

-- Public catalog
CREATE TABLE catalog_puzzles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    puzzle_id   UUID NOT NULL UNIQUE REFERENCES puzzles(id) ON DELETE CASCADE,
    featured    BOOLEAN NOT NULL DEFAULT false,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Play results
CREATE TABLE play_results (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    child_id    UUID NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    puzzle_id   UUID NOT NULL REFERENCES puzzles(id) ON DELETE CASCADE,
    completed   BOOLEAN NOT NULL DEFAULT false,
    duration_ms INT,
    attempts    INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_play_results_child_id ON play_results(child_id);
CREATE INDEX idx_play_results_puzzle_id ON play_results(puzzle_id);

-- Background task queue
CREATE TABLE tasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    payload     JSONB NOT NULL DEFAULT '{}',
    error       TEXT,
    attempts    INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tasks_status_created ON tasks(status, created_at) WHERE status = 'pending';
