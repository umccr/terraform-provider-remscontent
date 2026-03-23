package data_sources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/rems-client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &FormDataSource{}
	_ datasource.DataSourceWithConfigure = &FormDataSource{}
)

func NewFormDataSource() datasource.DataSource {
	return &FormDataSource{}
}

// FormDataSource defines the data source implementation.
type FormDataSource struct {
	BaseRemsDataSource
}

// FormDataSourceModel describes the data source data model.
type FormDataSourceModel struct {
	Id           types.Int64  `tfsdk:"id"`
	InternalName types.String `tfsdk:"internal_name"`
}

func (d *FormDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_form"
}

func (d *FormDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Form data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The form ID",
				Computed:            true,
			},
			"internal_name": schema.StringAttribute{
				MarkdownDescription: "Internal name for the form, visible to administrators only.",
				Required:            true,
			},
		},
	}
}

func (d *FormDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FormDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	formsResponse, formsErr := d.client.GetAPIFormsWithResponse(ctx, nil)
	if formsErr != nil {
		resp.Diagnostics.AddError("Error fetching forms list", formsErr.Error())
		return
	}
	if formsResponse.StatusCode() != 200 || formsResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error fetching forms list", fmt.Sprintf("status: %d, body: %s", formsResponse.StatusCode(), string(formsResponse.Body)))
		return
	}

	formsList := *formsResponse.JSON200
	if formsList == nil {
		resp.Diagnostics.AddError("No forms found", "The forms list is nil.")
		return
	}

	var matchedForm *remsclient.FormTemplateOverview
	for _, v := range formsList {

		if v.FormInternalName == data.InternalName.ValueString() {
			if matchedForm != nil {
				resp.Diagnostics.AddError(
					"Multiple Form Found",
					fmt.Sprintf("More than one form found with title: %s", data.InternalName.ValueString()),
				)
				return
			}
			matchedForm = &v
		}
	}

	data.Id = types.Int64Value(matchedForm.FormID)

	// Save to Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
