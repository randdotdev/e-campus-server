CREATE TABLE qa_questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    offering_id UUID NOT NULL REFERENCES course_offerings(id) ON DELETE CASCADE,

    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,

    is_anonymous BOOLEAN NOT NULL DEFAULT false,
    is_faq BOOLEAN NOT NULL DEFAULT false,

    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'answered', 'rejected')),

    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    edited_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE qa_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID NOT NULL UNIQUE REFERENCES qa_questions(id) ON DELETE CASCADE,

    body TEXT NOT NULL,

    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ
);

CREATE TABLE qa_question_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID NOT NULL REFERENCES qa_questions(id) ON DELETE CASCADE,
    file_path VARCHAR(500) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size INTEGER NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE qa_answer_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    answer_id UUID NOT NULL REFERENCES qa_answers(id) ON DELETE CASCADE,
    file_path VARCHAR(500) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size INTEGER NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE qa_rejections (
    question_id UUID PRIMARY KEY REFERENCES qa_questions(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    rejected_by UUID NOT NULL REFERENCES users(id),
    rejected_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_qa_questions_offering ON qa_questions(offering_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_qa_questions_status ON qa_questions(offering_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_qa_questions_created_by ON qa_questions(created_by) WHERE deleted_at IS NULL;
CREATE INDEX idx_qa_questions_faq ON qa_questions(offering_id) WHERE is_faq = true AND deleted_at IS NULL;
CREATE INDEX idx_qa_questions_cursor ON qa_questions(created_at DESC, id DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_qa_question_attachments_question ON qa_question_attachments(question_id);
CREATE INDEX idx_qa_answer_attachments_answer ON qa_answer_attachments(answer_id);
