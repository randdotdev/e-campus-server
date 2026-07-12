// Package management is the university administration context. It owns the
// institutional hierarchy (colleges, departments, programs), the academic
// calendar (years, semesters), the course catalogue (courses, offerings,
// teachers, sections), study plans (curriculum, requirements), admissions
// (applications), enrollment (enrollments, groups, pretake/retake requests),
// students and their full history (leaves, cohort moves, transcripts), and
// university-wide settings.
//
// Each noun lives in its own file — entity, read models, pure rules, ports,
// and service, in that order. Adapters live in http/ and postgres/.
package management
