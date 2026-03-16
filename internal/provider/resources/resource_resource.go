// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/rems-client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceResource{}
var _ resource.ResourceWithImportState = &ResourceResource{}
var _ resource.ResourceWithConfigure = &ResourceResource{}

func NewResourceResource() resource.Resource {
	return &ResourceResource{}
}

// ResourceResource defines the resource implementation.
type ResourceResource struct {
	BaseRemsResource
}

// ResourceResourceModel describes the resource data model.
type ResourceResourceModel struct {
	Id             types.Int64  `tfsdk:"id"`
	OrganizationId types.String `tfsdk:"organization_id"`
	Resid          types.String `tfsdk:"resid"`
	Licenses       *[]int64     `tfsdk:"licenses"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	Archived       types.Bool   `tfsdk:"archived"`
}

func (r *ResourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

func (r *ResourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a REMS resource, which represents a dataset or service that can be applied for via catalogue items.",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Resource internal identifier assigned by REMS.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the organization this resource belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resid": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "External resource identifier (e.g. a URI or dataset ID). Must be unique within the organization.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"licenses": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "List of license IDs that applicants must accept when applying for this resource.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether this resource is active. Defaults to `true`.",
			},
			"archived": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether this resource is archived. Defaults to `false`.",
			},
		},
	}
}

func (r *ResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	licenses := []int64{}
	if plan.Licenses != nil {
		licenses = *plan.Licenses
	}

	createCmd := remsclient.CreateResourceCommand{
		Organization: remsclient.OrganizationID{
			OrganizationID: plan.OrganizationId.ValueString(),
		},
		Resid:    plan.Resid.ValueString(),
		Licenses: licenses,
	}

	createResp, createErr := r.client.PostAPIResourcesCreateWithResponse(ctx, nil, createCmd)
	if createErr != nil {
		resp.Diagnostics.AddError("Error Creating Resource", createErr.Error())
		return
	}
	if createResp.StatusCode() != 200 || createResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Creating Resource", fmt.Sprintf("status: %d, body: %s", createResp.StatusCode(), string(createResp.Body)))
		return
	}
	if createResp.JSON200.ID == nil {
		resp.Diagnostics.AddError("Error Creating Resource", "API returned no ID")
		return
	}
	plan.Id = types.Int64Value(*createResp.JSON200.ID)

	enabledResp, enabledErr := r.client.PutAPIResourcesEnabledWithResponse(ctx, nil, remsclient.EnabledCommand{ID: plan.Id.ValueInt64(), Enabled: plan.Enabled.ValueBool()})
	if enabledErr != nil {
		resp.Diagnostics.AddError("Error Setting Resource Enabled State", enabledErr.Error())
		return
	}
	if enabledResp.StatusCode() != 200 || enabledResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Setting Resource Enabled State", fmt.Sprintf("status: %d, body: %s", enabledResp.StatusCode(), string(enabledResp.Body)))
		return
	}

	archivedResp, archivedErr := r.client.PutAPIResourcesArchivedWithResponse(ctx, nil, remsclient.ArchivedCommand{ID: plan.Id.ValueInt64(), Archived: plan.Archived.ValueBool()})
	if archivedErr != nil {
		resp.Diagnostics.AddError("Error Setting Resource Archived State", archivedErr.Error())
		return
	}
	if archivedResp.StatusCode() != 200 || archivedResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Setting Resource Archived State", fmt.Sprintf("status: %d, body: %s", archivedResp.StatusCode(), string(archivedResp.Body)))
		return
	}

	tflog.Trace(ctx, "created a resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readResp, readErr := r.client.GetAPIResourcesResourceIDWithResponse(ctx, state.Id.ValueInt64(), nil)
	if readErr != nil {
		resp.Diagnostics.AddError("Error Reading Resource", readErr.Error())
		return
	}
	if readResp.StatusCode() == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if readResp.StatusCode() != 200 || readResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Reading Resource", fmt.Sprintf("status: %d, body: %s", readResp.StatusCode(), string(readResp.Body)))
		return
	}

	item := readResp.JSON200
	state.Resid = types.StringValue(item.Resid)
	state.OrganizationId = types.StringValue(item.Organization.OrganizationID)
	state.Enabled = types.BoolValue(item.Enabled)
	state.Archived = types.BoolValue(item.Archived)

	state.Licenses = nil
	if len(item.Licenses) > 0 {
		ids := make([]int64, len(item.Licenses))
		for i, l := range item.Licenses {
			ids[i] = l.ID
		}
		state.Licenses = &ids
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	enabledResp, enabledErr := r.client.PutAPIResourcesEnabledWithResponse(ctx, nil, remsclient.EnabledCommand{ID: plan.Id.ValueInt64(), Enabled: plan.Enabled.ValueBool()})
	if enabledErr != nil {
		resp.Diagnostics.AddError("Error Setting Resource Enabled State", enabledErr.Error())
		return
	}
	if enabledResp.StatusCode() != 200 || enabledResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Setting Resource Enabled State", fmt.Sprintf("status: %d, body: %s", enabledResp.StatusCode(), string(enabledResp.Body)))
		return
	}

	archivedResp, archivedErr := r.client.PutAPIResourcesArchivedWithResponse(ctx, nil, remsclient.ArchivedCommand{ID: plan.Id.ValueInt64(), Archived: plan.Archived.ValueBool()})
	if archivedErr != nil {
		resp.Diagnostics.AddError("Error Setting Resource Archived State", archivedErr.Error())
		return
	}
	if archivedResp.StatusCode() != 200 || archivedResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Setting Resource Archived State", fmt.Sprintf("status: %d, body: %s", archivedResp.StatusCode(), string(archivedResp.Body)))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	archivedResp, archivedErr := r.client.PutAPIResourcesArchivedWithResponse(ctx, nil, remsclient.ArchivedCommand{ID: state.Id.ValueInt64(), Archived: true})
	if archivedErr != nil {
		resp.Diagnostics.AddError("Error Archiving Resource", archivedErr.Error())
		return
	}
	if archivedResp.StatusCode() != 200 || archivedResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Archiving Resource", fmt.Sprintf("status: %d, body: %s", archivedResp.StatusCode(), string(archivedResp.Body)))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Archived resource with ID: %d", state.Id.ValueInt64()))
}

func (r *ResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
