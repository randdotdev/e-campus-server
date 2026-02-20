-- Courses (templates)
CREATE TABLE courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    department_id UUID NOT NULL REFERENCES departments(id),

    code VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    subtitle VARCHAR(100),
    group_order INT NOT NULL DEFAULT 1,

    requires UUID REFERENCES courses(id),

    ects INT NOT NULL CHECK (ects > 0),
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(department_id, code, group_order)
);

CREATE INDEX idx_courses_department ON courses(department_id);
CREATE INDEX idx_courses_code ON courses(department_id, code);

-- Course Offerings (instances per cohort/semester)
CREATE TABLE course_offerings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id),
    semester_id UUID NOT NULL REFERENCES semesters(id),

    cohort_year INT NOT NULL CHECK (cohort_year >= 2000 AND cohort_year <= 2100),
    shift VARCHAR(10) NOT NULL CHECK (shift IN ('day', 'evening')),

    is_active BOOLEAN NOT NULL DEFAULT true,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(course_id, semester_id, shift)
);

CREATE INDEX idx_offerings_course ON course_offerings(course_id);
CREATE INDEX idx_offerings_semester ON course_offerings(semester_id);
CREATE INDEX idx_offerings_cohort ON course_offerings(cohort_year, shift);

-- Course Teachers
CREATE TABLE course_teachers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    offering_id UUID NOT NULL REFERENCES course_offerings(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),

    role VARCHAR(20) NOT NULL CHECK (role IN ('teacher', 'assistant')),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(offering_id, user_id)
);

CREATE INDEX idx_course_teachers_offering ON course_teachers(offering_id);
CREATE INDEX idx_course_teachers_user ON course_teachers(user_id);

-- Course Enrollments
CREATE TABLE course_enrollments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    offering_id UUID NOT NULL REFERENCES course_offerings(id),
    student_id UUID NOT NULL REFERENCES users(id),

    enrollment_type VARCHAR(20) NOT NULL DEFAULT 'curriculum'
        CHECK (enrollment_type IN ('curriculum', 'retake', 'pretake', 'extra')),
    status VARCHAR(20) NOT NULL DEFAULT 'enrolled'
        CHECK (status IN ('enrolled', 'dropped', 'completed', 'failed')),

    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,

    final_grade FLOAT,

    UNIQUE(offering_id, student_id)
);

CREATE INDEX idx_enrollments_offering ON course_enrollments(offering_id);
CREATE INDEX idx_enrollments_student ON course_enrollments(student_id);
CREATE INDEX idx_enrollments_type ON course_enrollments(enrollment_type);

-- Sections (weeks/containers)
CREATE TABLE sections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    offering_id UUID NOT NULL REFERENCES course_offerings(id) ON DELETE CASCADE,

    title VARCHAR(100) NOT NULL,
    order_index INT NOT NULL,

    unlock_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(offering_id, order_index)
);

CREATE INDEX idx_sections_offering ON sections(offering_id);

-- Lessons
CREATE TABLE lessons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    section_id UUID NOT NULL REFERENCES sections(id) ON DELETE CASCADE,
    offering_id UUID NOT NULL REFERENCES course_offerings(id),

    title VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(20) NOT NULL CHECK (type IN ('theory', 'practice', 'other')),

    scheduled_at TIMESTAMPTZ,
    duration_hours FLOAT,
    room VARCHAR(50),

    publish_at TIMESTAMPTZ,

    order_index INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(section_id, order_index)
);

CREATE INDEX idx_lessons_section ON lessons(section_id);
CREATE INDEX idx_lessons_offering ON lessons(offering_id);
CREATE INDEX idx_lessons_scheduled ON lessons(scheduled_at);

-- Triggers
CREATE TRIGGER update_courses_updated_at BEFORE UPDATE ON courses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
