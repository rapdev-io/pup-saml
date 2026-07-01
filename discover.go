package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/rapdev-io/pup-saml/internal/pupapi"
)

func runDiscover(org string) {
	client := pupapi.New(org)
	out := DiscoverOutput{Org: org}

	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		errs []string
	)

	addErr := func(msg string) {
		mu.Lock()
		errs = append(errs, msg)
		mu.Unlock()
	}

	var roleLookup map[string]string
	var roleLookupMu sync.Mutex
	roleLookupReady := make(chan struct{})

	wg.Add(6)

	// 1. Org settings
	go func() {
		defer wg.Done()
		var settings orgSettings
		if err := client.Get("v1/org", &settings); err != nil {
			addErr(fmt.Sprintf("org settings: %v", err))
			return
		}
		a := settings.Data.Attributes
		mu.Lock()
		out.SAMLConfig = SAMLConfig{
			Enabled:               a.Saml.Enabled,
			Enforced:              a.SamlStrictMode.Enabled,
			IdpInitiatedLogin:     a.SamlIdpInitiatedLogin.Enabled,
			IdpMetadataUploaded:   a.SamlIdpMetadataUploaded,
			LoginURL:              a.SamlLoginURL,
			AutoCreateDomains:     a.SamlAutoCreateUsersDomains,
			AutoCreateDefaultRole: a.SamlAutoCreateAccessRole,
			BitsAIRisk:            a.SamlAutoCreateAccessRole == "st",
		}
		mu.Unlock()
	}()

	// 2. SP metadata
	go func() {
		defer wg.Done()
		body, err := client.GetRaw("v1/saml/metadata")
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			out.SPMetadata = SPMetadata{Error: "not available"}
			return
		}
		var ed EntityDescriptor
		if err := xml.Unmarshal(body, &ed); err != nil {
			out.SPMetadata = SPMetadata{Error: "failed to parse XML"}
			return
		}
		sp := SPMetadata{EntityID: ed.EntityID}
		for _, acs := range ed.SPSSODescriptor.AssertionConsumerServices {
			if strings.Contains(acs.Binding, "POST") {
				sp.ACSURL = acs.Location
				break
			}
		}
		for _, kd := range ed.SPSSODescriptor.KeyDescriptors {
			if kd.Use == "signing" && kd.Info.X509Data.Certificate != "" {
				sp.Certificate = fmt.Sprintf("present (len=%d)", len(kd.Info.X509Data.Certificate))
				break
			}
		}
		out.SPMetadata = sp
	}()

	// 3. Authn mappings — waits for role lookup
	go func() {
		defer wg.Done()
		var resp authnMappingsResp
		if err := client.Get("v2/authn_mappings", &resp); err != nil {
			addErr(fmt.Sprintf("authn_mappings: %v", err))
			return
		}
		<-roleLookupReady
		roleLookupMu.Lock()
		rl := roleLookup
		roleLookupMu.Unlock()

		mu.Lock()
		for _, m := range resp.Data {
			out.AuthnMappings = append(out.AuthnMappings, MappingEntry{
				ID:             m.ID,
				AttributeKey:   m.Attributes.AttributeKey,
				AttributeValue: m.Attributes.AttributeValue,
				RoleID:         m.Relationships.Role.Data.ID,
				RoleName:       rl[m.Relationships.Role.Data.ID],
			})
		}
		mu.Unlock()
	}()

	// 4. Roles — builds roleLookup used by goroutines 3 and 5
	go func() {
		defer wg.Done()
		var resp rolesResp
		if err := client.Get("v2/roles?page[size]=100", &resp); err != nil {
			addErr(fmt.Sprintf("roles: %v", err))
			close(roleLookupReady)
			return
		}
		rl := make(map[string]string, len(resp.Data))
		for _, r := range resp.Data {
			rl[r.ID] = r.Attributes.Name
		}
		roleLookupMu.Lock()
		roleLookup = rl
		roleLookupMu.Unlock()
		close(roleLookupReady)

		mu.Lock()
		for _, r := range resp.Data {
			info := RoleInfo{
				ID:        r.ID,
				Name:      r.Attributes.Name,
				UserCount: r.Attributes.UserCount,
				BitsAI:    strings.Contains(strings.ToLower(r.Attributes.Name), "standard"),
			}
			if r.Attributes.BuiltIn {
				out.Roles.BuiltIn = append(out.Roles.BuiltIn, info)
			} else {
				out.Roles.Custom = append(out.Roles.Custom, info)
			}
		}
		mu.Unlock()
	}()

	// 5. Users
	go func() {
		defer wg.Done()
		rawUsers, err := client.Paginate("v2/users?filter[status]=active")
		if err != nil {
			mu.Lock()
			out.Users = UserDistribution{Error: err.Error()}
			mu.Unlock()
			return
		}
		dist := UserDistribution{Total: len(rawUsers)}
		var passwordEmails, noRoleEmails []string

		for _, raw := range rawUsers {
			var u userEntry
			if err := json.Unmarshal(raw, &u); err != nil {
				continue
			}
			switch u.Attributes.Status {
			case "active":
				dist.Active++
			case "disabled":
				dist.Disabled++
			default:
				dist.Pending++
			}
			if u.Attributes.Status != "active" {
				continue
			}
			if !u.Attributes.SamlEnabled {
				dist.PasswordOnly++
				if len(passwordEmails) < 20 {
					passwordEmails = append(passwordEmails, u.Attributes.Email)
				}
			} else {
				dist.SSOOnly++
			}
			if len(u.Relationships.Roles.Data) == 0 {
				dist.NoRole++
				if len(noRoleEmails) < 20 {
					noRoleEmails = append(noRoleEmails, u.Attributes.Email)
				}
			}
		}
		dist.PasswordUsers = passwordEmails
		dist.NoRoleUsers = noRoleEmails
		mu.Lock()
		out.Users = dist
		mu.Unlock()
	}()

	// 6. Service accounts
	go func() {
		defer wg.Done()
		body, err := client.GetRaw("v2/service_accounts")
		if err != nil {
			return
		}
		var resp struct {
			Data []serviceAccount `json:"data"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return
		}
		mu.Lock()
		for _, sa := range resp.Data {
			out.ServiceAccounts = append(out.ServiceAccounts, ServiceAccountInfo{
				ID:    sa.ID,
				Name:  sa.Attributes.Name,
				Email: sa.Attributes.Email,
			})
		}
		mu.Unlock()
	}()

	wg.Wait()

	// Key findings
	if out.SAMLConfig.BitsAIRisk {
		out.Findings = append(out.Findings, "Auto-provisioning default role is Standard — new users will inherit bits_ai_queries (billing risk)")
	}
	if out.Users.PasswordOnly > 0 {
		out.Findings = append(out.Findings, fmt.Sprintf("%d active users are not SAML-enabled and will be locked out if SAML is enforced", out.Users.PasswordOnly))
	}
	if out.Users.NoRole > 0 {
		out.Findings = append(out.Findings, fmt.Sprintf("%d active users have no role assigned — will be unrouted after migration", out.Users.NoRole))
	}
	if len(out.AuthnMappings) == 0 {
		out.Findings = append(out.Findings, "No authn_mappings configured — run /dd-saml-audit to generate target config")
	}
	if out.SPMetadata.Error != "" {
		out.Findings = append(out.Findings, "SP metadata unavailable — SAML may not be fully configured")
	}
	for _, e := range errs {
		out.Findings = append(out.Findings, "error: "+e)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "pup-saml: encoding output: %v\n", err)
		os.Exit(1)
	}
}
