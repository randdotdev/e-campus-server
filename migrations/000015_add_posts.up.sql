-- Posts (internal communication for university members)
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Scope
    scope_type VARCHAR(20) NOT NULL CHECK (scope_type IN ('university', 'college', 'department', 'program')),
    scope_id UUID,

    -- Nesting (null = top-level post, set = comment/reply)
    parent_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    root_id UUID REFERENCES posts(id) ON DELETE CASCADE,

    -- Content
    body TEXT NOT NULL,

    -- Metadata
    is_pinned BOOLEAN NOT NULL DEFAULT false,
    expires_at TIMESTAMPTZ,

    -- Author
    author_id UUID NOT NULL REFERENCES users(id),

    -- Counters (denormalized)
    like_count INT NOT NULL DEFAULT 0,
    comment_count INT NOT NULL DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT posts_scope_check CHECK (
        (scope_type = 'university' AND scope_id IS NULL) OR
        (scope_type != 'university' AND scope_id IS NOT NULL)
    ),
    CONSTRAINT posts_nesting_check CHECK (
        (parent_id IS NULL AND root_id IS NULL) OR
        (parent_id IS NOT NULL AND root_id IS NOT NULL)
    )
);

-- Post likes
CREATE TABLE post_likes (
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (post_id, user_id)
);

-- Post attachments
CREATE TABLE post_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    stored_file_id UUID NOT NULL REFERENCES stored_files(id),
    display_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(20) NOT NULL CHECK (file_type IN ('image', 'document', 'voice', 'video')),
    order_index INT NOT NULL DEFAULT 0
);

-- Post mentions
CREATE TABLE post_mentions (
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, user_id)
);

-- Indexes
CREATE INDEX idx_posts_scope ON posts(scope_type, scope_id);
CREATE INDEX idx_posts_root ON posts(root_id) WHERE root_id IS NOT NULL;
CREATE INDEX idx_posts_parent ON posts(parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_posts_author ON posts(author_id);
CREATE INDEX idx_posts_created ON posts(created_at DESC);
CREATE INDEX idx_posts_expires ON posts(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_posts_deleted ON posts(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_post_attachments_post ON post_attachments(post_id);
CREATE INDEX idx_post_mentions_user ON post_mentions(user_id);
CREATE INDEX idx_post_likes_user ON post_likes(user_id);
