ALTER TABLE posts ADD COLUMN publish_at TIMESTAMPTZ;

CREATE INDEX idx_posts_publish_at ON posts(publish_at) WHERE deleted_at IS NULL;
