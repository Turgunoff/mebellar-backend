package sms

import "fmt"

// MockSMSService для тестирования
type MockSMSService struct {
	SendOTPCalled bool
	SendSMSCalled bool
	LastPhone     string
	LastCode      string
	LastMessage   string
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

func (m *MockSMSService) SendSMS(phone, message string) error {
	m.SendSMSCalled = true
	m.LastPhone = phone
	m.LastMessage = message

	if m.ShouldFail {
		return fmt.Errorf("mock SMS error")
	}

	return nil
}
