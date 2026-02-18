package university

// Code validation

func IsValidCode(code string) bool {
	if len(code) < 2 || len(code) > 20 {
		return false
	}
	for _, r := range code {
		if !isAlphanumOrUnderscore(r) {
			return false
		}
	}
	return true
}

func isAlphanumOrUnderscore(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_'
}

// Degree type validation

func IsValidDegreeType(degreeType string) bool {
	switch degreeType {
	case "bachelor", "masters", "phd", "diploma", "certificate":
		return true
	default:
		return false
	}
}
