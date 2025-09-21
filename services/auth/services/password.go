package services

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const defaultCost = 4

type PasswordService struct {
	cost int
}

func NewPasswordService() *PasswordService {
	cost := defaultCost

	if cost < 4 {
		cost = 4
	}

	if cost > 31 {
		cost = 31
	}

	return &PasswordService{
		cost: cost,
	}
}

func (ps *PasswordService) HashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), ps.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

func (ps *PasswordService) VerifyPassword(hashedPassword, password string) error {
	if hashedPassword == "" || password == "" {
		return fmt.Errorf("password and hash cannot be empty")
	}

	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return fmt.Errorf("invalid password")
	}

	return nil
}

func (ps *PasswordService) IsPasswordValid(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	if len(password) > 128 {
		return fmt.Errorf("password must be no more than 128 characters long")
	}

	return nil
}

func (ps *PasswordService) ValidateAndHashPassword(password string) (string, error) {
	if err := ps.IsPasswordValid(password); err != nil {
		return "", err
	}

	return ps.HashPassword(password)
}
