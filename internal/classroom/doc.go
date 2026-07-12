// Package classroom owns what happens inside one course offering: course
// materials (sections and lessons), assignments, exams and their question
// bank, attendance, final-grade computation, questions & answers, student
// teams, and group projects. An offering is management's noun; this package
// begins where teaching does.
//
// Reading order: content.go, assignment.go, question.go, exam.go,
// attendance.go, grading.go, qa.go, team.go, project.go — each carries one
// noun completely. The *_reader.go files declare what classroom needs from
// its peers; errors.go holds every sentinel.
//
// Two laws inherited from the rest of the system: every attachment is a
// counted inode reference (attach = row + Link, detach = row gone + Unlink,
// through the FileStore port — never a bare UUID column), and authorization
// happens at the HTTP edge — a service method runs only for a caller the
// offering gate already admitted.
package classroom
