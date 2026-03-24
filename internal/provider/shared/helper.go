package shared

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/rems-client"
)

// ProviderConfig is passed as ProviderData to all resources and data sources.
type ProviderConfig struct {
	Client   *remsclient.ClientWithResponses
	Language string // e.g. "en", "fr"
}

func GetLocalizedString(m *remsclient.LocalizedString, language string) types.String {
	if m == nil {
		return types.StringNull()
	}
	val, ok := (*m)[language]
	if !ok {
		return types.StringNull()
	}
	return types.StringValue(val)
}

// ToLocalizedStringValue returns a LocalizedString value for required fields that are
// guaranteed to be non-null and non-unknown (e.g. schema Required attributes).
func ToLocalizedStringValue(s types.String, language string) remsclient.LocalizedString {
	return remsclient.LocalizedString{
		language: s.ValueString(),
	}
}

func ToLocalizedString(s types.String, language string) *remsclient.LocalizedString {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}

	return &remsclient.LocalizedString{
		language: s.ValueString(),
	}
}
