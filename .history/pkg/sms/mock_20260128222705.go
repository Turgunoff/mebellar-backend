package sms

import "fmt"

// MockSMSService для тестирования
type MockSMSService struct {
	SendOTPCalled bool
	LastPhone     string
	LastCode      string
	ShouldFail    bool
}

func (m *MockSMSService) SendOTP(phone, code string) error {
	m.SendOTPCalled = true
	m.LastPhone = phone
	m.LastCode = code

	if m.ShouldFail {
		return fmt.Errorf("mock SMS error")
	}

	return nil
}
