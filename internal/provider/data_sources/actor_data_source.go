// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
	_ datasource.DataSource              = &ActorDataSource{}
	_ datasource.DataSourceWithConfigure = &ActorDataSource{}
)

func NewActorDataSource() datasource.DataSource {
	return &ActorDataSource{}
}

// ActorDataSource defines the data source implementation.
type ActorDataSource struct {
	BaseRemsDataSource
}

// ActorDataSourceModel describes the data source data model.
type ActorDataSourceModel struct {
	Email types.String `tfsdk:"email"`
	Id    types.String `tfsdk:"id"`
}

func (d *ActorDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actor"
}

func (d *ActorDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Actor data source",
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

func (d *ActorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ActorDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	actorsResponse, actorsErr := d.client.GetAPIWorkflowsActorsWithResponse(ctx, nil)
	if actorsErr != nil {
		resp.Diagnostics.AddError("Error fetching actors list", actorsErr.Error())
		return
	}
	if actorsResponse.StatusCode() != 200 || actorsResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error fetching actors list", fmt.Sprintf("status: %d, body: %s", actorsResponse.StatusCode(), string(actorsResponse.Body)))
		return
	}

	actorsList := actorsResponse.JSON200

	for _, v := range *actorsList {

		if v.Email != nil && *v.Email == data.Email.ValueString() {
			data.Id = types.StringValue(v.Userid)
			break
		}

	}

	// Save to Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
