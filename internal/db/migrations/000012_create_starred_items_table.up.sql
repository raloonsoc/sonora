CREATE TABLE starred_items (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_type  TEXT NOT NULL CHECK (item_type IN ('track', 'album', 'artist')),
    item_id    UUID NOT NULL,
    starred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, item_type, item_id)
);

CREATE INDEX idx_starred_items_user_type ON starred_items(user_id, item_type);
