// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/rems-client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &FormResource{}
var _ resource.ResourceWithImportState = &FormResource{}
var _ resource.ResourceWithConfigure = &FormResource{}

func NewFormResource() resource.Resource {
	return &FormResource{}
}

func (r *FormResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_form"
}

// FormResource defines the resource implementation.
type FormResource struct {
	BaseRemsResource
}

/*
OpenAPI spec for forms

	{
	  "organization": {
	    "organization/id": "string"
	  },
	  "form/title": "string",
	  "form/internal-name": "string",
	  "form/external-title": {
	    "fi": "text in Finnish",
	    "en": "text in English"
	  },
	  "form/fields": [
	    {
	      "field/info-text": {
	        "fi": "text in Finnish",
	        "en": "text in English"
	      },
	      "field/title": {
	        "fi": "text in Finnish",
	        "en": "text in English"
	      },
	      "field/columns": [
	        {
	          "key": "string",
	          "label": {
	            "fi": "text in Finnish",
	            "en": "text in English"
	          }
	        }
	      ],
	      "field/max-length": 0,
	      "field/options": [
	        {
	          "key": "string",
	          "label": {
	            "fi": "text in Finnish",
	            "en": "text in English"
	          }
	        }
	      ],
	      "field/privacy": "private",
	      "field/visibility": {
	        "visibility/type": "only-if",
	        "visibility/field": {
	          "field/id": "string"
	        },
	        "visibility/values": [
	          "string"
	        ]
	      },
	      "field/type": "description",
	      "field/id": "string",
	      "field/optional": true,
	      "field/placeholder": {
	        "fi": "text in Finnish",
	        "en": "text in English"
	      }
	    }
	  ]
	}
*/
type FormFieldResourceModel struct {
	Type        types.String `tfsdk:"type"`
	Title       types.String `tfsdk:"title"`
	Info        types.String `tfsdk:"info"`
	Placeholder types.String `tfsdk:"placeholder"`
	Optional    types.Bool   `tfsdk:"optional"`
}

var fieldSchema = schema.NestedAttributeObject{
	Attributes: map[string]schema.Attribute{
		"type": schema.StringAttribute{
			Required: true,
			Validators: []validator.String{
				stringvalidator.OneOf(
					"description",
					"email",
					"date",
					"phone-number",
					"table",
					"header",
					"texta",
					"option",
					"label",
					"multiselect",
					"ip-address",
					"attachment",
					"text",
				),
			},
		},
		"title": schema.StringAttribute{
			Required: true,
		},
		"info": schema.StringAttribute{
			Optional: true,
		},
		"placeholder": schema.StringAttribute{
			Optional: true,
		},
		"optional": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			Default:  booldefault.StaticBool(false),
		},
	},
}

// FormResourceModel describes the resource data model.
type FormResourceModel struct {
	Id             types.Int64              `tfsdk:"id"`
	OrganizationId types.String             `tfsdk:"organization_id"`
	InternalName   types.String             `tfsdk:"internal_name"`
	ExternalTitle  types.String             `tfsdk:"external_title"`
	Fields         []FormFieldResourceModel `tfsdk:"fields"`
	Enabled        types.Bool               `tfsdk:"enabled"`
	Archived       types.Bool               `tfsdk:"archived"`
}

func (r *FormResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Form",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Form internal identifier",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the organization to associate this form with.",
				Required:            true,
			},
			"internal_name": schema.StringAttribute{
				MarkdownDescription: "Internal name for the form, visible to administrators only.",
				Required:            true,
			},
			"external_title": schema.StringAttribute{
				MarkdownDescription: "External title shown to applicants filling out the form.",
				Required:            true,
			},
			"fields": schema.ListNestedAttribute{
				NestedObject: fieldSchema,
				Optional:     true,
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the form is enabled and available for use. If set to `false`, the form will be inactive and cannot be used for new submissions. Defaults to `true`.",
			},
			"archived": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether the form is archived. Archived forms are not visible or usable in REMS. Defaults to `false`.",
			},
		},
	}
}

func (r *FormResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FormResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var fields []remsclient.NewFieldTemplate
	for _, item := range plan.Fields {
		fields = append(fields, remsclient.NewFieldTemplate{
			FieldType:        remsclient.NewFieldTemplateFieldType(item.Type.ValueString()),
			FieldTitle:       *toLocalizedString(item.Title),
			FieldOptional:    item.Optional.ValueBool(),
			FieldPlaceholder: toLocalizedString(item.Placeholder),
			FieldInfoText:    toLocalizedString(item.Info),
		})
	}

	formCreateCommand := remsclient.CreateFormCommand{
		Organization: remsclient.OrganizationID{
			OrganizationID: plan.OrganizationId.ValueString(),
		},
		FormInternalName: plan.InternalName.ValueStringPointer(),
		FormExternalTitle: &remsclient.LocalizedString{
			"en": plan.ExternalTitle.ValueString(),
		},
		FormFields: fields,
	}
	formResult, err := r.client.PostAPIFormsCreateWithResponse(ctx, nil, formCreateCommand)

	if err != nil || formResult.JSON200.ID == nil {

		var errorDetail string
		if err != nil {
			errorDetail = fmt.Sprintf("Unable to create form: %s", err)
		} else {
			errorDetail = "API returned a nil ID for the created license."
		}

		resp.Diagnostics.AddError(
			"Error Creating Form",
			errorDetail,
		)
		return
	}

	plan.Id = types.Int64Value(*formResult.JSON200.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

}

func (r *FormResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FormResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	formId := state.Id.ValueInt64()

	formItemResponse, err := r.client.GetAPIFormsFormIDWithResponse(ctx, formId, nil)
	formData := formItemResponse.JSON200

	if err != nil || formData == nil {
		resp.Diagnostics.AddError("Error", err.Error())
		return
	}

	state.Enabled = types.BoolValue(formData.Enabled)
	state.Archived = types.BoolValue(formData.Archived)
	state.InternalName = types.StringValue(formData.FormInternalName)
	state.OrganizationId = types.StringValue(formData.Organization.OrganizationID)
	state.ExternalTitle = types.StringValue(formData.FormExternalTitle["en"])

	state.Fields = []FormFieldResourceModel{}
	for _, formItem := range formData.FormFields {
		state.Fields = append(state.Fields, FormFieldResourceModel{
			Type:        types.StringValue(string(formItem.FieldType)),
			Title:       getLocalizedString(&formItem.FieldTitle),
			Info:        getLocalizedString(formItem.FieldInfoText),
			Placeholder: getLocalizedString(formItem.FieldPlaceholder),
			Optional:    types.BoolValue(formItem.FieldOptional),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *FormResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FormResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	formID := plan.Id.ValueInt64()

	// check if forms is editable
	isEditable, isEditableErr := r.client.GetAPIFormsFormIDEditableWithResponse(ctx, formID, nil)
	if isEditableErr != nil || isEditable.JSON200.Success == false {
		resp.Diagnostics.AddError(
			"Error Forms Not Editable",
			fmt.Sprintf("Unable to edit form id: %d", plan.Id.ValueInt64()),
		)
		return
	}

	var fields []remsclient.NewFieldTemplate
	for _, item := range plan.Fields {
		fields = append(fields, remsclient.NewFieldTemplate{
			FieldType:        remsclient.NewFieldTemplateFieldType(item.Type.ValueString()),
			FieldTitle:       *toLocalizedString(item.Title),
			FieldOptional:    item.Optional.ValueBool(),
			FieldPlaceholder: toLocalizedString(item.Placeholder),
			FieldInfoText:    toLocalizedString(item.Info),
		})
	}

	formUpdateCommand := remsclient.EditFormCommand{
		Organization: remsclient.OrganizationID{
			OrganizationID: plan.OrganizationId.ValueString(),
		},
		FormInternalName:  plan.InternalName.ValueStringPointer(),
		FormID:            plan.Id.ValueInt64(),
		FormExternalTitle: toLocalizedString(plan.ExternalTitle),
		FormFields:        fields,
	}

	updateResponse, updateErr := r.client.PutAPIFormsEditWithResponse(ctx, nil, formUpdateCommand)
	if updateErr != nil || updateResponse.JSON200 == nil || updateResponse.JSON200.Success == false {
		resp.Diagnostics.AddError(
			"Error Updating Form",
			fmt.Sprintf("Unable to edit on form id: %d", plan.Id.ValueInt64()),
		)
		return
	}

	// Update Archival state
	formArchiveCommand := remsclient.ArchivedCommand{
		ID:       formID,
		Archived: plan.Archived.ValueBool(),
	}
	archiveResponse, archiveErr := r.client.PutAPIFormsArchivedWithResponse(ctx, nil, formArchiveCommand)
	if archiveErr != nil || archiveResponse.JSON200 == nil || archiveResponse.JSON200.Success == false {
		resp.Diagnostics.AddError(
			"Error Setting Archiving Form",
			fmt.Sprintf("Unable to set archive/unarchive on form id: %d", plan.Id.ValueInt64()),
		)
		return
	}

	// Update Enabled state
	formEnabledCommand := remsclient.EnabledCommand{
		ID:      formID,
		Enabled: plan.Enabled.ValueBool(),
	}
	enabledResponse, enabledErr := r.client.PutAPIFormsEnabledWithResponse(ctx, nil, formEnabledCommand)
	if enabledErr != nil || enabledResponse.JSON200.Success == false {
		resp.Diagnostics.AddError(
			"Error Setting Enabled Form",
			fmt.Sprintf("Unable to set enabled/disabled on form id: %d", plan.Id.ValueInt64()),
		)
		return
	}

	formItemResponse, formItemErr := r.client.GetAPIFormsFormIDWithResponse(ctx, formID, nil)
	if formItemErr != nil {
		resp.Diagnostics.AddError(
			"Error Reading Updated Form",
			fmt.Sprintf("Could not read Form ID: %d", plan.Id.ValueInt64()),
		)
		return
	}

	plan.Fields = []FormFieldResourceModel{}
	for _, formItem := range formItemResponse.JSON200.FormFields {
		plan.Fields = append(plan.Fields, FormFieldResourceModel{
			Type:        types.StringValue(string(formItem.FieldType)),
			Title:       getLocalizedString(&formItem.FieldTitle),
			Info:        getLocalizedString(formItem.FieldInfoText),
			Placeholder: getLocalizedString(formItem.FieldPlaceholder),
			Optional:    types.BoolValue(formItem.FieldOptional),
		})
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FormResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FormResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	formArchiveCommand := remsclient.ArchivedCommand{
		ID:       state.Id.ValueInt64(),
		Archived: true,
	}
	archiveFormResponse, err := r.client.PutAPIFormsArchivedWithResponse(ctx, nil, formArchiveCommand)

	if err != nil || archiveFormResponse.JSON200.Success == false {
		resp.Diagnostics.AddError(
			"Error Archiving Form",
			fmt.Sprintf("Unable to archive form id: %d", state.Id.ValueInt64()),
		)
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Archived license with ID: %d", state.Id.ValueInt64()))

}

func (r *FormResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func (r *FormResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data FormResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

    // Add some config validation for the followong fields
    
	// regular: title, text, texta, date, email, phone, ip
	// optons: options, multi-select, table
	// no value: label, header

}
