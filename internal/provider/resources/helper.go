package resources

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/rems-client"
)

func getLocalizedString(m *remsclient.LocalizedString) types.String {
	if m == nil {
		return types.StringNull()
	}
	val, ok := (*m)["en"]
	if !ok {
		return types.StringNull()
	}
	return types.StringValue(val)
}

func toLocalizedString(s types.String) *remsclient.LocalizedString {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}

	return &remsclient.LocalizedString{
		"en": s.ValueString(),
	}

}
