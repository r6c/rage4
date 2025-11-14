// with the libdns interfaces for Rage4 DNS service.
package libdnsrage4

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

const baseURL = "https://rage4.com/rapi"

// Provider facilitates DNS record manipulation with Rage4.
type Provider struct {
	// Email is the account email for Rage4 API authentication
	Email string `json:"email,omitempty"`

	// APIKey is the API key for Rage4 API authentication
	APIKey string `json:"api_key,omitempty"`
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	domainID, err := p.getDomainID(ctx, zone)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain ID: %w", err)
	}

	// Remove trailing dot from zone for name conversion
	zoneName := strings.TrimSuffix(zone, ".")

	url := fmt.Sprintf("%s/GetRecords?id=%d", baseURL, domainID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(p.Email, p.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("received non-200 response: %d %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result []Rage4Record
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var records []libdns.Record
	for _, record := range result {
		records = append(records, toLibdnsRecord(record, zoneName))
	}
	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	domainID, err := p.getDomainID(ctx, zone)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain ID: %w", err)
	}

	// Remove trailing dot from zone for name construction
	zoneName := strings.TrimSuffix(zone, ".")

	var appendedRecords []libdns.Record
	for _, record := range records {
		ttl := int(record.TTL.Seconds())
		if ttl == 0 {
			ttl = 3600
		}

		// Construct the full record name (FQDN)
		var fullName string
		if record.Name == "" || record.Name == "@" {
			fullName = zoneName
		} else {
			fullName = record.Name + "." + zoneName
		}

		url := fmt.Sprintf("%s/CreateRecord?id=%d&name=%s&content=%s&type=%s&ttl=%d",
			baseURL, domainID, fullName, record.Value, record.Type, ttl)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.SetBasicAuth(p.Email, p.APIKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("failed to create record: %d %s", resp.StatusCode, string(body))
		}

		body, _ := io.ReadAll(resp.Body)
		var result CommonResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if !result.Status {
			return nil, fmt.Errorf("API returned error: %s", result.Error)
		}

		appendedRecords = append(appendedRecords, record)
	}

	return appendedRecords, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	existingRecords, err := p.GetRecords(ctx, zone)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing records: %w", err)
	}

	// Find records to delete (existing records with same name and type as new records)
	var toDelete []libdns.Record
	for _, existing := range existingRecords {
		for _, newRecord := range records {
			if existing.Name == newRecord.Name && existing.Type == newRecord.Type {
				toDelete = append(toDelete, existing)
				break
			}
		}
	}

	// Delete old records
	if len(toDelete) > 0 {
		_, err := p.DeleteRecords(ctx, zone, toDelete)
		if err != nil {
			return nil, fmt.Errorf("failed to delete old records: %w", err)
		}
	}

	// Append new records
	appendedRecords, err := p.AppendRecords(ctx, zone, records)
	if err != nil {
		return nil, fmt.Errorf("failed to append new records: %w", err)
	}

	return appendedRecords, nil
}

// DeleteRecords deletes the specified records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	domainID, err := p.getDomainID(ctx, zone)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain ID: %w", err)
	}

	var deletedRecords []libdns.Record
	for _, record := range records {
		// If record has an ID, use it directly; otherwise, find it by name/type/value
		recordID := 0
		if record.ID != "" {
			// Try to parse the ID if it's provided
			if id, err := strconv.Atoi(record.ID); err == nil {
				recordID = id
			}
		}

		// If no ID, find it by matching name, type, and value
		if recordID == 0 {
			var err error
			recordID, err = p.getRecordID(ctx, domainID, record)
			if err != nil {
				return nil, fmt.Errorf("failed to get record ID: %w", err)
			}
		}

		url := fmt.Sprintf("%s/DeleteRecord?id=%d", baseURL, recordID)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.SetBasicAuth(p.Email, p.APIKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("failed to delete record: %d %s", resp.StatusCode, string(body))
		}

		body, _ := io.ReadAll(resp.Body)
		var result CommonResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if !result.Status {
			return nil, fmt.Errorf("API returned error: %s", result.Error)
		}

		deletedRecords = append(deletedRecords, record)
	}

	return deletedRecords, nil
}

// Rage4Record represents a DNS record from Rage4 API
type Rage4Record struct {
	ID               int      `json:"id"`
	DomainID         int      `json:"domain_id"`
	Name             string   `json:"name"`
	Content          string   `json:"content"`
	Type             string   `json:"type"`
	TTL              int      `json:"ttl"`
	Priority         int      `json:"priority"`
	IsActive         bool     `json:"is_active"`
	FailoverEnabled  bool     `json:"failover_enabled"`
	FailoverContent  *string  `json:"failover_content"`
	FailoverWithdraw bool     `json:"failover_withdraw"`
	FailoverActive   bool     `json:"failover_active"`
	GeoRegionID      int      `json:"geo_region_id"`
	GeoLat           *float64 `json:"geo_lat"`
	GeoLong          *float64 `json:"geo_long"`
	GeoAsNum         *int64   `json:"geo_asnum"`
	UDPLimit         bool     `json:"udp_limit"`
	Description      *string  `json:"description"`
	WebhookID        *int     `json:"webhook_id"`
	IsSystem         bool     `json:"is_system"`
	Weight           int      `json:"weight"`
}

// CommonResponse represents a common API response from Rage4
type CommonResponse struct {
	Status bool   `json:"status"`
	ID     int    `json:"id"`
	Error  string `json:"error"`
}

// DomainResponse represents a domain from Rage4 API
type DomainResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"owner_email"`
}

// getDomainID retrieves the domain ID from Rage4 API
func (p *Provider) getDomainID(ctx context.Context, zone string) (int, error) {
	// Remove trailing dot if present
	zone = strings.TrimSuffix(zone, ".")

	url := fmt.Sprintf("%s/GetDomains", baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(p.Email, p.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("received non-200 response: %d %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	var domains []DomainResponse
	if err := json.Unmarshal(body, &domains); err != nil {
		return 0, fmt.Errorf("failed to parse JSON: %w", err)
	}

	for _, domain := range domains {
		if domain.Name == zone {
			return domain.ID, nil
		}
	}

	return 0, fmt.Errorf("domain not found: %s", zone)
}

// getRecordID retrieves the record ID by matching name, type, and value
func (p *Provider) getRecordID(ctx context.Context, domainID int, record libdns.Record) (int, error) {
	url := fmt.Sprintf("%s/GetRecords?id=%d", baseURL, domainID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(p.Email, p.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("received non-200 response: %d %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	var records []Rage4Record
	if err := json.Unmarshal(body, &records); err != nil {
		return 0, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// We need to get the zone name to convert Rage4's full names to relative names
	// Get domain info to retrieve the zone name
	domainURL := fmt.Sprintf("%s/GetDomain?id=%d", baseURL, domainID)
	domainReq, err := http.NewRequestWithContext(ctx, "GET", domainURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create domain request: %w", err)
	}
	domainReq.SetBasicAuth(p.Email, p.APIKey)
	domainResp, err := http.DefaultClient.Do(domainReq)
	if err != nil {
		return 0, fmt.Errorf("failed to get domain info: %w", err)
	}
	defer domainResp.Body.Close()

	var domain DomainResponse
	if err := json.NewDecoder(domainResp.Body).Decode(&domain); err != nil {
		return 0, fmt.Errorf("failed to parse domain info: %w", err)
	}
	zoneName := domain.Name

	for _, r := range records {
		// Convert Rage4's full name to relative name for comparison
		relativeName := r.Name
		if strings.HasSuffix(r.Name, "."+zoneName) {
			relativeName = strings.TrimSuffix(r.Name, "."+zoneName)
		} else if r.Name == zoneName {
			relativeName = "@"
		}

		// For TXT records, compare values with and without quotes
		// since Rage4 API adds quotes to TXT record values
		valueMatches := false
		if r.Type == "TXT" {
			// Remove quotes from API response if present
			apiValue := r.Content
			if len(apiValue) >= 2 && apiValue[0] == '"' && apiValue[len(apiValue)-1] == '"' {
				apiValue = apiValue[1 : len(apiValue)-1]
			}
			valueMatches = (apiValue == record.Value)
		} else {
			valueMatches = (r.Content == record.Value)
		}

		// Compare using relative names
		if relativeName == record.Name && r.Type == record.Type && valueMatches {
			return r.ID, nil
		}
	}

	return 0, fmt.Errorf("record not found: %s %s", record.Name, record.Type)
}

// toLibdnsRecord converts a Rage4Record to a libdns.Record
// It converts the full FQDN name from Rage4 to a relative name for libdns
func toLibdnsRecord(r Rage4Record, zoneName string) libdns.Record {
	// Convert full name to relative name
	// If name equals zone, it's the root record (@)
	// Otherwise, strip the zone suffix
	var relativeName string
	if r.Name == zoneName {
		relativeName = "@"
	} else if strings.HasSuffix(r.Name, "."+zoneName) {
		relativeName = strings.TrimSuffix(r.Name, "."+zoneName)
	} else {
		// Fallback to the full name if it doesn't match the zone
		relativeName = r.Name
	}

	// Remove surrounding quotes from TXT records
	// Rage4 API automatically adds quotes to TXT record values
	value := r.Content
	if r.Type == "TXT" && len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}

	return libdns.Record{
		ID:       strconv.Itoa(r.ID),
		Type:     r.Type,
		Name:     relativeName,
		Value:    value,
		TTL:      time.Duration(r.TTL) * time.Second,
		Priority: uint(r.Priority),
		Weight:   uint(r.Weight),
	}
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
