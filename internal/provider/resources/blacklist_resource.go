package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/rems-client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BlacklistResource{}
var _ resource.ResourceWithImportState = &BlacklistResource{}
var _ resource.ResourceWithConfigure = &BlacklistResource{}

func NewBlacklistResource() resource.Resource {
	return &BlacklistResource{}
}

// BlacklistResource defines the resource implementation.
type BlacklistResource struct {
	BaseRemsResource
}

// BlacklistResourceModel describes the resource data model.
type BlacklistResourceModel struct {
	ResourceExtID types.String `tfsdk:"resource_ext_id"`
	UserID        types.String `tfsdk:"user_id"`
	Comment       types.String `tfsdk:"comment"`
}

func (r *BlacklistResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blacklist"
}

func (r *BlacklistResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a REMS blacklist entry, which blocks a specific user from accessing a specific resource.",

		Attributes: map[string]schema.Attribute{
			"resource_ext_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The external resource ID (e.g. a URI) to blacklist the user from.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The REMS user ID to blacklist.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"comment": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "A comment explaining the reason for the blacklist entry.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *BlacklistResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BlacklistResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cmd := remsclient.BlacklistCommand{
		BlacklistResource: remsclient.BlacklistCommandResource{
			ResourceExtID: plan.ResourceExtID.ValueString(),
		},
		BlacklistUser: remsclient.User{
			Userid: plan.UserID.ValueString(),
		},
		Comment: plan.Comment.ValueString(),
	}

	addResp, addErr := r.client.PostAPIBlacklistAddWithResponse(ctx, nil, cmd)
	if addErr != nil {
		resp.Diagnostics.AddError("Error Adding Blacklist Entry", addErr.Error())
		return
	}
	if addResp.StatusCode() != 200 || addResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Adding Blacklist Entry", fmt.Sprintf("status: %d, body: %s", addResp.StatusCode(), string(addResp.Body)))
		return
	}
	if !addResp.JSON200.Success {
		resp.Diagnostics.AddError("Error Adding Blacklist Entry", "API returned success=false")
		return
	}

	tflog.Trace(ctx, "created a blacklist entry")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BlacklistResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BlacklistResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceExtID := state.ResourceExtID.ValueString()
	userID := state.UserID.ValueString()

	readResp, readErr := r.client.GetAPIBlacklistWithResponse(ctx, &remsclient.GetAPIBlacklistParams{
		Resource: &resourceExtID,
		User:     &userID,
	})
	if readErr != nil {
		resp.Diagnostics.AddError("Error Reading Blacklist", readErr.Error())
		return
	}
	if readResp.StatusCode() == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if readResp.StatusCode() != 200 || readResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Reading Blacklist", fmt.Sprintf("status: %d, body: %s", readResp.StatusCode(), string(readResp.Body)))
		return
	}

	// If the entry is gone, remove from state.
	if len(*readResp.JSON200) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	// it shouldn't return more than 1
	if len(*readResp.JSON200) != 1 {
		resp.Diagnostics.AddError("Unexpected blacklist", "Error contain more than 1 blacklist ")
		return
	}

	blacklistResource := (*readResp.JSON200)[0]
	state.Comment = types.StringValue(blacklistResource.BlacklistComment)

	// The comment is not returned by the GET, so preserve whatever is in state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BlacklistResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes have RequiresReplace, so Update is never called.
}

func (r *BlacklistResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BlacklistResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cmd := remsclient.BlacklistCommand{
		BlacklistResource: remsclient.BlacklistCommandResource{
			ResourceExtID: state.ResourceExtID.ValueString(),
		},
		BlacklistUser: remsclient.User{
			Userid: state.UserID.ValueString(),
		},
		Comment: state.Comment.ValueString(),
	}

	removeResp, removeErr := r.client.PostAPIBlacklistRemoveWithResponse(ctx, nil, cmd)
	if removeErr != nil {
		resp.Diagnostics.AddError("Error Removing Blacklist Entry", removeErr.Error())
		return
	}
	if removeResp.StatusCode() != 200 || removeResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Removing Blacklist Entry", fmt.Sprintf("status: %d, body: %s", removeResp.StatusCode(), string(removeResp.Body)))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Removed blacklist entry: resource=%s user=%s", state.ResourceExtID, state.UserID.ValueString()))
}

func (r *BlacklistResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
