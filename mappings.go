package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rapdev-io/pup-saml/internal/pupapi"
)

func runMappings(org string) {
	client := pupapi.New(org)

	roleLookup := map[string]string{}
	var rolesData rolesResp
	if err := client.Get("v2/roles?page[size]=100", &rolesData); err == nil {
		for _, r := range rolesData.Data {
			roleLookup[r.ID] = r.Attributes.Name
		}
	}

	var resp authnMappingsResp
	if err := client.Get("v2/authn_mappings", &resp); err != nil {
		fmt.Fprintf(os.Stderr, "pup-saml: %v\n", err)
		os.Exit(1)
	}

	out := make([]MappingEntry, 0, len(resp.Data))
	for _, m := range resp.Data {
		out = append(out, MappingEntry{
			ID:             m.ID,
			AttributeKey:   m.Attributes.AttributeKey,
			AttributeValue: m.Attributes.AttributeValue,
			RoleID:         m.Relationships.Role.Data.ID,
			RoleName:       roleLookup[m.Relationships.Role.Data.ID],
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}
