CREATE TABLE news (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    publisher_type VARCHAR(20) NOT NULL CHECK (publisher_type IN ('university', 'college', 'department')),
    publisher_id UUID,

    category VARCHAR(20) NOT NULL CHECK (category IN ('announcement', 'event', 'achievement', 'academic', 'general')),

    title_en VARCHAR(255) NOT NULL,
    title_local VARCHAR(255),
    body_en TEXT NOT NULL,
    body_local TEXT,

    cover_image_id UUID REFERENCES stored_files(id),
    author_id UUID NOT NULL REFERENCES users(id),

    is_pinned BOOLEAN NOT NULL DEFAULT false,
    publish_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT news_publisher_check CHECK (
        (publisher_type = 'university' AND publisher_id IS NULL) OR
        (publisher_type != 'university' AND publisher_id IS NOT NULL)
    )
);

CREATE TABLE news_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    news_id UUID NOT NULL REFERENCES news(id) ON DELETE CASCADE,
    stored_file_id UUID NOT NULL REFERENCES stored_files(id),
    display_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(20) NOT NULL CHECK (file_type IN ('image', 'document', 'video')),
    order_index INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_news_publisher ON news(publisher_type, publisher_id);
CREATE INDEX idx_news_category ON news(category);
CREATE INDEX idx_news_author ON news(author_id);
CREATE INDEX idx_news_publish_at ON news(publish_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_news_created ON news(created_at DESC);
CREATE INDEX idx_news_pinned ON news(is_pinned) WHERE is_pinned = true AND deleted_at IS NULL;
CREATE INDEX idx_news_attachments_news ON news_attachments(news_id);
