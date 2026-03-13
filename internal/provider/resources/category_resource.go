// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
				Optional:            true,
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
		CategoryDescription:  shared.ToLocalizedString(plan.Description),
		CategoryDisplayOrder: plan.DisplayOrder.ValueInt64Pointer(),
		CategoryTitle:        *shared.ToLocalizedString(plan.Title),
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
	categoryItem := addResp.JSON200

	// The swagger doesn't have proper definition of the category
	// Marshal then unmarshal to convert to the categoryCreateBody
	type CategoryCreateBody struct {
		Id      int64 `json:"category/id"`
		Success bool  `json:"success"`
	}
	categoryBodyBytes, err := json.Marshal(categoryItem)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal workflow body", err.Error())
		return
	}
	var categoryBody CategoryCreateBody
	if err := json.Unmarshal(categoryBodyBytes, &categoryBody); err != nil {
		resp.Diagnostics.AddError("Failed to parse workflow body", err.Error())
		return
	}

	if categoryBody.Success == false {
		resp.Diagnostics.AddError("Error Adding Category Entry", "API returned success=false")

	}
	plan.Id = types.Int64Value(categoryBody.Id)

	tflog.Trace(ctx, "created a blacklist entry")
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
	if readResp.StatusCode() != 200 || readResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Reading Category", fmt.Sprintf("status: %d, body: %s", readResp.StatusCode(), string(readResp.Body)))
		return
	}
	item := readResp.JSON200

	state.Children = nil
	categoryChildrenResp := *item.CategoryChildren
	if len(categoryChildrenResp) > 0 {
		child := make([]int64, len(categoryChildrenResp))
		for i, v := range categoryChildrenResp {
			child[i] = v.CategoryID
		}
		state.Children = &child
	}

	state.Description = shared.GetLocalizedString(item.CategoryDescription)
	state.DisplayOrder = types.Int64PointerValue(item.CategoryDisplayOrder)
	state.Title = shared.GetLocalizedString(&item.CategoryTitle)

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
		CategoryChildren:     categoriesChildren,
		CategoryDescription:  shared.ToLocalizedString(plan.Description),
		CategoryDisplayOrder: plan.DisplayOrder.ValueInt64Pointer(),
		CategoryTitle:        *shared.ToLocalizedString(plan.Title),
	}

	editResp, editErr := r.client.PutAPICategoriesEditWithResponse(ctx, nil, cmd)
	if editErr != nil {
		resp.Diagnostics.AddError("Error Edit Category: ", editErr.Error())
		return
	}
	if editResp.StatusCode() != 200 || editResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Adding Category Entry", fmt.Sprintf("status: %d, body: %s", editResp.StatusCode(), string(editResp.Body)))
		return
	}

	if editResp.JSON200.Success == false {
		resp.Diagnostics.AddError("Error Edit Category Entry", "API returned success=false")
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
	// Import ID format: "resource_ext_id|user_id"
	parts := strings.SplitN(req.ID, "|", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			`Expected import ID in the format "resource_ext_id|user_id", e.g. "urn:example:dataset1|alice"`,
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_ext_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("comment"), "")...)
}
