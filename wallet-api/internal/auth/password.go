package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// dummyHash kullanıcı bulunamadığında da bcrypt çalıştırıp
// cevap süresinden kullanıcı adı tahmin edilmesini önlemek için.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
