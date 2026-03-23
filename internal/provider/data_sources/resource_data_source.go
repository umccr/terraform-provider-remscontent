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
	_ datasource.DataSource              = &ResourceDataSource{}
	_ datasource.DataSourceWithConfigure = &ResourceDataSource{}
)

func NewResourceDataSource() datasource.DataSource {
	return &ResourceDataSource{}
}

// ResourceDataSource defines the data source implementation.
type ResourceDataSource struct {
	BaseRemsDataSource
}

// ResourceDataSourceModel describes the data source data model.
type ResourceDataSourceModel struct {
	Id            types.Int64  `tfsdk:"id"`
	ResourceExtID types.String `tfsdk:"resource_ext_id"`
}

func (d *ResourceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

func (d *ResourceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Resource data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The resource ID",
				Computed:            true,
			},
			"resource_ext_id": schema.StringAttribute{
				MarkdownDescription: "The external resource identifier or URN to look up. This should match the external ID assigned to the resource in REMS.",
				Required:            true,
			},
		},
	}
}

func (d *ResourceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ResourceDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourcesResponse, resourcesErr := d.client.GetAPIResourcesWithResponse(ctx, nil)
	if resourcesErr != nil {
		resp.Diagnostics.AddError("Error fetching resources list", resourcesErr.Error())
		return
	}
	if resourcesResponse.StatusCode() != 200 || resourcesResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error fetching resources list", fmt.Sprintf("status: %d, body: %s", resourcesResponse.StatusCode(), string(resourcesResponse.Body)))
		return
	}

	resourcesList := *resourcesResponse.JSON200
	if resourcesList == nil {
		resp.Diagnostics.AddError("No resources found", "The resources list is nil.")
		return
	}

	var matchedResource *remsclient.Resource
	for _, v := range resourcesList {

		if v.Resid == data.ResourceExtID.ValueString() {
			if matchedResource != nil {
				resp.Diagnostics.AddError(
					"Multiple Resource Found",
					fmt.Sprintf("More than one resource found with ResourceExternalId: %s", data.ResourceExtID.ValueString()),
				)
				return
			}
			matchedResource = &v
		}
	}
	if matchedResource == nil {
		resp.Diagnostics.AddError(
			"Resource Not Found",
			fmt.Sprintf("No resource found with ResourceExternalId: %s", data.ResourceExtID.ValueString()),
		)
		return
	}

	data.Id = types.Int64Value(matchedResource.ID)

	// Save to Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
