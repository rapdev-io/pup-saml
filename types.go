package main

// ── Datadog API response types ───────────────────────────────────────────────

type orgSettings struct {
	Data struct {
		Attributes struct {
			Name string `json:"name"`
			Saml struct {
				Enabled bool `json:"enabled"`
			} `json:"saml"`
			SamlStrictMode struct {
				Enabled bool `json:"enabled"`
			} `json:"saml_strict_mode"`
			SamlIdpInitiatedLogin struct {
				Enabled bool `json:"enabled"`
			} `json:"saml_idp_initiated_login"`
			SamlIdpMetadataUploaded    bool     `json:"saml_idp_metadata_uploaded"`
			SamlLoginURL               string   `json:"saml_login_url"`
			SamlAutoCreateUsersDomains []string `json:"saml_autocreate_users_domains"`
			SamlAutoCreateAccessRole   string   `json:"saml_autocreate_access_role"`
		} `json:"attributes"`
	} `json:"data"`
}

type authnMapping struct {
	ID         string `json:"id"`
	Attributes struct {
		AttributeKey   string `json:"attribute_key"`
		AttributeValue string `json:"attribute_value"`
		CreatedAt      string `json:"created_at"`
	} `json:"attributes"`
	Relationships struct {
		Role struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"role"`
	} `json:"relationships"`
}

type authnMappingsResp struct {
	Data []authnMapping `json:"data"`
}

type roleEntry struct {
	ID         string `json:"id"`
	Attributes struct {
		Name      string `json:"name"`
		BuiltIn   bool   `json:"built_in"`
		UserCount int    `json:"user_count"`
	} `json:"attributes"`
}

type rolesResp struct {
	Data []roleEntry `json:"data"`
}

type userEntry struct {
	Attributes struct {
		Email        string `json:"email"`
		Status       string `json:"status"`
		LoginMethods any    `json:"login_methods"`
		SamlEnabled  bool   `json:"saml_can_be_enabled"`
	} `json:"attributes"`
	Relationships struct {
		Roles struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"roles"`
	} `json:"relationships"`
}

type serviceAccount struct {
	ID         string `json:"id"`
	Attributes struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"attributes"`
}

// ── SP metadata XML ──────────────────────────────────────────────────────────

type EntityDescriptor struct {
	EntityID        string          `xml:"entityID,attr"`
	SPSSODescriptor SPSSODescriptor `xml:"SPSSODescriptor"`
}

type SPSSODescriptor struct {
	KeyDescriptors            []KeyDescriptor            `xml:"KeyDescriptor"`
	AssertionConsumerServices []AssertionConsumerService `xml:"AssertionConsumerService"`
}

type KeyDescriptor struct {
	Use  string  `xml:"use,attr"`
	Info KeyInfo `xml:"KeyInfo"`
}

type KeyInfo struct {
	X509Data X509Data `xml:"X509Data"`
}

type X509Data struct {
	Certificate string `xml:"X509Certificate"`
}

type AssertionConsumerService struct {
	Binding  string `xml:"Binding,attr"`
	Location string `xml:"Location,attr"`
	Index    string `xml:"index,attr"`
}

// ── Output types ─────────────────────────────────────────────────────────────

type SAMLConfig struct {
	Enabled               bool     `json:"enabled"`
	Enforced              bool     `json:"enforced"`
	IdpInitiatedLogin     bool     `json:"idp_initiated_login"`
	IdpMetadataUploaded   bool     `json:"idp_metadata_uploaded"`
	LoginURL              string   `json:"login_url,omitempty"`
	AutoCreateDomains     []string `json:"autocreate_domains,omitempty"`
	AutoCreateDefaultRole string   `json:"autocreate_default_role,omitempty"`
	BitsAIRisk            bool     `json:"bits_ai_risk"`
}

type SPMetadata struct {
	EntityID    string `json:"entity_id,omitempty"`
	ACSURL      string `json:"acs_url,omitempty"`
	Certificate string `json:"certificate_subject,omitempty"`
	Error       string `json:"error,omitempty"`
}

type MappingEntry struct {
	ID             string `json:"id"`
	AttributeKey   string `json:"attribute_key"`
	AttributeValue string `json:"attribute_value"`
	RoleID         string `json:"role_id"`
	RoleName       string `json:"role_name"`
}

type RoleSummary struct {
	BuiltIn []RoleInfo `json:"builtin"`
	Custom  []RoleInfo `json:"custom"`
}

type RoleInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UserCount int    `json:"user_count"`
	BitsAI    bool   `json:"bits_ai_risk,omitempty"`
}

type UserDistribution struct {
	Total         int      `json:"total"`
	Active        int      `json:"active"`
	Disabled      int      `json:"disabled"`
	Pending       int      `json:"pending"`
	SSOOnly       int      `json:"sso_only"`
	PasswordOnly  int      `json:"password_only"`
	NoRole        int      `json:"no_role"`
	PasswordUsers []string `json:"password_only_emails,omitempty"`
	NoRoleUsers   []string `json:"no_role_emails,omitempty"`
	Error         string   `json:"error,omitempty"`
}

type ServiceAccountInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type DiscoverOutput struct {
	Org             string               `json:"org"`
	SAMLConfig      SAMLConfig           `json:"saml_config"`
	SPMetadata      SPMetadata           `json:"sp_metadata"`
	AuthnMappings   []MappingEntry       `json:"authn_mappings"`
	Roles           RoleSummary          `json:"roles"`
	Users           UserDistribution     `json:"users"`
	ServiceAccounts []ServiceAccountInfo `json:"service_accounts"`
	Findings        []string             `json:"key_findings"`
}
