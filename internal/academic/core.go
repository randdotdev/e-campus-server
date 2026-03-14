package academic

func IsValidAcademicYearStatus(status string) bool {
	switch status {
	case AcademicYearStatusUpcoming, AcademicYearStatusActive, AcademicYearStatusFinalized, AcademicYearStatusArchived:
		return true
	}
	return false
}

func IsValidSemesterStatus(status string) bool {
	switch status {
	case SemesterStatusUpcoming, SemesterStatusActive, SemesterStatusGrading, SemesterStatusFinalized, SemesterStatusArchived:
		return true
	}
	return false
}

func IsValidSemesterType(semester string) bool {
	switch semester {
	case SemesterTypeFall, SemesterTypeSpring, SemesterTypeSummer, SemesterTypeAnnual:
		return true
	}
	return false
}

func IsValidSemesterTransition(from, to string) bool {
	transitions := map[string][]string{
		SemesterStatusUpcoming:  {SemesterStatusActive},
		SemesterStatusActive:    {SemesterStatusGrading},
		SemesterStatusGrading:   {SemesterStatusFinalized},
		SemesterStatusFinalized: {SemesterStatusArchived, SemesterStatusGrading},
		SemesterStatusArchived:  {},
	}

	allowed, ok := transitions[from]
	if !ok {
		return false
	}

	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}
