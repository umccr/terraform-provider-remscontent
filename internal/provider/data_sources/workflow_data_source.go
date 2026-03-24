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
	_ datasource.DataSource              = &WorkflowDataSource{}
	_ datasource.DataSourceWithConfigure = &WorkflowDataSource{}
)

func NewWorkflowDataSource() datasource.DataSource {
	return &WorkflowDataSource{}
}

// WorkflowDataSource defines the data source implementation.
type WorkflowDataSource struct {
	BaseRemsDataSource
}

// WorkflowDataSourceModel describes the data source data model.
type WorkflowDataSourceModel struct {
	Id    types.Int64  `tfsdk:"id"`
	Title types.String `tfsdk:"title"`
}

func (d *WorkflowDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (d *WorkflowDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Workflow data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The workflow ID",
				Computed:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The title of the workflow to look up. If multiple workflows have the same title, an error is returned.",
				Required:            true,
			},
		},
	}
}

func (d *WorkflowDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data WorkflowDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workflowsResponse, workflowsErr := d.client.GetAPIWorkflowsWithResponse(ctx, nil)
	if workflowsErr != nil {
		resp.Diagnostics.AddError("Error fetching workflows list", workflowsErr.Error())
		return
	}
	if workflowsResponse.StatusCode() != 200 || workflowsResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error fetching workflows list", fmt.Sprintf("status: %d, body: %s", workflowsResponse.StatusCode(), string(workflowsResponse.Body)))
		return
	}

	workflowsList := *workflowsResponse.JSON200
	if workflowsList == nil {
		resp.Diagnostics.AddError("No workflows found", "The workflows list is nil.")
		return
	}

	var matchedWorkflow *remsclient.Workflow
	for _, v := range workflowsList {

		if v.Title == data.Title.ValueString() {
			if matchedWorkflow != nil {
				resp.Diagnostics.AddError(
					"Multiple Workflow Found",
					fmt.Sprintf("More than one workflow found with title: %s", data.Title.ValueString()),
				)
				return
			}
			matchedWorkflow = &v
		}
	}

	if matchedWorkflow == nil {
		resp.Diagnostics.AddError(
			"Workflow Not Found",
			fmt.Sprintf("No workflow found with title: %s", data.Title.ValueString()),
		)
		return
	}

	data.Id = types.Int64Value(matchedWorkflow.ID)

	// Save to Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
