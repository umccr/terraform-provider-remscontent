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
	"github.com/umccr/terraform-provider-remscontent/internal/provider/shared"

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

type KeyLabelModel struct {
	Key   types.String `tfsdk:"key"`
	Label types.String `tfsdk:"label"`
}

var keyLabelSchema = schema.NestedAttributeObject{
	Attributes: map[string]schema.Attribute{
		"key": schema.StringAttribute{
			Required: true,
		},
		"label": schema.StringAttribute{
			Required: true,
		},
	},
}

type VisibilityModel struct {
	VisibilityType types.String `tfsdk:"visibility_type"`
	FieldId        types.String `tfsdk:"field_id"`
	HasValue       *[]string    `tfsdk:"has_value"`
}

type FormFieldResourceModel struct {
	Id          types.String     `tfsdk:"id"`
	Type        types.String     `tfsdk:"type"`
	Title       types.String     `tfsdk:"title"`
	Info        types.String     `tfsdk:"info"`
	Placeholder types.String     `tfsdk:"placeholder"`
	Optional    types.Bool       `tfsdk:"optional"`
	Options     *[]KeyLabelModel `tfsdk:"options"`
	Columns     *[]KeyLabelModel `tfsdk:"columns"`
	MaxLength   types.Int64      `tfsdk:"max_length"`
	Privacy     types.String     `tfsdk:"privacy"`
	Visibility  *VisibilityModel `tfsdk:"visibility"`
}

var fieldSchema = schema.NestedAttributeObject{
	Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Field identifier. Must be unique within the form. Any string is allowed; the simplest approach is to use incrementing numbers (e.g., 1, 2, 3), but any unique string is valid.",
		},
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
			Required:            true,
			MarkdownDescription: "The title for the field",
		},
		"info": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Field description explain applicants what the field about.",
		},
		"placeholder": schema.StringAttribute{
			Optional: true,
		},
		"optional": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			Default:  booldefault.StaticBool(false),
		},
		"options": schema.ListNestedAttribute{
			NestedObject: keyLabelSchema,
			Optional:     true,
		},
		"columns": schema.ListNestedAttribute{
			NestedObject: keyLabelSchema,
			Optional:     true,
		},
		"max_length": schema.Int64Attribute{
			Optional:            true,
			MarkdownDescription: "maximum character for the field",
		},
		"privacy": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Visibility of the field. Default: `public`",
			Validators: []validator.String{
				stringvalidator.OneOf(
					"public",
					"private",
				),
			},
		},
		"visibility": schema.SingleNestedAttribute{
			Optional:            true,
			MarkdownDescription: "Defines conditional visibility for this field, driven by the value of a referenced option or multiselect field.",
			Attributes: map[string]schema.Attribute{
				"visibility_type": schema.StringAttribute{
					Required:            true,
					MarkdownDescription: "When to show this field. Use `always` to always show, or `only-if` to show only when another field matches a specific value. Default: always",
					Validators: []validator.String{
						stringvalidator.OneOf(
							"only-if", "always",
						),
					},
				},
				"field_id": schema.StringAttribute{
					Optional:            true,
					MarkdownDescription: "The ID of the field this visibility depends on. Only applies when `visibility_type` is `only-if`.",
				},
				"has_value": schema.ListAttribute{
					Optional:            true,
					MarkdownDescription: "List of option keys that the field referenced by `field_id` must match for this field to be visible. Only applies when `visibility_type` is `only-if`.",
					ElementType:         types.StringType,
				},
			},
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
				Required:     true,
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

	formCreateCommand := remsclient.CreateFormCommand{
		Organization: remsclient.OrganizationID{
			OrganizationID: plan.OrganizationId.ValueString(),
		},
		FormInternalName:  plan.InternalName.ValueStringPointer(),
		FormExternalTitle: shared.ToLocalizedString(plan.ExternalTitle, r.language),
		FormFields:        fromFormFieldModels(plan.Fields, r.language),
	}
	formResponse, formErr := r.client.PostAPIFormsCreateWithResponse(ctx, nil, formCreateCommand)
	if formErr != nil {
		resp.Diagnostics.AddError("Error Creating Form", formErr.Error())
		return
	}
	if formResponse.JSON200 == nil || formResponse.JSON200.ID == nil {
		resp.Diagnostics.AddError("Error Creating Form", fmt.Sprintf("status: %d, body: %s", formResponse.StatusCode(), string(formResponse.Body)))
		return
	}
	planId := *formResponse.JSON200.ID
	plan.Id = types.Int64Value(planId)

	// Update Archival state
	formArchiveCommand := remsclient.ArchivedCommand{
		ID:       planId,
		Archived: plan.Archived.ValueBool(),
	}
	archiveResponse, archiveErr := r.client.PutAPIFormsArchivedWithResponse(ctx, nil, formArchiveCommand)
	if archiveErr != nil {
		resp.Diagnostics.AddError("Error Archiving/Unarchiving Form", archiveErr.Error())
		return
	}
	if archiveResponse.JSON200 == nil || !archiveResponse.JSON200.Success {
		resp.Diagnostics.AddError("Error Archiving/Unarchiving Form", fmt.Sprintf("status: %d, body: %s", archiveResponse.StatusCode(), string(archiveResponse.Body)))
		return
	}

	// Update Enabled state
	formEnabledCommand := remsclient.EnabledCommand{
		ID:      planId,
		Enabled: plan.Enabled.ValueBool(),
	}
	enabledResponse, enabledErr := r.client.PutAPIFormsEnabledWithResponse(ctx, nil, formEnabledCommand)
	if enabledErr != nil {
		resp.Diagnostics.AddError("Error Enabled/Disabled Form", enabledErr.Error())
		return
	}
	if enabledResponse.JSON200 == nil || !enabledResponse.JSON200.Success {
		resp.Diagnostics.AddError("Enabled/Disabled Form is Unsuccessful", fmt.Sprintf("status: %d, body: %s", enabledResponse.StatusCode(), string(enabledResponse.Body)))
		return
	}

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

	formItemResponse, fromItemErr := r.client.GetAPIFormsFormIDWithResponse(ctx, formId, nil)
	if fromItemErr != nil {
		resp.Diagnostics.AddError("Error Reading Form", fromItemErr.Error())
		return
	}
	if formItemResponse.StatusCode() == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if formItemResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error Reading Form", fmt.Sprintf("status: %d, body: %s", formItemResponse.StatusCode(), string(formItemResponse.Body)))
		return
	}
	formData := formItemResponse.JSON200

	state.Enabled = types.BoolValue(formData.Enabled)
	state.Archived = types.BoolValue(formData.Archived)
	state.InternalName = types.StringValue(formData.FormInternalName)
	state.OrganizationId = types.StringValue(formData.Organization.OrganizationID)
	state.ExternalTitle = shared.GetLocalizedString(&formData.FormExternalTitle, r.language)

	state.Fields = []FormFieldResourceModel{}
	for _, formItem := range formData.FormFields {
		state.Fields = append(state.Fields, toFormFieldModel(formItem, r.language))
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

	// check if forms is editable before calling update
	isEditable, isEditableErr := r.client.GetAPIFormsFormIDEditableWithResponse(ctx, formID, nil)
	if isEditableErr != nil {
		resp.Diagnostics.AddError("Error Checking Editable Form", isEditableErr.Error())
		return
	}
	if isEditable.StatusCode() != 200 || isEditable.JSON200 == nil {
		resp.Diagnostics.AddError("Error Checking Editable Form", fmt.Sprintf("status: %d, body: %s", isEditable.StatusCode(), string(isEditable.Body)))
		return
	}
	if !isEditable.JSON200.Success {
		resp.Diagnostics.AddError("Form is not editable. Details:", string(isEditable.Body))
		return
	}

	formUpdateCommand := remsclient.EditFormCommand{
		Organization: remsclient.OrganizationID{
			OrganizationID: plan.OrganizationId.ValueString(),
		},
		FormInternalName:  plan.InternalName.ValueStringPointer(),
		FormID:            plan.Id.ValueInt64(),
		FormExternalTitle: shared.ToLocalizedString(plan.ExternalTitle, r.language),
		FormFields:        fromFormFieldModels(plan.Fields, r.language),
	}

	updateResponse, updateErr := r.client.PutAPIFormsEditWithResponse(ctx, nil, formUpdateCommand)
	if updateErr != nil {
		resp.Diagnostics.AddError("Error Updating Form", updateErr.Error())
		return
	}
	if updateResponse.StatusCode() != 200 || updateResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error Updating Form", fmt.Sprintf("status: %d, body: %s", updateResponse.StatusCode(), string(updateResponse.Body)))
		return
	}

	// Update Archival state
	formArchiveCommand := remsclient.ArchivedCommand{
		ID:       formID,
		Archived: plan.Archived.ValueBool(),
	}
	archiveResponse, archiveErr := r.client.PutAPIFormsArchivedWithResponse(ctx, nil, formArchiveCommand)
	if archiveErr != nil {
		resp.Diagnostics.AddError("Error Archiving/Unarchiving Form", archiveErr.Error())
		return
	}
	if archiveResponse.JSON200 == nil || !archiveResponse.JSON200.Success {
		resp.Diagnostics.AddError("Error Archiving/Unarchiving Form", fmt.Sprintf("status: %d, body: %s", archiveResponse.StatusCode(), string(archiveResponse.Body)))
		return
	}

	// Update Enabled state
	formEnabledCommand := remsclient.EnabledCommand{
		ID:      formID,
		Enabled: plan.Enabled.ValueBool(),
	}
	enabledResponse, enabledErr := r.client.PutAPIFormsEnabledWithResponse(ctx, nil, formEnabledCommand)
	if enabledErr != nil {
		resp.Diagnostics.AddError("Error Enabled/Disabled Form", enabledErr.Error())
		return
	}
	if enabledResponse.JSON200 == nil || !enabledResponse.JSON200.Success {
		resp.Diagnostics.AddError("Error Enabled/Disabled Form", fmt.Sprintf("status: %d, body: %s", enabledResponse.StatusCode(), string(enabledResponse.Body)))
		return
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
	if err != nil {
		resp.Diagnostics.AddError("Error Archiving/Unarchiving Form", err.Error())
		return
	}
	if archiveFormResponse.StatusCode() != 200 || archiveFormResponse.JSON200 == nil {
		resp.Diagnostics.AddError("Error Archiving/Unarchiving Form", fmt.Sprintf("status: %d, body: %s", archiveFormResponse.StatusCode(), string(archiveFormResponse.Body)))
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

	// Only check if fields are present at the form
	if len(data.Fields) == 0 {
		return
	}

	// Build a map of field ID to valid option keys for each field.
	// This enables validation of visibility rules, ensuring that:
	// - The referenced field_id exists in the form.
	// - Each value in has_value is a valid option key for the referenced field.
	// Used for cross-field validation in form configuration.
	fieldOptionKeys := make(map[string]map[string]bool, len(data.Fields))
	for i, f := range data.Fields {
		if !f.Id.IsNull() && !f.Id.IsUnknown() {
			fieldId := f.Id.ValueString()

			// Check for duplicate IDs
			if _, duplicate := fieldOptionKeys[fieldId]; duplicate {
				resp.Diagnostics.AddAttributeError(
					path.Root("fields").AtListIndex(i).AtName("id"),
					"Duplicate field ID",
					fmt.Sprintf("Field ID `%s` must be unique within the form.", fieldId),
				)
			}

			keys := make(map[string]bool)
			if f.Options != nil {
				for _, opt := range *f.Options {
					if !opt.Key.IsNull() && !opt.Key.IsUnknown() {
						keys[opt.Key.ValueString()] = true
					}
				}
			}
			fieldOptionKeys[f.Id.ValueString()] = keys
		}
	}

	// Iterate every fields in the array
	for i, item := range data.Fields {
		fieldPath := path.Root("fields").AtListIndex(i)
		fieldType := item.Type.ValueString()

		// option/multiselect must have options
		shouldHaveOptions := fieldType == "option" || fieldType == "multiselect"
		if shouldHaveOptions && item.Options == nil {
			resp.Diagnostics.AddAttributeError(
				fieldPath.AtName("options"),
				"Missing options",
				fmt.Sprintf("`options` is required when field type is `%s`.", fieldType),
			)
		}
		if !shouldHaveOptions && item.Options != nil {
			resp.Diagnostics.AddAttributeError(
				fieldPath.AtName("options"),
				"Unexpected options",
				fmt.Sprintf("`options` should not be set when field type is `%s`.", fieldType),
			)
		}

		// table must have columns
		if fieldType == "table" && item.Columns == nil {
			resp.Diagnostics.AddAttributeError(
				fieldPath.AtName("columns"),
				"Missing columns",
				"`columns` is required when field type is `table`.",
			)
		}
		if fieldType != "table" && item.Columns != nil {
			resp.Diagnostics.AddAttributeError(
				fieldPath.AtName("columns"),
				"Unexpected columns",
				"`columns` should only be set when field type is `table`.",
			)
		}

		// visibility only-if must have field_id and has_value
		// visibility always must not have field_id and has_value
		if item.Visibility != nil {
			visibilityPath := fieldPath.AtName("visibility")
			visibilityType := item.Visibility.VisibilityType.ValueString()

			switch visibilityType {
			case "only-if":
				if item.Visibility.FieldId.IsNull() {
					resp.Diagnostics.AddAttributeError(
						visibilityPath.AtName("field_id"),
						"Missing field_id",
						"`field_id` is required when `visibility_type` is `only-if`.",
					)
				} else {
					// Check for refId existence with other field's id
					refId := item.Visibility.FieldId.ValueString()
					optionKeys, exists := fieldOptionKeys[refId]
					if !exists {
						resp.Diagnostics.AddAttributeError(
							visibilityPath.AtName("field_id"),
							"Invalid field_id",
							fmt.Sprintf("`field_id` `%s` does not reference any field in this form.", refId),
						)
					} else if item.Visibility.HasValue != nil {
						// check if the has_value is referring to the options available from the refId
						for _, val := range *item.Visibility.HasValue {
							if !optionKeys[val] {
								resp.Diagnostics.AddAttributeError(
									visibilityPath.AtName("has_value"),
									"Invalid has_value",
									fmt.Sprintf("`%s` is not a valid option key in field `%s`.", val, refId),
								)
							}
						}
					}

				}

				if item.Visibility.HasValue == nil || len(*item.Visibility.HasValue) == 0 {
					resp.Diagnostics.AddAttributeError(
						visibilityPath.AtName("has_value"),
						"Missing has_value",
						"`has_value` is required when `visibility_type` is `only-if`.",
					)
				}
			case "always":
				if !item.Visibility.FieldId.IsNull() {
					resp.Diagnostics.AddAttributeError(
						visibilityPath.AtName("field_id"),
						"Unexpected field_id",
						"`field_id` should not be set when `visibility_type` is `always`.",
					)
				}
				if item.Visibility.HasValue != nil && len(*item.Visibility.HasValue) > 0 {
					resp.Diagnostics.AddAttributeError(
						visibilityPath.AtName("has_value"),
						"Unexpected has_value",
						"`has_value` should not be set when `visibility_type` is `always`.",
					)
				}
			}
		}
	}

}

// Helper function.
func toKeyLabelOptions(item *[]remsclient.FormTemplateFieldsOptions, language string) *[]KeyLabelModel {
	if item == nil {
		return nil
	}
	opts := make([]KeyLabelModel, 0, len(*item))
	for _, opt := range *item {
		opts = append(opts, KeyLabelModel{
			Key:   types.StringValue(opt.Key),
			Label: shared.GetLocalizedString(&opt.Label, language),
		})
	}
	return &opts
}

func toKeyLabelColumns(item *[]remsclient.FormTemplateFieldsColumns, language string) *[]KeyLabelModel {
	if item == nil {
		return nil
	}
	opts := make([]KeyLabelModel, 0, len(*item))
	for _, opt := range *item {
		opts = append(opts, KeyLabelModel{
			Key:   types.StringValue(opt.Key),
			Label: shared.GetLocalizedString(&opt.Label, language),
		})
	}
	return &opts
}

func toVisibilityModel(item *remsclient.FormTemplateFieldsVisibility) *VisibilityModel {
	if item == nil {
		return nil
	}

	var fieldId types.String
	if item.VisibilityField != nil {
		fieldId = types.StringValue(item.VisibilityField.FieldID)
	}

	return &VisibilityModel{
		VisibilityType: types.StringValue(string(item.VisibilityType)),
		FieldId:        fieldId,
		HasValue:       item.VisibilityValues,
	}
}

func toFormFieldModel(formItem remsclient.FieldTemplate, language string) FormFieldResourceModel {

	return FormFieldResourceModel{
		Id:          types.StringValue(formItem.FieldID),
		Type:        types.StringValue(string(formItem.FieldType)),
		Title:       shared.GetLocalizedString(&formItem.FieldTitle, language),
		Info:        shared.GetLocalizedString(formItem.FieldInfoText, language),
		Placeholder: shared.GetLocalizedString(formItem.FieldPlaceholder, language),
		Optional:    types.BoolValue(formItem.FieldOptional),
		Options:     toKeyLabelOptions(formItem.FieldOptions, language),
		Columns:     toKeyLabelColumns(formItem.FieldColumns, language),
		MaxLength:   types.Int64PointerValue(formItem.FieldMaxLength),
		Privacy:     types.StringPointerValue((*string)(formItem.FieldPrivacy)),
		Visibility:  toVisibilityModel(formItem.FieldVisibility),
	}
}

func fromKeyLabelOptions(items *[]KeyLabelModel, language string) *[]remsclient.CreateFormCommandFieldsOptions {
	if items == nil {
		return nil
	}
	opts := make([]remsclient.CreateFormCommandFieldsOptions, 0, len(*items))
	for _, item := range *items {
		opts = append(opts, remsclient.CreateFormCommandFieldsOptions{
			Key:   item.Key.ValueString(),
			Label: shared.ToLocalizedStringValue(item.Label, language),
		})
	}
	return &opts
}

func fromKeyLabelColumns(items *[]KeyLabelModel, language string) *[]remsclient.CreateFormCommandFieldsColumns {
	if items == nil {
		return nil
	}
	cols := make([]remsclient.CreateFormCommandFieldsColumns, 0, len(*items))
	for _, item := range *items {
		cols = append(cols, remsclient.CreateFormCommandFieldsColumns{
			Key:   item.Key.ValueString(),
			Label: shared.ToLocalizedStringValue(item.Label, language),
		})
	}
	return &cols
}

func fromVisibilityModel(item *VisibilityModel) *remsclient.CreateFormCommandFieldsVisibility {
	if item == nil {
		return nil
	}

	var visibilityField *remsclient.CreateFormCommandFieldsVisibilityField
	if !item.FieldId.IsNull() && !item.FieldId.IsUnknown() {
		visibilityField = &remsclient.CreateFormCommandFieldsVisibilityField{
			FieldID: item.FieldId.ValueString(),
		}
	}

	return &remsclient.CreateFormCommandFieldsVisibility{
		VisibilityField:  visibilityField,
		VisibilityType:   remsclient.CreateFormCommandFieldsVisibilityVisibilityType(item.VisibilityType.ValueString()),
		VisibilityValues: item.HasValue,
	}
}

func fromFormFieldModels(items []FormFieldResourceModel, language string) []remsclient.NewFieldTemplate {
	fields := make([]remsclient.NewFieldTemplate, 0, len(items))
	for _, item := range items {

		var fieldPrivacy *remsclient.NewFieldTemplateFieldPrivacy
		if !item.Privacy.IsNull() && !item.Privacy.IsUnknown() {
			p := remsclient.NewFieldTemplateFieldPrivacy(item.Privacy.ValueString())
			fieldPrivacy = &p
		}

		fields = append(fields, remsclient.NewFieldTemplate{
			FieldID:          item.Id.ValueStringPointer(),
			FieldType:        remsclient.NewFieldTemplateFieldType(item.Type.ValueString()),
			FieldTitle:       shared.ToLocalizedStringValue(item.Title, language),
			FieldOptional:    item.Optional.ValueBool(),
			FieldPlaceholder: shared.ToLocalizedString(item.Placeholder, language),
			FieldInfoText:    shared.ToLocalizedString(item.Info, language),
			FieldOptions:     fromKeyLabelOptions(item.Options, language),
			FieldColumns:     fromKeyLabelColumns(item.Columns, language),
			FieldMaxLength:   item.MaxLength.ValueInt64Pointer(),
			FieldPrivacy:     fieldPrivacy,
			FieldVisibility:  fromVisibilityModel(item.Visibility),
		})
	}
	return fields
}
