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
	_ datasource.DataSource              = &CatalogueItemDataSource{}
	_ datasource.DataSourceWithConfigure = &CatalogueItemDataSource{}
)

func NewCatalogueItemDataSource() datasource.DataSource {
	return &CatalogueItemDataSource{}
}

// CatalogueItemDataSource defines the data source implementation.
type CatalogueItemDataSource struct {
	BaseRemsDataSource
}

// CatalogueItemDataSourceModel describes the data source data model.
type CatalogueItemDataSourceModel struct {
	Id    types.Int64  `tfsdk:"id"`
	Title types.String `tfsdk:"title"`
}

func (d *CatalogueItemDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catalogue_item"
}

func (d *CatalogueItemDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "CatalogueItem data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The catalogue_item ID",
				Computed:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The catalogue_item title",
				Required:            true,
			},
		},
	}
}

func (d *CatalogueItemDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CatalogueItemDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	catalogueItemResponse, catalogue_itemErr := d.client.GetAPICatalogueItemsWithResponse(ctx, nil)
	if catalogue_itemErr != nil {
		resp.Diagnostics.AddError("Error fetching catalogue_item list", catalogue_itemErr.Error())
		return
	}
	if catalogueItemResponse.StatusCode() != 200 || catalogueItemResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error fetching catalogue_item list", fmt.Sprintf("status: %d, body: %s", catalogueItemResponse.StatusCode(), string(catalogueItemResponse.Body)))
		return
	}

	catalogueItemList := *catalogueItemResponse.JSON200
	if catalogueItemList == nil {
		resp.Diagnostics.AddError("No catalogue_item found", "The catalogue_item list is nil.")
		return
	}

	var matchedCatalogueItem *remsclient.CatalogueItem
	for _, v := range catalogueItemList {

		localTitle := v.Localizations[d.language].Title
		if localTitle == data.Title.ValueString() {
			if matchedCatalogueItem != nil {
				resp.Diagnostics.AddError(
					"Multiple CatalogueItem Found",
					fmt.Sprintf("More than one catalogue_item found with title: %s", data.Title.ValueString()),
				)
				return
			}
			matchedCatalogueItem = &v
		}
	}

	if matchedCatalogueItem == nil {
		resp.Diagnostics.AddError(
			"Catalogue Item Not Found",
			fmt.Sprintf("No catalogueItem found with title: %s", data.Title.ValueString()),
		)
		return
	}

	data.Id = types.Int64Value(matchedCatalogueItem.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
