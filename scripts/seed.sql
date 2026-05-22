INSERT INTO settings (id, settings) VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a00',
    '{
        "academic": {
            "fx_exam_enabled": true,
            "pass_threshold": 50,
            "attendance_penalty_enabled": false
        },
        "features": {
            "show_members_tab": true,
            "allow_video_download": false,
            "qna_enabled": true,
            "anonymous_questions": true
        },
        "branding": {
            "primary_color": "#1e40af"
        }
    }'
) ON CONFLICT DO NOTHING;

INSERT INTO users (id, email, password_hash, full_name_en, full_name_local, is_active, is_verified) VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01',
    'president@university.edu',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/X4.qJGnV.kPZKuSy.',
    'University President',
    'سەرۆکی زانکۆ',
    true,
    true
) ON CONFLICT (email) DO NOTHING;

INSERT INTO roles (user_id, title_en, title_local, permission, scope_type, assigned_by) VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01',
    'President',
    'سەرۆک',
    'super_admin',
    'university',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01'
) ON CONFLICT DO NOTHING;

INSERT INTO colleges (id, name_en, name_local, code) VALUES (
    'b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01',
    'College of Engineering',
    'کۆلێجی ئەندازیاری',
    'ENG'
) ON CONFLICT DO NOTHING;

INSERT INTO departments (id, college_id, name_en, name_local, code) VALUES (
    'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01',
    'b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01',
    'Computer Science',
    'زانستی کۆمپیوتەر',
    'CS'
) ON CONFLICT DO NOTHING;

INSERT INTO programs (id, department_id, name_en, name_local, code, degree_type, duration_years, total_credits) VALUES (
    'd0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01',
    'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01',
    'Bachelor of Computer Science',
    'بەکالۆریۆسی زانستی کۆمپیوتەر',
    'BCS',
    'bachelor',
    4,
    240
) ON CONFLICT DO NOTHING;

INSERT INTO academic_years (id, year, start_date, end_date, status) VALUES (
    'e0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01',
    2024,
    '2024-09-01',
    '2025-07-31',
    'active'
) ON CONFLICT DO NOTHING;

INSERT INTO semesters (id, academic_year_id, semester, start_date, end_date, status) VALUES (
    'f0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01',
    'e0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01',
    'fall',
    '2024-09-01',
    '2025-01-15',
    'active'
) ON CONFLICT DO NOTHING;
