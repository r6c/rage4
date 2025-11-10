Rage4 for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/github.com/libdns/rage4.svg)](https://pkg.go.dev/github.com/libdns/rage4)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [Rage4 DNS](https://rage4.com/), allowing you to manage DNS records.

## Configuration

To use this provider, you need:

1. A Rage4 account with API access
2. Your account email address
3. Your API key (available in your Rage4 account settings)

## Usage

```go
package main

import (
	"context"
	"fmt"
	"github.com/libdns/libdns"
	"github.com/libdns/rage4"
)

func main() {
	provider := rage4.Provider{
		Email:  "your-email@example.com",
		APIKey: "your-api-key",
	}

	zone := "example.com."

	// Get all records
	records, err := provider.GetRecords(context.Background(), zone)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Append a new record
	newRecords, err := provider.AppendRecords(context.Background(), zone, []libdns.Record{
		{
			Type:  "A",
			Name:  "www",
			Value: "192.0.2.1",
			TTL:   3600,
		},
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Set records (replace existing ones with same name/type)
	setRecords, err := provider.SetRecords(context.Background(), zone, []libdns.Record{
		{
			Type:  "A",
			Name:  "www",
			Value: "192.0.2.2",
			TTL:   7200,
		},
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Delete records
	deletedRecords, err := provider.DeleteRecords(context.Background(), zone, newRecords)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
}
```

## Supported Record Types

This provider supports all standard DNS record types including:
- A (IPv4 address)
- AAAA (IPv6 address)
- CNAME (Canonical name)
- MX (Mail exchange)
- TXT (Text record)
- NS (Name server)
- SRV (Service record)
- And more...

## Notes

- Record names should be relative to the zone (e.g., "www" for "www.example.com." in zone "example.com.")
- Zone names should include the trailing dot (e.g., "example.com.")
- The provider uses HTTP Basic Authentication with your email and API key
- All operations are safe for concurrent use
