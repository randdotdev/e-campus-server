-- Enrollment requests for pretake (bypass prerequisite) and retake (retry failed course)
CREATE TABLE enrollment_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(20) NOT NULL CHECK (type IN ('pretake', 'retake')),
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    semester_id UUID NOT NULL REFERENCES semesters(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    rejection_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(student_id, course_id, semester_id, type)
);

CREATE INDEX idx_enrollment_requests_student ON enrollment_requests(student_id);
CREATE INDEX idx_enrollment_requests_semester ON enrollment_requests(semester_id);
CREATE INDEX idx_enrollment_requests_pending ON enrollment_requests(status) WHERE status = 'pending';
