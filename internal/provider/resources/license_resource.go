// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/rems-client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &LicenseResource{}
var _ resource.ResourceWithImportState = &LicenseResource{}
var _ resource.ResourceWithValidateConfig = &LicenseResource{}

func NewLicenseResource() resource.Resource {
	return &LicenseResource{}
}

// LicenseResource defines the resource implementation.
type LicenseResource struct {
	BaseRemsResource
}

// LicenseResourceModel describes the resource data model.
type LicenseResourceModel struct {
	Id             types.Int64  `tfsdk:"id"`
	OrganizationId types.String `tfsdk:"organization_id"`
	Type           types.String `tfsdk:"type"`
	Title          types.String `tfsdk:"title"`
	Content        types.String `tfsdk:"content"`
	Path           types.String `tfsdk:"path"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	Archived       types.Bool   `tfsdk:"archived"`
}

func (r *LicenseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_license"
}
func (r *LicenseResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data LicenseResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if data.Type.ValueString() == "attachment" {
		if data.Path.IsNull() || data.Path.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("path"),
				"Missing path",
				"path must be set when type is attachment.",
			)
		}
	} else {
		if data.Content.IsNull() || data.Content.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("content"),
				"Missing content",
				"content must be set when type is text or link.",
			)
		}
	}

}
func (r *LicenseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a REMS license.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "License identifier.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the organization to associate this license with.",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: `License type. Must be one of: "link", "text", "attachment". Changing this forces a new resource.`,
				Validators: []validator.String{
					stringvalidator.OneOf("link", "text", "attachment"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"title": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "License title in English.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"content": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "License content. Required if type is not attachment",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "License filepath. Required for attachment type.",
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the license is active and ready to be used. Defaults to `true`.",
			},
			"archived": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If current state is archived. Defaults to `false`.",
			},
		},
	}
}

func (r *LicenseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LicenseResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	licensetype := remsclient.CreateLicenseCommandLicensetype(plan.Type.ValueString())

	licenseCommand := remsclient.CreateLicenseCommand{
		Licensetype: licensetype,
		Organization: remsclient.OrganizationID{
			OrganizationID: plan.OrganizationId.ValueString(),
		},
		Localizations: map[string]remsclient.LicenseLocalization{
			"en": {
				Title:       plan.Title.ValueString(),
				Textcontent: plan.Content.ValueString(),
			},
		},
	}

	// If the type is attachment, content is a file path
	if plan.Type.ValueString() == "attachment" {
		filePath := plan.Path.ValueString()
		attachment_id, err := r.uploadAttachment(ctx, filePath)

		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Attachment File",
				fmt.Sprintf("Unable to read file at path %s: %s", filePath, err),
			)
			return
		}

		enLoc := licenseCommand.Localizations["en"]
		enLoc.AttachmentID = &attachment_id
		enLoc.Textcontent = filepath.Base(filePath)
		licenseCommand.Localizations["en"] = enLoc

		// Ensure content is null because
		plan.Content = types.StringNull()

	}

	licenseResult, err := r.client.PostAPILicensesCreateWithResponse(
		ctx,
		nil,
		remsclient.PostAPILicensesCreateJSONRequestBody(licenseCommand),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating License",
			fmt.Sprintf("Unable to create license: %s", err),
		)
		return
	}

	if licenseResult.JSON200.ID == nil {
		resp.Diagnostics.AddError(
			"Error Creating License",
			"API returned a nil ID for the created license.",
		)
		return
	}

	plan.Id = types.Int64Value(*licenseResult.JSON200.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LicenseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LicenseResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	licenseResponse, err := r.client.GetAPILicensesLicenseIDWithResponse(ctx, state.Id.ValueInt64(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Error", err.Error())
		return
	}
	if licenseResponse.StatusCode() == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if licenseResponse.JSON200 == nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unexpected status: %s", licenseResponse.Status()))
		return
	}

	licenseResult := licenseResponse.JSON200

	state.OrganizationId = types.StringValue(licenseResult.Organization.OrganizationID)
	state.Type = types.StringValue(string(licenseResult.Licensetype))
	state.Archived = types.BoolValue(licenseResult.Archived)
	state.Enabled = types.BoolValue(licenseResult.Enabled)

	enLoc, _ := licenseResult.Localizations["en"]
	state.Title = types.StringValue(enLoc.Title)
	state.Content = types.StringValue(enLoc.Textcontent)

	// we ignore content for attachment to prevent conflicting compute and required
	if state.Type.ValueString() == "attachment" {
		state.Content = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LicenseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan LicenseResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	licenseResponse, err := r.client.GetAPILicensesLicenseIDWithResponse(ctx, plan.Id.ValueInt64(), nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading License",
			fmt.Sprintf("Unable to read license %d: %s", plan.Id.ValueInt64(), err),
		)
		return
	}

	licenseResult := licenseResponse.JSON200

	// archiving api requests
	if licenseResult.Archived != plan.Archived.ValueBool() {
		licenseArchiveCommand := remsclient.ArchivedCommand{
			ID:       plan.Id.ValueInt64(),
			Archived: plan.Archived.ValueBool(),
		}

		archivedResponse, _ := r.client.PutAPILicensesArchivedWithResponse(ctx, nil, licenseArchiveCommand)

		if archivedResponse.JSON200.Success == false {
			resp.Diagnostics.AddError(
				"Error Archiving/Unarchiving License",
				fmt.Sprintf("Unable to archive/unarchive license id: %d", plan.Id.ValueInt64()),
			)
			return
		}
	}

	// disabled api request
	if licenseResult.Enabled != plan.Enabled.ValueBool() {
		licenseEnabledCommand := remsclient.EnabledCommand{
			ID:      plan.Id.ValueInt64(),
			Enabled: plan.Enabled.ValueBool(),
		}

		enabledResponse, _ := r.client.PutAPILicensesEnabledWithResponse(ctx, nil, licenseEnabledCommand)
		if enabledResponse.JSON200.Success == false {
			resp.Diagnostics.AddError(
				"Error Enabled/Disabled License",
				fmt.Sprintf("Unable to enabled/disabled license id: %d", plan.Id.ValueInt64()),
			)
			return
		}

	}

	licenseResponse, _ = r.client.GetAPILicensesLicenseIDWithResponse(ctx, plan.Id.ValueInt64(), nil)
	licenseResult = licenseResponse.JSON200

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading License",
			fmt.Sprintf("Unable to read license %d: %s", plan.Id.ValueInt64(), err),
		)
		return
	}
	plan.Archived = types.BoolValue(licenseResult.Archived)
	plan.Enabled = types.BoolValue(licenseResult.Enabled)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

}

func (r *LicenseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LicenseResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// REMS does not support deletion - archive instead
	licenseArchiveCommand := remsclient.ArchivedCommand{
		ID:       state.Id.ValueInt64(),
		Archived: true,
	}
	archivedResponse, _ := r.client.PutAPILicensesArchivedWithResponse(ctx, nil, licenseArchiveCommand)

	if archivedResponse.JSON200.Success == false {
		resp.Diagnostics.AddError(
			"Error Archiving License",
			fmt.Sprintf("Unable to archive license id: %d", state.Id.ValueInt64()),
		)
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Archived license with ID: %d", state.Id.ValueInt64()))
}

func (r *LicenseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	// Convert the import ID string to int64
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

func (r *LicenseResource) uploadAttachment(ctx context.Context, filePath string) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("unable to open file: %s", err)
	}
	defer file.Close()

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return 0, fmt.Errorf("unable to create form file: %s", err)
	}

	if _, err = io.Copy(part, file); err != nil {
		return 0, fmt.Errorf("unable to copy file: %s", err)
	}
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.client.ClientInterface.(*remsclient.Client).Server+"api/licenses/add_attachment", &b)
	if err != nil {
		return 0, fmt.Errorf("unable to create request: %s", err)
	}

	underlyingClient := r.client.ClientInterface.(*remsclient.Client)
	for _, editor := range underlyingClient.RequestEditors {
		if err := editor(ctx, req); err != nil {
			return 0, fmt.Errorf("unable to apply request editor: %s", err)
		}
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("unable to upload attachment: %s", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 300 {
		body, _ := io.ReadAll(httpResp.Body)
		return 0, fmt.Errorf("upload failed with status %d: %s", httpResp.StatusCode, string(body))
	}
	var result remsclient.AddLicenseAttachmentResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("unable to decode response: %s", err)
	}

	if !result.Success {
		return 0, fmt.Errorf("attachment upload was not successful")
	}

	return result.ID, nil
}
