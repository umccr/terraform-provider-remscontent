// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
	_ datasource.DataSource              = &LicenseDataSource{}
	_ datasource.DataSourceWithConfigure = &LicenseDataSource{}
)

func NewLicenseDataSource() datasource.DataSource {
	return &LicenseDataSource{}
}

// LicenseDataSource defines the data source implementation.
type LicenseDataSource struct {
	BaseRemsDataSource
}

// LicenseDataSourceModel describes the data source data model.
type LicenseDataSourceModel struct {
	Id    types.Int64  `tfsdk:"id"`
	Title types.String `tfsdk:"title"`
}

func (d *LicenseDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_license"
}

func (d *LicenseDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "License data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "License id",
				Computed:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The title in english for the lookup",
				Required:            true,
			},
		},
	}
}

func (d *LicenseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LicenseDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	licenseResponse, err := d.client.GetAPILicensesWithResponse(ctx, nil)
	licenseResult := *licenseResponse.JSON200

	if err != nil {
		resp.Diagnostics.AddError("Error reading license", err.Error())
		return
	}
	if licenseResult == nil {
		resp.Diagnostics.AddError("No license found", "The license list is nil.")
		return
	}

	var matchedLicense *remsclient.License
	for _, license := range licenseResult {
		if enLocalizations, ok := license.Localizations["en"]; ok {

			if enLocalizations.Title == data.Title.ValueString() {
				if matchedLicense != nil {
					resp.Diagnostics.AddError(
						"Multiple Licenses Found",
						fmt.Sprintf("More than one license found with title: %s", data.Title.ValueString()),
					)
					return
				}
				matchedLicense = &license
			}
		}
	}

	// Check if any license was found
	if matchedLicense == nil {
		resp.Diagnostics.AddError(
			"License Not Found",
			fmt.Sprintf("No license found with title: %s", data.Title.ValueString()),
		)
		return
	}

	data.Id = types.Int64Value(int64(matchedLicense.ID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
