package data_sources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/umccr/terraform-provider-remscontent/internal/provider/shared"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/rems-client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &CategoryDataSource{}
	_ datasource.DataSourceWithConfigure = &CategoryDataSource{}
)

func NewCategoryDataSource() datasource.DataSource {
	return &CategoryDataSource{}
}

// CategoryDataSource defines the data source implementation.
type CategoryDataSource struct {
	BaseRemsDataSource
}

// CategoryDataSourceModel describes the data source data model.
type CategoryDataSourceModel struct {
	Id    types.Int64  `tfsdk:"id"`
	Title types.String `tfsdk:"title"`
}

func (d *CategoryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_category"
}

func (d *CategoryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Category data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The category ID",
				Computed:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The category title",
				Required:            true,
			},
		},
	}
}

func (d *CategoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CategoryDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	categoryResponse, categoryErr := d.client.GetAPICategoriesWithResponse(ctx, nil)
	if categoryErr != nil {
		resp.Diagnostics.AddError("Error fetching category list", categoryErr.Error())
		return
	}
	if categoryResponse.StatusCode() != 200 || categoryResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error fetching category list", fmt.Sprintf("status: %d, body: %s", categoryResponse.StatusCode(), string(categoryResponse.Body)))
		return
	}

	categoryList := *categoryResponse.JSON200
	if categoryList == nil {
		resp.Diagnostics.AddError("No category found", "The category list is nil.")
		return
	}

	var matchedCategory *remsclient.Category
	for _, v := range categoryList {

		localTitle := shared.GetLocalizedString(&v.CategoryTitle, d.language)

		if localTitle.ValueString() == data.Title.ValueString() {
			if matchedCategory != nil {
				resp.Diagnostics.AddError(
					"Multiple Category Found",
					fmt.Sprintf("More than one category found with title: %s", data.Title.ValueString()),
				)
				return
			}
			matchedCategory = &v
		}
	}

	data.Id = types.Int64Value(matchedCategory.CategoryID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
