-- Schema finish: attachment FKs say inode_id (000055 renamed the table,
-- this renames its referrer columns), inode referrers declare RESTRICT
-- explicitly, the GC-probed FK columns get indexes, the authz enum columns
-- get CHECKs, dead tables drop, and posts' course scope becomes offering.

-- Classroom posts address a course offering, not a course; the value now
-- says so. Kept in one migration with the CHECK swap so no window exists
-- where rows violate it.
ALTER TABLE posts DROP CONSTRAINT posts_scope_type_check;
UPDATE posts SET scope_type = 'offering' WHERE scope_type = 'course';
ALTER TABLE posts ADD CONSTRAINT posts_scope_type_check
    CHECK (scope_type IN ('university', 'college', 'department', 'program', 'offering'));

-- stored_file_id → inode_id across the nine attachment tables. The FK is
-- re-added RESTRICT: every attachment is a counted link, and the FK is the
-- backstop that vetoes a GC reclaim if the counter ever drifts.
ALTER TABLE activity_attachments RENAME COLUMN stored_file_id TO inode_id;
ALTER TABLE activity_attachments DROP CONSTRAINT activity_attachments_stored_file_id_fkey;
ALTER TABLE activity_attachments ADD CONSTRAINT activity_attachments_inode_id_fkey
    FOREIGN KEY (inode_id) REFERENCES inodes(id) ON DELETE RESTRICT;

ALTER TABLE assignment_attachments RENAME COLUMN stored_file_id TO inode_id;
ALTER TABLE assignment_attachments DROP CONSTRAINT assignment_attachments_stored_file_id_fkey;
ALTER TABLE assignment_attachments ADD CONSTRAINT assignment_attachments_inode_id_fkey
    FOREIGN KEY (inode_id) REFERENCES inodes(id) ON DELETE RESTRICT;

ALTER TABLE lesson_attachments RENAME COLUMN stored_file_id TO inode_id;
ALTER TABLE lesson_attachments DROP CONSTRAINT lesson_attachments_stored_file_id_fkey;
ALTER TABLE lesson_attachments ADD CONSTRAINT lesson_attachments_inode_id_fkey
    FOREIGN KEY (inode_id) REFERENCES inodes(id) ON DELETE RESTRICT;

ALTER TABLE post_attachments RENAME COLUMN stored_file_id TO inode_id;
ALTER TABLE post_attachments DROP CONSTRAINT post_attachments_stored_file_id_fkey;
ALTER TABLE post_attachments ADD CONSTRAINT post_attachments_inode_id_fkey
    FOREIGN KEY (inode_id) REFERENCES inodes(id) ON DELETE RESTRICT;

ALTER TABLE project_attachments RENAME COLUMN stored_file_id TO inode_id;
ALTER TABLE project_attachments DROP CONSTRAINT project_attachments_stored_file_id_fkey;
ALTER TABLE project_attachments ADD CONSTRAINT project_attachments_inode_id_fkey
    FOREIGN KEY (inode_id) REFERENCES inodes(id) ON DELETE RESTRICT;

ALTER TABLE project_submission_files RENAME COLUMN stored_file_id TO inode_id;
ALTER TABLE project_submission_files DROP CONSTRAINT project_submission_files_stored_file_id_fkey;
ALTER TABLE project_submission_files ADD CONSTRAINT project_submission_files_inode_id_fkey
    FOREIGN KEY (inode_id) REFERENCES inodes(id) ON DELETE RESTRICT;

ALTER TABLE qa_answer_attachments RENAME COLUMN stored_file_id TO inode_id;
ALTER TABLE qa_answer_attachments DROP CONSTRAINT qa_answer_attachments_stored_file_id_fkey;
ALTER TABLE qa_answer_attachments ADD CONSTRAINT qa_answer_attachments_inode_id_fkey
    FOREIGN KEY (inode_id) REFERENCES inodes(id) ON DELETE RESTRICT;

ALTER TABLE qa_question_attachments RENAME COLUMN stored_file_id TO inode_id;
ALTER TABLE qa_question_attachments DROP CONSTRAINT qa_question_attachments_stored_file_id_fkey;
ALTER TABLE qa_question_attachments ADD CONSTRAINT qa_question_attachments_inode_id_fkey
    FOREIGN KEY (inode_id) REFERENCES inodes(id) ON DELETE RESTRICT;

ALTER TABLE submission_files RENAME COLUMN stored_file_id TO inode_id;
ALTER TABLE submission_files DROP CONSTRAINT submission_files_stored_file_id_fkey;
ALTER TABLE submission_files ADD CONSTRAINT submission_files_inode_id_fkey
    FOREIGN KEY (inode_id) REFERENCES inodes(id) ON DELETE RESTRICT;

-- The two remaining inode referrers, same declared intent.
ALTER TABLE activities DROP CONSTRAINT activities_cover_image_id_fkey;
ALTER TABLE activities ADD CONSTRAINT activities_cover_image_id_fkey
    FOREIGN KEY (cover_image_id) REFERENCES inodes(id) ON DELETE RESTRICT;
ALTER TABLE questions DROP CONSTRAINT questions_image_id_fkey;
ALTER TABLE questions ADD CONSTRAINT questions_image_id_fkey
    FOREIGN KEY (image_id) REFERENCES inodes(id) ON DELETE RESTRICT;

-- GC's reclaim runs one FK existence probe per referrer table; these keep
-- each probe on an index. The other hot FK columns from the audit are
-- already covered by leading PK/unique columns.
ALTER INDEX idx_lesson_attachments_stored_file RENAME TO idx_lesson_attachments_inode;
CREATE INDEX idx_activity_attachments_inode ON activity_attachments (inode_id);
CREATE INDEX idx_assignment_attachments_inode ON assignment_attachments (inode_id);
CREATE INDEX idx_post_attachments_inode ON post_attachments (inode_id);
CREATE INDEX idx_project_attachments_inode ON project_attachments (inode_id);
CREATE INDEX idx_project_submission_files_inode ON project_submission_files (inode_id);
CREATE INDEX idx_qa_answer_attachments_inode ON qa_answer_attachments (inode_id);
CREATE INDEX idx_qa_question_attachments_inode ON qa_question_attachments (inode_id);
CREATE INDEX idx_submission_files_inode ON submission_files (inode_id);
CREATE INDEX idx_activities_cover ON activities (cover_image_id) WHERE cover_image_id IS NOT NULL;
CREATE INDEX idx_enrollment_requests_course ON enrollment_requests (course_id);

-- The authz enum columns mirror closed Go sets; the schema now agrees.
ALTER TABLE authz_policies
    ADD CONSTRAINT authz_policies_scope_type_check
        CHECK (scope_type IS NULL OR scope_type IN ('program', 'department', 'college', 'university', 'platform')),
    ADD CONSTRAINT authz_policies_min_level_check
        CHECK (min_level IS NULL OR min_level IN ('viewer', 'operator', 'admin', 'super_admin')),
    ADD CONSTRAINT authz_policies_course_role_check
        CHECK (course_role IS NULL OR course_role IN ('teacher', 'assistant', 'student', 'observer')),
    ADD CONSTRAINT authz_policies_domain_check
        CHECK (domain IS NULL OR domain IN ('administration'));
ALTER TABLE roles
    ADD CONSTRAINT roles_domain_check
        CHECK (domain IS NULL OR domain IN ('administration'));

-- Schema for code that does not exist teaches wrong models. audit_logs and
-- the media pipeline predate the current architecture; billing, if ever
-- built, charges universities by subscription tier and student tuition
-- tracking would want a fresh design — payments goes too.
DROP TABLE audit_logs;
DROP TABLE media_jobs;
DROP TABLE renditions;
DROP TABLE media;
DROP TABLE payments;
