package server

import (
	"mebellar-backend/pkg/apperror"
	"mebellar-backend/pkg/validator"
)

// LoginRequestValidation validates login request
type LoginRequestValidation struct {
	Phone    string `validate:"required,phone_uz"`
	Password string `validate:"required,min=6"`
}

func (s *AuthServiceServer) validateLoginRequest(phone, password string) error {
	req := LoginRequestValidation{
		Phone:    phone,
		Password: password,
	}

	if err := validator.Validate(req); err != nil {
		return apperror.NewValidationError(err.Error())
	}
	return nil
}

// RegisterRequestValidation validates register request
type RegisterRequestValidation struct {
	FullName string `validate:"required,min=2,max=100"`
	Phone    string `validate:"required,phone_uz"`
	Password string `validate:"required,strong_password"`
	Role     string `validate:"omitempty,oneof=customer seller"`
}

func (s *AuthServiceServer) validateRegisterRequest(fullName, phone, password, role string) error {
	req := RegisterRequestValidation{
		FullName: fullName,
		Phone:    phone,
		Password: password,
		Role:     role,
	}

	if err := validator.Validate(req); err != nil {
		return apperror.NewValidationError(err.Error())
	}
	return nil
}

// SendOTPRequestValidation validates send OTP request
type SendOTPRequestValidation struct {
	Phone string `validate:"required,phone_uz"`
}

func (s *AuthServiceServer) validateSendOTPRequest(phone string) error {
	req := SendOTPRequestValidation{
		Phone: phone,
	}

	if err := validator.Validate(req); err != nil {
		return apperror.NewValidationError(err.Error())
	}
	return nil
}
