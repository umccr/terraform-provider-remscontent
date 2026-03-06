package shared

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/rems-client"
)

func GetLocalizedString(m *remsclient.LocalizedString) types.String {
	if m == nil {
		return types.StringNull()
	}
	val, ok := (*m)["en"]
	if !ok {
		return types.StringNull()
	}
	return types.StringValue(val)
}

func ToLocalizedString(s types.String) *remsclient.LocalizedString {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}

	return &remsclient.LocalizedString{
		"en": s.ValueString(),
	}

}

func HandleAPIError(diags *diag.Diagnostics, summary string, err error, statusCode int, body []byte) bool {
	if err != nil {
		diags.AddError(summary, err.Error())
		return true
	}
	if statusCode != 200 {
		diags.AddError(summary, fmt.Sprintf("status: %d, body: %s", statusCode, string(body)))
		return true
	}
	return false
}
