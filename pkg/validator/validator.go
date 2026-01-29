package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// Регистрация кастомных валидаторов
	validate.RegisterValidation("phone_uz", validateUzbekPhone)
	validate.RegisterValidation("strong_password", validateStrongPassword)
}

// Validate валидирует структуру
func Validate(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		return formatValidationError(err)
	}
	return nil
}

// validateUzbekPhone проверяет формат узбекского телефона
func validateUzbekPhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	matched, _ := regexp.MatchString(`^\+998[0-9]{9}$`, phone)
	return matched
}

// validateStrongPassword проверяет надежность пароля
func validateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// Минимум 8 символов
	if len(password) < 8 {
		return false
	}

	// Должна быть хотя бы одна цифра и одна буква
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)

	return hasDigit && hasLetter
}

// formatValidationError форматирует ошибки валидации для пользователя
func formatValidationError(err error) error {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, e := range validationErrors {
			messages = append(messages, formatFieldError(e))
		}
		return fmt.Errorf("%s", strings.Join(messages, "; "))
	}
	return err
}

// formatFieldError форматирует ошибку конкретного поля
func formatFieldError(e validator.FieldError) string {
	field := e.Field()

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s обязательно для заполнения", field)
	case "email":
		return fmt.Sprintf("%s должен быть валидным email адресом", field)
	case "min":
		return fmt.Sprintf("%s должен быть минимум %s символов", field, e.Param())
	case "max":
		return fmt.Sprintf("%s должен быть максимум %s символов", field, e.Param())
	case "phone_uz":
		return fmt.Sprintf("%s должен быть в формате +998XXXXXXXXX", field)
	case "strong_password":
		return fmt.Sprintf("%s должен содержать минимум 8 символов, включая буквы и цифры", field)
	default:
		return fmt.Sprintf("%s не прошел валидацию: %s", field, e.Tag())
	}
}
