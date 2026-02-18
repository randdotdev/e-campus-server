package user

import "golang.org/x/crypto/bcrypt"

func checkPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func derefInt(p *int, defaultVal int) int {
	if p == nil {
		return defaultVal
	}
	return *p
}
