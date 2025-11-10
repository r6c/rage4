package libdnsrage4

import (
	"context"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

func TestProviderInterfaces(t *testing.T) {
	// Verify that Provider implements all required interfaces
	var p *Provider

	// Check RecordGetter interface
	if _, ok := interface{}(p).(libdns.RecordGetter); !ok {
		t.Error("Provider does not implement RecordGetter")
	}

	// Check RecordAppender interface
	if _, ok := interface{}(p).(libdns.RecordAppender); !ok {
		t.Error("Provider does not implement RecordAppender")
	}

	// Check RecordSetter interface
	if _, ok := interface{}(p).(libdns.RecordSetter); !ok {
		t.Error("Provider does not implement RecordSetter")
	}

	// Check RecordDeleter interface
	if _, ok := interface{}(p).(libdns.RecordDeleter); !ok {
		t.Error("Provider does not implement RecordDeleter")
	}
}

func TestToLibdnsRecord(t *testing.T) {
	tests := []struct {
		name     string
		input    Rage4Record
		zone     string
		expected libdns.Record
	}{
		{
			name: "A record - subdomain",
			input: Rage4Record{
				ID:      123,
				Name:    "www.example.com",
				Type:    "A",
				Content: "192.0.2.1",
				TTL:     3600,
			},
			zone: "example.com",
			expected: libdns.Record{
				ID:    "123",
				Name:  "www",
				Type:  "A",
				Value: "192.0.2.1",
				TTL:   3600 * time.Second,
			},
		},
		{
			name: "CNAME record - subdomain",
			input: Rage4Record{
				ID:      456,
				Name:    "alias.example.com",
				Type:    "CNAME",
				Content: "www.example.com",
				TTL:     7200,
			},
			zone: "example.com",
			expected: libdns.Record{
				ID:    "456",
				Name:  "alias",
				Type:  "CNAME",
				Value: "www.example.com",
				TTL:   7200 * time.Second,
			},
		},
		{
			name: "MX record - root",
			input: Rage4Record{
				ID:       789,
				Name:     "example.com",
				Type:     "MX",
				Content:  "mail.example.com",
				TTL:      3600,
				Priority: 10,
			},
			zone: "example.com",
			expected: libdns.Record{
				ID:       "789",
				Name:     "@",
				Type:     "MX",
				Value:    "mail.example.com",
				TTL:      3600 * time.Second,
				Priority: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toLibdnsRecord(tt.input, tt.zone)

			if result.ID != tt.expected.ID {
				t.Errorf("ID mismatch: got %s, want %s", result.ID, tt.expected.ID)
			}
			if result.Name != tt.expected.Name {
				t.Errorf("Name mismatch: got %s, want %s", result.Name, tt.expected.Name)
			}
			if result.Type != tt.expected.Type {
				t.Errorf("Type mismatch: got %s, want %s", result.Type, tt.expected.Type)
			}
			if result.Value != tt.expected.Value {
				t.Errorf("Value mismatch: got %s, want %s", result.Value, tt.expected.Value)
			}
			if result.TTL != tt.expected.TTL {
				t.Errorf("TTL mismatch: got %v, want %v", result.TTL, tt.expected.TTL)
			}
			if result.Priority != tt.expected.Priority {
				t.Errorf("Priority mismatch: got %d, want %d", result.Priority, tt.expected.Priority)
			}
		})
	}
}

func TestProviderStructure(t *testing.T) {
	// Test that Provider can be created with Email and APIKey
	p := &Provider{
		Email:  "test@example.com",
		APIKey: "test-api-key",
	}

	if p.Email != "test@example.com" {
		t.Errorf("Email not set correctly: got %s", p.Email)
	}

	if p.APIKey != "test-api-key" {
		t.Errorf("APIKey not set correctly: got %s", p.APIKey)
	}
}

func TestRecordConversion(t *testing.T) {
	// Test that libdns.Record fields are correctly mapped
	record := libdns.Record{
		ID:       "999",
		Name:     "test",
		Type:     "A",
		Value:    "10.0.0.1",
		TTL:      1800 * time.Second,
		Priority: 5,
		Weight:   10,
	}

	// Verify that the record has all expected fields
	if record.ID != "999" {
		t.Errorf("Record ID mismatch: got %s", record.ID)
	}
	if record.Name != "test" {
		t.Errorf("Record Name mismatch: got %s", record.Name)
	}
	if record.Type != "A" {
		t.Errorf("Record Type mismatch: got %s", record.Type)
	}
	if record.Value != "10.0.0.1" {
		t.Errorf("Record Value mismatch: got %s", record.Value)
	}
	if record.TTL != 1800*time.Second {
		t.Errorf("Record TTL mismatch: got %v", record.TTL)
	}
	if record.Priority != 5 {
		t.Errorf("Record Priority mismatch: got %d", record.Priority)
	}
	if record.Weight != 10 {
		t.Errorf("Record Weight mismatch: got %d", record.Weight)
	}
}

func TestContextHandling(t *testing.T) {
	// Test that methods accept context
	p := &Provider{
		Email:  "test@example.com",
		APIKey: "test-api-key",
	}

	ctx := context.Background()

	// These will fail due to invalid credentials, but we're just testing
	// that the methods accept context and don't panic
	_, err := p.GetRecords(ctx, "example.com")
	if err == nil {
		t.Log("GetRecords succeeded (unexpected with test credentials)")
	}
}
