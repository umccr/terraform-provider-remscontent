// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package data_sources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &OrganizationDataSource{}
	_ datasource.DataSourceWithConfigure = &OrganizationDataSource{}
)

func NewOrganizationDataSource() datasource.DataSource {
	return &OrganizationDataSource{}
}

// OrganizationDataSource defines the data source implementation.
type OrganizationDataSource struct {
	BaseRemsDataSource
}

// OrganizationDataSourceModel describes the data source data model.
type OrganizationDataSourceModel struct {
	Id types.String `tfsdk:"id"`
}

func (d *OrganizationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *OrganizationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Organization data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Organization id / Title",
				Required:            true,
			},
		},
	}
}

func (d *OrganizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OrganizationDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgsResult, _, err := d.client.OrganizationsAPI.
		ApiOrganizationsOrganizationIdGet(ctx, data.Id.ValueString()).
		Execute()

	if err != nil {
		resp.Diagnostics.AddError("Error reading organization", err.Error())
		return
	}

	jsonBytes, err := json.MarshalIndent(orgsResult, "", "  ")
	tflog.Info(ctx, fmt.Sprintf("OUTPUT - orgsResult:\n%s", string(jsonBytes)))

	// // 3. Map API response → Model
	// data.Name = types.StringValue(*orgResult.OrganizationId.OrganizationId)
	// data.Archived = types.BoolValue(*orgResult.Archived)
	// data.Enabled = types.BoolValue(*orgResult.Enabled)

	// 4. Save to Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
