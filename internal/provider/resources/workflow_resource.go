// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/umccr/terraform-provider-remscontent/internal/provider/shared"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/rems-client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &WorkflowResource{}
var _ resource.ResourceWithImportState = &WorkflowResource{}

func NewWorkflowResource() resource.Resource {
	return &WorkflowResource{}
}

// WorkflowResource defines the resource implementation.
type WorkflowResource struct {
	BaseRemsResource
}

type DisableCommandModel struct {
	Command   types.String `tfsdk:"command"`
	WhenRole  []string     `tfsdk:"when_role"`
	WhenState []string     `tfsdk:"when_state"`
}

type ProcessingStateModel struct {
	Value types.String `tfsdk:"value"`
	Title types.String `tfsdk:"title"`
}

// WorkflowResourceModel describes the resource data model.
type WorkflowResourceModel struct {
	Id                types.Int64             `tfsdk:"id"`
	Title             types.String            `tfsdk:"title"`
	OrganizationID    types.String            `tfsdk:"organization_id"`
	Type              types.String            `tfsdk:"type"`
	Licenses          *[]int64                `tfsdk:"licenses"`
	Forms             *[]int64                `tfsdk:"forms"`
	Handlers          *[]string               `tfsdk:"handlers"`
	AnonymizeHandling types.Bool              `tfsdk:"anonymize_handling"`
	DisableCommands   *[]DisableCommandModel  `tfsdk:"disable_commands"`
	ProcessingStates  *[]ProcessingStateModel `tfsdk:"processing_states"`
	Enabled           types.Bool              `tfsdk:"enabled"`
	Archived          types.Bool              `tfsdk:"archived"`
}

func (r *WorkflowResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (r *WorkflowResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a REMS workflow, which defines the review process for applications submitted to a catalogue item.",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Workflow internal identifier assigned by REMS.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable title for the workflow, visible to administrators.",
			},
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the organization this workflow belongs to.",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Workflow type. Use `workflow/default` for standard handler-based review or `workflow/decider` for decider-based review.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"workflow/default",
						"workflow/decider",
					),
				},
			},
			"licenses": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "List of license IDs that applicants must accept when submitting through this workflow.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"forms": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "List of form IDs attached to this workflow. Applicants will fill these forms on submission.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"handlers": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of handler user IDs (CILogon `userid`) responsible for reviewing applications. Use the `remscontent_actor` data source to look up a user ID by email.",
			},
			"anonymize_handling": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "When `true`, handler identities are hidden from applicants. Defaults to `false`.",
			},
			"disable_commands": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Rules to disable specific application commands for certain roles or states. see: https://github.com/CSCfi/rems/blob/master/docs/application-permissions.md",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"command": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The application command to disable (e.g. `accept‑licenses`).",
						},
						"when_role": schema.ListAttribute{
							Optional:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "Roles for which this command is disabled.",
						},
						"when_state": schema.ListAttribute{
							Optional:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "Application states in which this command is disabled.",
						},
					},
				},
			},
			"processing_states": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Custom processing states visible during application review.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Internal value for the processing state.",
						},
						"title": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Human-readable title for the processing state.",
						},
					},
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether this workflow is active. Disabled workflows cannot be assigned to new catalogue items. Defaults to `true`.",
			},
			"archived": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether this workflow is archived. Archived workflows are hidden from administrators. Defaults to `false`.",
			},
		},
	}
}

func (r *WorkflowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WorkflowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var formsCmd *[]remsclient.CreateWorkflowCommandForms
	if plan.Forms != nil {
		s := make([]remsclient.CreateWorkflowCommandForms, len(*plan.Forms))
		for i, v := range *plan.Forms {
			s[i] = remsclient.CreateWorkflowCommandForms{FormID: v}
		}
		formsCmd = &s
	}

	var licensesCmd *[]remsclient.LicenseID
	if plan.Licenses != nil {
		s := make([]remsclient.LicenseID, len(*plan.Licenses))
		for i, v := range *plan.Licenses {
			s[i] = remsclient.LicenseID{LicenseID: v}
		}
		licensesCmd = &s
	}

	var disableCommandsCmd *[]remsclient.DisableCommandRule
	if plan.DisableCommands != nil {
		s := make([]remsclient.DisableCommandRule, len(*plan.DisableCommands))
		for i, v := range *plan.DisableCommands {
			s[i] = remsclient.DisableCommandRule{
				Command:   v.Command.ValueString(),
				WhenRole:  &v.WhenRole,
				WhenState: &v.WhenState,
			}
		}
		disableCommandsCmd = &s
	}

	var processingStatesCmd *[]remsclient.ProcessingState
	if plan.ProcessingStates != nil {
		s := make([]remsclient.ProcessingState, len(*plan.ProcessingStates))
		for i, v := range *plan.ProcessingStates {
			s[i] = remsclient.ProcessingState{
				ProcessingStateValue: v.Value.ValueString(),
				ProcessingStateTitle: shared.ToLocalizedString(v.Title),
			}
		}
		processingStatesCmd = &s
	}

	workflowCreateResourceCommand := remsclient.CreateWorkflowCommand{
		AnonymizeHandling: plan.AnonymizeHandling.ValueBoolPointer(),
		Forms:             formsCmd,
		Handlers:          plan.Handlers,
		Licenses:          licensesCmd,
		Organization: remsclient.OrganizationID{
			OrganizationID: plan.OrganizationID.ValueString(),
		},
		Title:            plan.Title.ValueString(),
		Type:             remsclient.CreateWorkflowCommandType(plan.Type.ValueString()),
		DisableCommands:  disableCommandsCmd,
		ProcessingStates: processingStatesCmd,
	}

	workflowCreateResponse, workflowCreateErr := r.client.PostAPIWorkflowsCreateWithResponse(ctx, nil, workflowCreateResourceCommand)
	if workflowCreateErr != nil {
		resp.Diagnostics.AddError("Error Creating Workflow", workflowCreateErr.Error())
		return
	}
	if workflowCreateResponse.StatusCode() != 200 || workflowCreateResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error Creating Workflow", fmt.Sprintf("status: %d, body: %s", workflowCreateResponse.StatusCode(), string(workflowCreateResponse.Body)))
		return
	}

	plan.Id = types.Int64Value(*workflowCreateResponse.JSON200.ID)
	wfEnabledResponse, wfEnabledErr := r.client.PutAPIWorkflowsEnabledWithResponse(ctx, nil, remsclient.EnabledCommand{ID: plan.Id.ValueInt64(), Enabled: plan.Enabled.ValueBool()})
	if wfEnabledErr != nil {
		resp.Diagnostics.AddError("Error Enabled/Disabled Workflow", wfEnabledErr.Error())
		return
	}
	if wfEnabledResponse.StatusCode() != 200 || wfEnabledResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error Enabled/Disabled Workflow", fmt.Sprintf("status: %d, body: %s", wfEnabledResponse.StatusCode(), string(wfEnabledResponse.Body)))
		return
	}

	wfArchivedResponse, wfArchivedErr := r.client.PutAPIWorkflowsArchivedWithResponse(ctx, nil, remsclient.ArchivedCommand{ID: plan.Id.ValueInt64(), Archived: plan.Archived.ValueBool()})
	if wfArchivedErr != nil {
		resp.Diagnostics.AddError("Error Archiving/Unarchiving Workflow", wfArchivedErr.Error())
		return
	}
	if wfArchivedResponse.StatusCode() != 200 || wfArchivedResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error Archiving/Unarchiving Workflow", fmt.Sprintf("status: %d, body: %s", wfArchivedResponse.StatusCode(), string(wfArchivedResponse.Body)))
		return
	}

	tflog.Trace(ctx, "created a workflow resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WorkflowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wfId := state.Id.ValueInt64()

	wfResponse, wfErr := r.client.GetAPIWorkflowsWorkflowIDWithResponse(ctx, wfId, nil)
	if wfErr != nil {
		resp.Diagnostics.AddError("Error Retrieving Workflow", wfErr.Error())
		return
	}
	if wfResponse.StatusCode() != 200 || wfResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error Retrieving Workflow", fmt.Sprintf("status: %d, body: %s", wfResponse.StatusCode(), string(wfResponse.Body)))
		return
	}

	wfItem := wfResponse.JSON200

	// The swagger doesn't have proper definition of the workflow detail
	// Marshal then unmarshal to convert to the workflowBody
	type workflowBody struct {
		Type              string `json:"type"`
		AnonymizeHandling bool   `json:"anonymize-handling"`
		Handlers          []struct {
			UserID string `json:"userid"`
		} `json:"handlers"`
		Licenses []struct {
			LicenseID int64 `json:"license/id"`
		} `json:"licenses"`
		Forms []struct {
			FormID int64 `json:"form/id"`
		} `json:"forms"`
	}
	wfBodyBytes, err := json.Marshal(wfItem.Workflow)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal workflow body", err.Error())
		return
	}
	var wfBody workflowBody
	if err := json.Unmarshal(wfBodyBytes, &wfBody); err != nil {
		resp.Diagnostics.AddError("Failed to parse workflow body", err.Error())
		return
	}

	state.Title = types.StringValue(wfItem.Title)
	state.OrganizationID = types.StringValue(wfItem.Organization.OrganizationID)
	state.Enabled = types.BoolValue(wfItem.Enabled)
	state.Archived = types.BoolValue(wfItem.Archived)
	state.Type = types.StringValue(wfBody.Type)
	state.AnonymizeHandling = types.BoolValue(wfBody.AnonymizeHandling)

	state.Handlers = nil
	if len(wfBody.Handlers) > 0 {
		handlers := make([]string, len(wfBody.Handlers))
		for i, h := range wfBody.Handlers {
			handlers[i] = h.UserID
		}
		state.Handlers = &handlers
	}

	// Forms
	state.Forms = nil
	if len(wfBody.Forms) > 0 {
		forms := make([]int64, len(wfBody.Forms))
		for i, f := range wfBody.Forms {
			forms[i] = f.FormID
		}
		state.Forms = &forms
	}

	// Licenses
	state.Licenses = nil
	if len(wfBody.Licenses) > 0 {
		licenses := make([]int64, len(wfBody.Licenses))
		for i, l := range wfBody.Licenses {
			licenses[i] = l.LicenseID
		}
		state.Licenses = &licenses
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WorkflowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WorkflowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wfEnabledResponse, wfEnabledErr := r.client.PutAPIWorkflowsEnabledWithResponse(ctx, nil, remsclient.EnabledCommand{ID: plan.Id.ValueInt64(), Enabled: plan.Enabled.ValueBool()})
	if wfEnabledErr != nil {
		resp.Diagnostics.AddError("Error Enabled/Disabled Workflow", wfEnabledErr.Error())
		return
	}
	if wfEnabledResponse.StatusCode() != 200 || wfEnabledResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error Enabled/Disabled Workflow", fmt.Sprintf("status: %d, body: %s", wfEnabledResponse.StatusCode(), string(wfEnabledResponse.Body)))
		return
	}

	wfArchivedResponse, wfArchivedErr := r.client.PutAPIWorkflowsArchivedWithResponse(ctx, nil, remsclient.ArchivedCommand{ID: plan.Id.ValueInt64(), Archived: plan.Archived.ValueBool()})
	if wfArchivedErr != nil {
		resp.Diagnostics.AddError("Error Archiving/Unarchiving Workflow", wfArchivedErr.Error())
		return
	}
	if wfArchivedResponse.StatusCode() != 200 || wfArchivedResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error Archiving/Unarchiving Workflow", fmt.Sprintf("status: %d, body: %s", wfArchivedResponse.StatusCode(), string(wfArchivedResponse.Body)))
		return
	}

	var disableCommandsCmd *[]remsclient.DisableCommandRule
	if plan.DisableCommands != nil {
		s := make([]remsclient.DisableCommandRule, len(*plan.DisableCommands))
		for i, v := range *plan.DisableCommands {
			s[i] = remsclient.DisableCommandRule{
				Command:   v.Command.ValueString(),
				WhenRole:  &v.WhenRole,
				WhenState: &v.WhenState,
			}
		}
		disableCommandsCmd = &s
	}

	var processingStatesCmd *[]remsclient.ProcessingState
	if plan.ProcessingStates != nil {
		s := make([]remsclient.ProcessingState, len(*plan.ProcessingStates))
		for i, v := range *plan.ProcessingStates {
			s[i] = remsclient.ProcessingState{
				ProcessingStateValue: v.Value.ValueString(),
				ProcessingStateTitle: shared.ToLocalizedString(v.Title),
			}
		}
		processingStatesCmd = &s
	}

	workflowEditResourceCommand := remsclient.EditWorkflowCommand{
		AnonymizeHandling: plan.AnonymizeHandling.ValueBoolPointer(),
		DisableCommands:   disableCommandsCmd,
		Handlers:          plan.Handlers,
		ID:                plan.Id.ValueInt64(),
		Organization: &remsclient.OrganizationID{
			OrganizationID: plan.OrganizationID.ValueString(),
		},
		ProcessingStates: processingStatesCmd,
		Title:            plan.Title.ValueStringPointer(),
	}

	workflowEditResponse, workflowCreateErr := r.client.PutAPIWorkflowsEditWithResponse(ctx, nil, workflowEditResourceCommand)
	if workflowCreateErr != nil {
		resp.Diagnostics.AddError("Error Enabled/Disabled Workflow", workflowCreateErr.Error())
		return
	}
	if workflowEditResponse.StatusCode() != 200 || workflowEditResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error Enabled/Disabled Workflow", fmt.Sprintf("status: %d, body: %s", workflowEditResponse.StatusCode(), string(workflowEditResponse.Body)))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WorkflowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wfArchivedResponse, wfArchivedErr := r.client.PutAPIWorkflowsArchivedWithResponse(ctx, nil, remsclient.ArchivedCommand{ID: state.Id.ValueInt64(), Archived: true})
	if wfArchivedErr != nil {
		resp.Diagnostics.AddError("Error Archiving/Unarchiving Workflow", wfArchivedErr.Error())
		return
	}
	if wfArchivedResponse.StatusCode() != 200 || wfArchivedResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error Archiving/Unarchiving Workflow", fmt.Sprintf("status: %d, body: %s", wfArchivedResponse.StatusCode(), string(wfArchivedResponse.Body)))
		return
	}

}

func (r *WorkflowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
