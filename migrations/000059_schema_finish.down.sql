-- Recreate the dropped dead tables as they stood (no code references them).
CREATE TABLE audit_logs (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    actor_id uuid REFERENCES users(id),
    action varchar(100) NOT NULL,
    entity_type varchar(50) NOT NULL,
    entity_id uuid NOT NULL,
    old_values jsonb,
    new_values jsonb,
    ip_address varchar(45),
    user_agent text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_logs_actor ON audit_logs (actor_id);
CREATE INDEX idx_audit_logs_created ON audit_logs (created_at);
CREATE INDEX idx_audit_logs_entity ON audit_logs (entity_type, entity_id);

CREATE TABLE media (
    inode_id uuid PRIMARY KEY REFERENCES inodes(id) ON DELETE CASCADE,
    kind text NOT NULL,
    state text NOT NULL DEFAULT 'pending',
    duration_ms bigint,
    width integer,
    height integer,
    has_thumbnail boolean NOT NULL DEFAULT false,
    has_storyboard boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT media_kind_valid CHECK (kind IN ('video', 'audio', 'image')),
    CONSTRAINT media_state_valid CHECK (state IN ('pending', 'processing', 'ready', 'failed'))
);

CREATE TABLE media_jobs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    inode_id uuid NOT NULL REFERENCES inodes(id) ON DELETE CASCADE,
    state text NOT NULL DEFAULT 'pending',
    attempts integer NOT NULL DEFAULT 0,
    last_error text,
    heartbeat_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT media_jobs_state_valid CHECK (state IN ('pending', 'processing', 'ready', 'failed'))
);
CREATE INDEX idx_media_jobs_pending ON media_jobs (created_at) WHERE state = 'pending';

CREATE TABLE renditions (
    inode_id uuid NOT NULL REFERENCES inodes(id) ON DELETE CASCADE,
    quality text NOT NULL,
    state text NOT NULL DEFAULT 'pending',
    size_bytes bigint,
    PRIMARY KEY (inode_id, quality),
    CONSTRAINT renditions_quality_valid CHECK (quality IN ('1080p', '720p', '480p', '360p', '240p', 'audio')),
    CONSTRAINT renditions_state_valid CHECK (state IN ('pending', 'ready', 'failed'))
);

CREATE TABLE payments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    semester_id uuid NOT NULL REFERENCES semesters(id) ON DELETE RESTRICT,
    amount numeric(10,2) NOT NULL,
    currency varchar(3) NOT NULL DEFAULT 'USD',
    status varchar(20) NOT NULL DEFAULT 'pending',
    due_date date NOT NULL,
    paid_at timestamptz,
    receipt_number varchar(100),
    notes text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT payments_amount_check CHECK (amount >= 0),
    CONSTRAINT payments_check CHECK ((status <> 'paid' AND paid_at IS NULL) OR (status = 'paid' AND paid_at IS NOT NULL) OR status = 'waived'),
    CONSTRAINT payments_status_check CHECK (status IN ('pending', 'paid', 'overdue', 'waived'))
);
CREATE INDEX idx_payments_semester ON payments (semester_id);
CREATE INDEX idx_payments_status ON payments (status);
CREATE INDEX idx_payments_student ON payments (student_id);

ALTER TABLE authz_policies
    DROP CONSTRAINT authz_policies_scope_type_check,
    DROP CONSTRAINT authz_policies_min_level_check,
    DROP CONSTRAINT authz_policies_course_role_check,
    DROP CONSTRAINT authz_policies_domain_check;
ALTER TABLE roles DROP CONSTRAINT roles_domain_check;

DROP INDEX idx_activity_attachments_inode;
DROP INDEX idx_assignment_attachments_inode;
DROP INDEX idx_post_attachments_inode;
DROP INDEX idx_project_attachments_inode;
DROP INDEX idx_project_submission_files_inode;
DROP INDEX idx_qa_answer_attachments_inode;
DROP INDEX idx_qa_question_attachments_inode;
DROP INDEX idx_submission_files_inode;
DROP INDEX idx_activities_cover;
DROP INDEX idx_enrollment_requests_course;
ALTER INDEX idx_lesson_attachments_inode RENAME TO idx_lesson_attachments_stored_file;

ALTER TABLE activities DROP CONSTRAINT activities_cover_image_id_fkey;
ALTER TABLE activities ADD CONSTRAINT activities_cover_image_id_fkey
    FOREIGN KEY (cover_image_id) REFERENCES inodes(id);
ALTER TABLE questions DROP CONSTRAINT questions_image_id_fkey;
ALTER TABLE questions ADD CONSTRAINT questions_image_id_fkey
    FOREIGN KEY (image_id) REFERENCES inodes(id);

ALTER TABLE activity_attachments RENAME COLUMN inode_id TO stored_file_id;
ALTER TABLE activity_attachments DROP CONSTRAINT activity_attachments_inode_id_fkey;
ALTER TABLE activity_attachments ADD CONSTRAINT activity_attachments_stored_file_id_fkey
    FOREIGN KEY (stored_file_id) REFERENCES inodes(id);

ALTER TABLE assignment_attachments RENAME COLUMN inode_id TO stored_file_id;
ALTER TABLE assignment_attachments DROP CONSTRAINT assignment_attachments_inode_id_fkey;
ALTER TABLE assignment_attachments ADD CONSTRAINT assignment_attachments_stored_file_id_fkey
    FOREIGN KEY (stored_file_id) REFERENCES inodes(id);

ALTER TABLE lesson_attachments RENAME COLUMN inode_id TO stored_file_id;
ALTER TABLE lesson_attachments DROP CONSTRAINT lesson_attachments_inode_id_fkey;
ALTER TABLE lesson_attachments ADD CONSTRAINT lesson_attachments_stored_file_id_fkey
    FOREIGN KEY (stored_file_id) REFERENCES inodes(id);

ALTER TABLE post_attachments RENAME COLUMN inode_id TO stored_file_id;
ALTER TABLE post_attachments DROP CONSTRAINT post_attachments_inode_id_fkey;
ALTER TABLE post_attachments ADD CONSTRAINT post_attachments_stored_file_id_fkey
    FOREIGN KEY (stored_file_id) REFERENCES inodes(id);

ALTER TABLE project_attachments RENAME COLUMN inode_id TO stored_file_id;
ALTER TABLE project_attachments DROP CONSTRAINT project_attachments_inode_id_fkey;
ALTER TABLE project_attachments ADD CONSTRAINT project_attachments_stored_file_id_fkey
    FOREIGN KEY (stored_file_id) REFERENCES inodes(id);

ALTER TABLE project_submission_files RENAME COLUMN inode_id TO stored_file_id;
ALTER TABLE project_submission_files DROP CONSTRAINT project_submission_files_inode_id_fkey;
ALTER TABLE project_submission_files ADD CONSTRAINT project_submission_files_stored_file_id_fkey
    FOREIGN KEY (stored_file_id) REFERENCES inodes(id);

ALTER TABLE qa_answer_attachments RENAME COLUMN inode_id TO stored_file_id;
ALTER TABLE qa_answer_attachments DROP CONSTRAINT qa_answer_attachments_inode_id_fkey;
ALTER TABLE qa_answer_attachments ADD CONSTRAINT qa_answer_attachments_stored_file_id_fkey
    FOREIGN KEY (stored_file_id) REFERENCES inodes(id);

ALTER TABLE qa_question_attachments RENAME COLUMN inode_id TO stored_file_id;
ALTER TABLE qa_question_attachments DROP CONSTRAINT qa_question_attachments_inode_id_fkey;
ALTER TABLE qa_question_attachments ADD CONSTRAINT qa_question_attachments_stored_file_id_fkey
    FOREIGN KEY (stored_file_id) REFERENCES inodes(id);

ALTER TABLE submission_files RENAME COLUMN inode_id TO stored_file_id;
ALTER TABLE submission_files DROP CONSTRAINT submission_files_inode_id_fkey;
ALTER TABLE submission_files ADD CONSTRAINT submission_files_stored_file_id_fkey
    FOREIGN KEY (stored_file_id) REFERENCES inodes(id);

ALTER TABLE posts DROP CONSTRAINT posts_scope_type_check;
UPDATE posts SET scope_type = 'course' WHERE scope_type = 'offering';
ALTER TABLE posts ADD CONSTRAINT posts_scope_type_check
    CHECK (scope_type IN ('university', 'college', 'department', 'program', 'course'));
