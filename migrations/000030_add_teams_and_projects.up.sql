-- Teams: student-formed, reusable across projects
CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100),
    leader_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE team_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(team_id, student_id)
);

CREATE INDEX idx_teams_leader ON teams(leader_id);
CREATE INDEX idx_teams_status ON teams(status);
CREATE INDEX idx_team_members_team ON team_members(team_id);
CREATE INDEX idx_team_members_student ON team_members(student_id);

CREATE TRIGGER update_teams_updated_at
    BEFORE UPDATE ON teams
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Projects: group-based assignments
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    offering_id UUID NOT NULL REFERENCES course_offerings(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    deadline TIMESTAMPTZ NOT NULL,
    max_score FLOAT NOT NULL CHECK (max_score > 0),
    min_members INT NOT NULL DEFAULT 2 CHECK (min_members >= 1),
    max_members INT NOT NULL DEFAULT 5 CHECK (max_members >= 1),
    merge_target INT CHECK (merge_target >= 1),
    registration_deadline TIMESTAMPTZ,
    visibility VARCHAR(20) NOT NULL DEFAULT 'hidden' CHECK (visibility IN ('hidden', 'registered', 'all')),
    allow_late BOOLEAN NOT NULL DEFAULT false,
    publish_at TIMESTAMPTZ,
    scores_public BOOLEAN NOT NULL DEFAULT false,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_member_range CHECK (min_members <= max_members),
    CONSTRAINT valid_merge_target CHECK (merge_target IS NULL OR (merge_target >= min_members AND merge_target <= max_members))
);

CREATE TABLE project_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    stored_file_id UUID NOT NULL REFERENCES stored_files(id),
    display_name VARCHAR(255) NOT NULL,
    order_index INT NOT NULL DEFAULT 0,
    added_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_projects_offering ON projects(offering_id);
CREATE INDEX idx_project_attachments_project ON project_attachments(project_id);

-- Project registrations: teams declare interest + project title
CREATE TABLE project_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    project_title VARCHAR(255) NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, team_id)
);

CREATE INDEX idx_project_registrations_project ON project_registrations(project_id);
CREATE INDEX idx_project_registrations_team ON project_registrations(team_id);

-- Course groups: merged teams for a specific project
CREATE TABLE course_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(100),
    project_title VARCHAR(255),
    leader_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    finalized BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE course_group_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_group_id UUID NOT NULL REFERENCES course_groups(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_team_id UUID REFERENCES teams(id) ON DELETE SET NULL,
    UNIQUE(course_group_id, student_id)
);

CREATE INDEX idx_course_groups_project ON course_groups(project_id);
CREATE INDEX idx_course_groups_leader ON course_groups(leader_id);
CREATE INDEX idx_course_group_members_group ON course_group_members(course_group_id);
CREATE INDEX idx_course_group_members_student ON course_group_members(student_id);

-- Project submissions: per course group
CREATE TABLE project_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    course_group_id UUID NOT NULL REFERENCES course_groups(id) ON DELETE CASCADE,
    content TEXT,
    submitted_at TIMESTAMPTZ,
    submitted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    UNIQUE(project_id, course_group_id)
);

CREATE TABLE project_submission_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID NOT NULL REFERENCES project_submissions(id) ON DELETE CASCADE,
    stored_file_id UUID NOT NULL REFERENCES stored_files(id),
    display_name VARCHAR(255) NOT NULL,
    order_index INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_project_submissions_project ON project_submissions(project_id);
CREATE INDEX idx_project_submissions_group ON project_submissions(course_group_id);
CREATE INDEX idx_project_submission_files_submission ON project_submission_files(submission_id);

CREATE TRIGGER update_project_submissions_updated_at
    BEFORE UPDATE ON project_submissions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Project grades: individual per member
CREATE TABLE project_grades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID NOT NULL REFERENCES project_submissions(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    score FLOAT,
    feedback TEXT,
    graded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    graded_at TIMESTAMPTZ,
    UNIQUE(submission_id, student_id)
);

CREATE INDEX idx_project_grades_submission ON project_grades(submission_id);
CREATE INDEX idx_project_grades_student ON project_grades(student_id);
