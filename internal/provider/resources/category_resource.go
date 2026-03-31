package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/umccr/terraform-provider-remscontent/internal/provider/shared"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/rems-client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &CategoryResource{}
var _ resource.ResourceWithImportState = &CategoryResource{}
var _ resource.ResourceWithConfigure = &CategoryResource{}

func NewCategoryResource() resource.Resource {
	return &CategoryResource{}
}

// CategoryResource defines the resource implementation.
type CategoryResource struct {
	BaseRemsResource
}

// CategoryResourceModel describes the resource data model.
type CategoryResourceModel struct {
	Id           types.Int64  `tfsdk:"id"`
	Children     *[]int64     `tfsdk:"children"`
	Description  types.String `tfsdk:"description"`
	DisplayOrder types.Int64  `tfsdk:"display_order"`
	Title        types.String `tfsdk:"title"`
}

func (r *CategoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_category"
}

func (r *CategoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a REMS categories for the resource items.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Category identifier",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"children": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "List of categories children.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Category description.",
			},
			"display_order": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Display order for the category.",
			},
			"title": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Category title.",
			},
		},
	}
}

func (r *CategoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CategoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var categoriesChildren *[]remsclient.CategoryID
	if plan.Children != nil {
		s := make([]remsclient.CategoryID, len(*plan.Children))
		for i, v := range *plan.Children {
			s[i] = remsclient.CategoryID{CategoryID: v}
		}
		categoriesChildren = &s
	}

	cmd := remsclient.CreateCategoryCommand{
		CategoryChildren:     categoriesChildren,
		CategoryDescription:  shared.ToLocalizedString(plan.Description, r.language),
		CategoryDisplayOrder: plan.DisplayOrder.ValueInt64Pointer(),
		CategoryTitle:        shared.ToLocalizedStringValue(plan.Title, r.language),
	}

	addResp, addErr := r.client.PostAPICategoriesCreateWithResponse(ctx, nil, cmd)
	if addErr != nil {
		resp.Diagnostics.AddError("Error Adding Category Entry", addErr.Error())
		return
	}
	if addResp.StatusCode() != 200 || addResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Adding Category Entry", fmt.Sprintf("status: %d, body: %s", addResp.StatusCode(), string(addResp.Body)))
		return
	}

	// The swagger doesn't have proper definition of the category
	// Marshal then unmarshal to convert to the categoryCreateBody
	var categoryBody struct {
		Success    bool  `json:"success"`
		CategoryID int64 `json:"category/id"`
	}

	if err := json.Unmarshal(addResp.Body, &categoryBody); err != nil {
		resp.Diagnostics.AddError("Error Unmarshal Category Body Response", err.Error())
		return
	}

	if !categoryBody.Success {
		resp.Diagnostics.AddError("Error Adding Category Entry", fmt.Sprintf("API returned success=false. Full response: %s", string(addResp.Body)))
		return
	}
	plan.Id = types.Int64Value(categoryBody.CategoryID)

	tflog.Trace(ctx, "created a category entry")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CategoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CategoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readResp, readErr := r.client.GetAPICategoriesCategoryIDWithResponse(ctx, state.Id.ValueInt64(), nil)
	if readErr != nil {
		resp.Diagnostics.AddError("Error Reading Category", readErr.Error())
		return
	}
	if readResp.StatusCode() == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if readResp.StatusCode() != 200 || readResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Reading Category", fmt.Sprintf("status: %d, body: %s", readResp.StatusCode(), string(readResp.Body)))
		return
	}
	item := readResp.JSON200

	state.Children = nil
	if item.CategoryChildren != nil && len(*item.CategoryChildren) > 0 {
		categoryChildrenResp := *item.CategoryChildren
		child := make([]int64, len(categoryChildrenResp))
		for i, v := range categoryChildrenResp {
			child[i] = v.CategoryID
		}
		state.Children = &child
	}

	state.Description = shared.GetLocalizedString(item.CategoryDescription, r.language)
	state.DisplayOrder = types.Int64PointerValue(item.CategoryDisplayOrder)
	state.Title = shared.GetLocalizedString(&item.CategoryTitle, r.language)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CategoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CategoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var categoriesChildren *[]remsclient.CategoryID
	if plan.Children != nil {
		s := make([]remsclient.CategoryID, len(*plan.Children))
		for i, v := range *plan.Children {
			s[i] = remsclient.CategoryID{CategoryID: v}
		}
		categoriesChildren = &s
	}

	cmd := remsclient.UpdateCategoryCommand{
		CategoryID:           plan.Id.ValueInt64(),
		CategoryChildren:     categoriesChildren,
		CategoryDescription:  shared.ToLocalizedString(plan.Description, r.language),
		CategoryDisplayOrder: plan.DisplayOrder.ValueInt64Pointer(),
		CategoryTitle:        shared.ToLocalizedStringValue(plan.Title, r.language),
	}

	editResp, editErr := r.client.PutAPICategoriesEditWithResponse(ctx, nil, cmd)
	if editErr != nil {
		resp.Diagnostics.AddError("Error Edit Category: ", editErr.Error())
		return
	}
	if editResp.JSON200 == nil || !editResp.JSON200.Success {
		resp.Diagnostics.AddError("Error Edit Category Entry", fmt.Sprintf("status: %d, body: %s", editResp.StatusCode(), string(editResp.Body)))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CategoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CategoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cmd := remsclient.DeleteCategoryCommand{
		CategoryID: state.Id.ValueInt64(),
	}

	removeResp, removeErr := r.client.PostAPICategoriesDeleteWithResponse(ctx, nil, cmd)
	if removeErr != nil {
		resp.Diagnostics.AddError("Error Removing Category Entry", removeErr.Error())
		return
	}
	if removeResp.StatusCode() != 200 || removeResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Removing Category Entry", fmt.Sprintf("status: %d, body: %s", removeResp.StatusCode(), string(removeResp.Body)))
		return
	}

}

func (r *CategoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idInt, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Unable to convert import ID to integer: %s", err),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idInt)...)
}
