package data_sources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &BlacklistUserDataSource{}
	_ datasource.DataSourceWithConfigure = &BlacklistUserDataSource{}
)

func NewBlacklistUserDataSource() datasource.DataSource {
	return &BlacklistUserDataSource{}
}

// BlacklistUserDataSource defines the data source implementation.
type BlacklistUserDataSource struct {
	BaseRemsDataSource
}

// BlacklistUserDataSourceModel describes the data source data model.
type BlacklistUserDataSourceModel struct {
	Email types.String `tfsdk:"email"`
	Id    types.String `tfsdk:"id"`
}

func (d *BlacklistUserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blacklist_user"
}

func (d *BlacklistUserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "It only finds users who are available to be blacklisted (not currently blacklisted).",
		Attributes: map[string]schema.Attribute{
			"email": schema.StringAttribute{
				MarkdownDescription: "The email of the logged in user for lookup that is available as handler",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The user ID or username login",
				Computed:            true,
			},
		},
	}
}

func (d *BlacklistUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BlacklistUserDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	usersResponse, usersErr := d.client.GetAPIBlacklistUsersWithResponse(ctx, nil)
	if usersErr != nil {
		resp.Diagnostics.AddError("Error fetching user available for blacklist", usersErr.Error())
		return
	}
	if usersResponse.StatusCode() != 200 || usersResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error fetching user available for blacklist", fmt.Sprintf("status: %d, body: %s", usersResponse.StatusCode(), string(usersResponse.Body)))
		return
	}

	usersList := usersResponse.JSON200

	for _, v := range *usersList {
		if v.Email != nil && *v.Email == data.Email.ValueString() {
			data.Id = types.StringValue(v.Userid)
			break
		}
	}

	if data.Id.IsNull() {
		resp.Diagnostics.AddError(
			"User Not Found for Blacklist",
			fmt.Sprintf("No user available to blacklist with email: %s", data.Email.ValueString()),
		)
		return
	}

	// Save to Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
