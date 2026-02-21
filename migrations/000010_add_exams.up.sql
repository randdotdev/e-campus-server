-- Questions (question bank per course_code)
CREATE TABLE questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_code VARCHAR(50) NOT NULL,

    text TEXT NOT NULL,
    image_url VARCHAR(500),

    type VARCHAR(20) NOT NULL CHECK (type IN ('single', 'multiple', 'true_false', 'short_answer')),
    options JSONB,
    correct JSONB,
    default_score FLOAT NOT NULL DEFAULT 1 CHECK (default_score > 0),
    difficulty VARCHAR(20) CHECK (difficulty IN ('easy', 'medium', 'hard')),

    is_active BOOLEAN NOT NULL DEFAULT true,

    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_questions_course ON questions(course_code) WHERE is_active = true;
CREATE INDEX idx_questions_difficulty ON questions(course_code, difficulty) WHERE is_active = true;
CREATE INDEX idx_questions_created_by ON questions(created_by);

-- Exams
CREATE TABLE exams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    offering_id UUID NOT NULL REFERENCES course_offerings(id) ON DELETE CASCADE,
    section_id UUID REFERENCES sections(id) ON DELETE SET NULL,

    title VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(20) NOT NULL CHECK (type IN ('exam', 'quiz')),
    mode VARCHAR(20) NOT NULL DEFAULT 'online' CHECK (mode IN ('online', 'manual')),

    questions JSONB NOT NULL DEFAULT '[]',
    total_score FLOAT NOT NULL DEFAULT 0 CHECK (total_score >= 0),

    duration_minutes INT CHECK (duration_minutes > 0),
    shuffle_questions BOOLEAN NOT NULL DEFAULT false,
    shuffle_options BOOLEAN NOT NULL DEFAULT false,
    show_results VARCHAR(20) NOT NULL DEFAULT 'after_submit'
        CHECK (show_results IN ('immediately', 'after_submit', 'after_deadline', 'manual')),
    max_attempts INT NOT NULL DEFAULT 1 CHECK (max_attempts >= 1),

    available_from TIMESTAMPTZ,
    available_until TIMESTAMPTZ,
    used_at TIMESTAMPTZ,

    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'closed')),
    published_at TIMESTAMPTZ,

    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (available_until IS NULL OR available_from IS NULL OR available_until > available_from)
);

CREATE INDEX idx_exams_offering ON exams(offering_id);
CREATE INDEX idx_exams_section ON exams(section_id) WHERE section_id IS NOT NULL;
CREATE INDEX idx_exams_status ON exams(status);

-- Exam Attempts
CREATE TABLE exam_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,

    answers JSONB,
    scores JSONB,
    total_score FLOAT,

    started_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    submitted_at TIMESTAMPTZ,

    late_accepted BOOLEAN,

    graded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    graded_at TIMESTAMPTZ,

    visibility VARCHAR(20) NOT NULL DEFAULT 'private'
        CHECK (visibility IN ('private', 'public', 'scheduled')),
    visible_at TIMESTAMPTZ
);

CREATE INDEX idx_attempts_exam ON exam_attempts(exam_id);
CREATE INDEX idx_attempts_student ON exam_attempts(student_id);
CREATE INDEX idx_attempts_submitted ON exam_attempts(submitted_at) WHERE submitted_at IS NOT NULL;

-- Triggers
CREATE TRIGGER update_questions_updated_at BEFORE UPDATE ON questions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
