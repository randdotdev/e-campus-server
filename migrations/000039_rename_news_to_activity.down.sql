UPDATE authz_policies SET resource = 'news' WHERE resource = 'activity';

ALTER INDEX idx_activity_attachments_activity RENAME TO idx_news_attachments_news;
ALTER INDEX idx_activities_pinned     RENAME TO idx_news_pinned;
ALTER INDEX idx_activities_created    RENAME TO idx_news_created;
ALTER INDEX idx_activities_publish_at RENAME TO idx_news_publish_at;
ALTER INDEX idx_activities_author     RENAME TO idx_news_author;
ALTER INDEX idx_activities_type       RENAME TO idx_news_category;
ALTER INDEX idx_activities_publisher  RENAME TO idx_news_publisher;

ALTER TABLE activities DROP CONSTRAINT activities_type_check;
ALTER TABLE activities RENAME COLUMN type TO category;
ALTER TABLE activities ADD CONSTRAINT news_category_check
    CHECK (category IN ('announcement', 'event', 'achievement', 'academic', 'general'));

ALTER TABLE activities DROP CONSTRAINT activities_publisher_check;
ALTER TABLE activities ADD CONSTRAINT news_publisher_check CHECK (
    (publisher_type = 'university' AND publisher_id IS NULL) OR
    (publisher_type != 'university' AND publisher_id IS NOT NULL)
);

ALTER TABLE activity_attachments RENAME COLUMN activity_id TO news_id;

ALTER TABLE activity_attachments RENAME TO news_attachments;
ALTER TABLE activities RENAME TO news;
