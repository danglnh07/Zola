package security

import "golang.org/x/crypto/bcrypt"

// Methods to hash passwords using bcrypt
func BcryptHash(str string) (string, error) {
	// Use bcrypt to hash the password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// Method to compare a bcrypt hashed password with a plain text password
func BcryptCompare(hashedStr, plainStr string) bool {
	// Compare the hashed password with the plain text password
	err := bcrypt.CompareHashAndPassword([]byte(hashedStr), []byte(plainStr))
	return err == nil
}
