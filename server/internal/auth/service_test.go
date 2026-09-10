package auth

import "testing"

func TestValidMainlandPhone(t *testing.T) {
	for _, phone := range []string{"+8613812345678", "+8619912345678"} {
		if !validMainlandPhone(phone) {
			t.Fatalf("valid phone rejected: %s", phone)
		}
	}
	for _, phone := range []string{"13812345678", "+8612812345678", "+85261234567", "+861381234567"} {
		if validMainlandPhone(phone) {
			t.Fatalf("invalid phone accepted: %s", phone)
		}
	}
}
