package management

import "github.com/google/uuid"

// ScopeFilter pins a list to the viewer's organisational unit. The gate's
// decision compiles it; handlers pass it through verbatim and repositories
// AND it into the WHERE clause. The zero value narrows nothing.
type ScopeFilter struct {
	ProgramID    *uuid.UUID
	DepartmentID *uuid.UUID
	CollegeID    *uuid.UUID
}
