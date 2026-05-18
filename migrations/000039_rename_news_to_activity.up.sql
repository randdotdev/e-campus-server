-- Rename news → activities

ALTER TABLE news RENAME TO activities;
ALTER TABLE news_attachments RENAME TO activity_attachments;

ALTER TABLE activity_attachments RENAME COLUMN news_id TO activity_id;

ALTER TABLE activities DROP CONSTRAINT news_publisher_check;
ALTER TABLE activities ADD CONSTRAINT activities_publisher_check CHECK (
    (publisher_type = 'university' AND publisher_id IS NULL) OR
    (publisher_type != 'university' AND publisher_id IS NOT NULL)
);

ALTER TABLE activities RENAME COLUMN category TO type;
ALTER TABLE activities DROP CONSTRAINT news_category_check;
ALTER TABLE activities ADD CONSTRAINT activities_type_check
    CHECK (type IN ('news', 'announcement', 'webinar', 'workshop', 'conference', 'symposium', 'training_course'));

ALTER INDEX idx_news_publisher  RENAME TO idx_activities_publisher;
ALTER INDEX idx_news_category   RENAME TO idx_activities_type;
ALTER INDEX idx_news_author     RENAME TO idx_activities_author;
ALTER INDEX idx_news_publish_at RENAME TO idx_activities_publish_at;
ALTER INDEX idx_news_created    RENAME TO idx_activities_created;
ALTER INDEX idx_news_pinned     RENAME TO idx_activities_pinned;
ALTER INDEX idx_news_attachments_news RENAME TO idx_activity_attachments_activity;

UPDATE authz_policies SET resource = 'activity' WHERE resource = 'news';
