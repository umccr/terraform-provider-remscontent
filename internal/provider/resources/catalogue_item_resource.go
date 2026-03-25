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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/rems-client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &CatalogueItemResource{}
var _ resource.ResourceWithImportState = &CatalogueItemResource{}
var _ resource.ResourceWithConfigure = &CatalogueItemResource{}

func NewCatalogueItemResource() resource.Resource {
	return &CatalogueItemResource{}
}

// CatalogueItemResource defines the resource implementation.
type CatalogueItemResource struct {
	BaseRemsResource
}

// CatalogueItemLocalizationModel represents a single language localization.
type CatalogueItemLocalizationModel struct {
	Title   types.String `tfsdk:"title"`
	Infourl types.String `tfsdk:"infourl"`
}

// CatalogueItemResourceModel describes the resource data model.
type CatalogueItemResourceModel struct {
	Id             types.Int64                     `tfsdk:"id"`
	OrganizationId types.String                    `tfsdk:"organization_id"`
	ResourceId     types.Int64                     `tfsdk:"resource_id"`
	WorkflowId     types.Int64                     `tfsdk:"workflow_id"`
	FormId         types.Int64                     `tfsdk:"form_id"`
	Localizations  *CatalogueItemLocalizationModel `tfsdk:"localizations"`
	Enabled        types.Bool                      `tfsdk:"enabled"`
	Archived       types.Bool                      `tfsdk:"archived"`
	CategoryIds    *[]types.Int64                  `tfsdk:"categories"`
}

func (r *CatalogueItemResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catalogue_item"
}

func (r *CatalogueItemResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a REMS catalogue item, which links a resource, workflow, and optional form into an item users can apply for.",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Catalogue item internal identifier assigned by REMS.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the organization this catalogue item belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "The internal ID of the REMS resource to link.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"workflow_id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "The internal ID of the workflow to use.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"form_id": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "The internal ID of the form to use. If omitted, no form is required.",
				// PlanModifiers: []planmodifier.Int64{
				// 	int64planmodifier.RequiresReplace(),
				// },
			},
			"localizations": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Localization for the catalogue item (English).",
				Attributes: map[string]schema.Attribute{
					"title": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "The display title of the catalogue item.",
					},
					"infourl": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "An optional URL with more information about this catalogue item.",
					},
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether this catalogue item is active. Defaults to `true`.",
			},
			"archived": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether this catalogue item is archived. Defaults to `false`.",
			},
			"categories": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "List of categories.",
			},
		},
	}
}

func (r *CatalogueItemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CatalogueItemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var categoriesCmd *[]remsclient.CategoryID
	if plan.CategoryIds != nil {
		s := make([]remsclient.CategoryID, len(*plan.CategoryIds))
		for i, v := range *plan.CategoryIds {
			s[i] = remsclient.CategoryID{CategoryID: v.ValueInt64()}
		}
		categoriesCmd = &s
	}

	createCmd := remsclient.CreateCatalogueItemCommand{
		Organization: remsclient.OrganizationID{
			OrganizationID: plan.OrganizationId.ValueString(),
		},
		Resid: plan.ResourceId.ValueInt64(),
		Wfid:  plan.WorkflowId.ValueInt64(),
		Localizations: remsclient.WriteCatalogueItemLocalizations{
			"en": remsclient.CatalogueItemLocalization{
				Infourl: plan.Localizations.Infourl.ValueStringPointer(),
				Title:   plan.Localizations.Title.ValueString(),
			},
		},
		Enabled:    plan.Enabled.ValueBoolPointer(),
		Archived:   plan.Archived.ValueBoolPointer(),
		Form:       plan.FormId.ValueInt64Pointer(),
		Categories: categoriesCmd,
	}

	createResp, createErr := r.client.PostAPICatalogueItemsCreateWithResponse(ctx, nil, createCmd)
	if createErr != nil {
		resp.Diagnostics.AddError("Error Creating Catalogue Item", createErr.Error())
		return
	}
	if createResp.StatusCode() != 200 || createResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Creating Catalogue Item", fmt.Sprintf("status: %d, body: %s", createResp.StatusCode(), string(createResp.Body)))
		return
	}
	if createResp.JSON200.ID == nil {
		resp.Diagnostics.AddError("Error Creating Catalogue Item", "API returned no ID")
		return
	}
	plan.Id = types.Int64Value(*createResp.JSON200.ID)

	tflog.Trace(ctx, "created a catalogue item")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CatalogueItemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CatalogueItemResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readResp, readErr := r.client.GetAPICatalogueItemsItemIDWithResponse(ctx, state.Id.ValueInt64(), nil)
	if readErr != nil {
		resp.Diagnostics.AddError("Error Reading Catalogue Item", readErr.Error())
		return
	}
	if readResp.StatusCode() == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if readResp.StatusCode() != 200 || readResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Reading Catalogue Item", fmt.Sprintf("status: %d, body: %s", readResp.StatusCode(), string(readResp.Body)))
		return
	}
	item := readResp.JSON200
	state.OrganizationId = types.StringValue(item.Organization.OrganizationID)
	state.ResourceId = types.Int64Value(item.ResourceID)
	state.WorkflowId = types.Int64Value(item.Wfid)
	state.Enabled = types.BoolValue(item.Enabled)
	state.Archived = types.BoolValue(item.Archived)
	if item.Formid != nil {
		state.FormId = types.Int64Value(*item.Formid)
	} else {
		state.FormId = types.Int64Null()
	}

	state.Localizations = &CatalogueItemLocalizationModel{
		Title:   types.StringValue(item.Localizations[r.language].Title),
		Infourl: types.StringPointerValue(item.Localizations[r.language].Infourl),
	}

	state.CategoryIds = nil

	if item.Categories != nil && len(*item.Categories) > 0 {
		ids := make([]types.Int64, len(*item.Categories))
		for i, c := range *item.Categories {
			ids[i] = types.Int64Value(c.CategoryID)
		}
		state.CategoryIds = &ids
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CatalogueItemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CatalogueItemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var categoriesCmd *[]remsclient.CategoryID
	if plan.CategoryIds != nil {
		s := make([]remsclient.CategoryID, len(*plan.CategoryIds))
		for i, v := range *plan.CategoryIds {
			s[i] = remsclient.CategoryID{CategoryID: v.ValueInt64()}
		}
		categoriesCmd = &s
	}
	editCmd := remsclient.EditCatalogueItemCommand{
		ID: plan.Id.ValueInt64(),
		Organization: &remsclient.OrganizationID{
			OrganizationID: plan.OrganizationId.ValueString(),
		},
		Localizations: remsclient.WriteCatalogueItemLocalizations{
			"en": remsclient.CatalogueItemLocalization{
				Infourl: plan.Localizations.Infourl.ValueStringPointer(),
				Title:   plan.Localizations.Title.ValueString(),
			},
		},
		Categories: categoriesCmd,
	}

	editResp, editErr := r.client.PutAPICatalogueItemsEditWithResponse(ctx, nil, editCmd)
	if editErr != nil {
		resp.Diagnostics.AddError("Error Updating Catalogue Item", editErr.Error())
		return
	}
	if editResp.StatusCode() != 200 || editResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Updating Catalogue Item", fmt.Sprintf("status: %d, body: %s", editResp.StatusCode(), string(editResp.Body)))
		return
	}

	// Update enabled state
	enabledResp, enabledErr := r.client.PutAPICatalogueItemsEnabledWithResponse(ctx, nil, remsclient.EnabledCommand{
		ID:      plan.Id.ValueInt64(),
		Enabled: plan.Enabled.ValueBool(),
	})
	if enabledErr != nil {
		resp.Diagnostics.AddError("Error Setting Catalogue Item Enabled State", enabledErr.Error())
		return
	}
	if enabledResp.StatusCode() != 200 || enabledResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Setting Catalogue Item Enabled State", fmt.Sprintf("status: %d, body: %s", enabledResp.StatusCode(), string(enabledResp.Body)))
		return
	}

	// Update archived state
	archivedResp, archivedErr := r.client.PutAPICatalogueItemsArchivedWithResponse(ctx, nil, remsclient.ArchivedCommand{
		ID:       plan.Id.ValueInt64(),
		Archived: plan.Archived.ValueBool(),
	})
	if archivedErr != nil {
		resp.Diagnostics.AddError("Error Setting Catalogue Item Archived State", archivedErr.Error())
		return
	}
	if archivedResp.StatusCode() != 200 || archivedResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Setting Catalogue Item Archived State", fmt.Sprintf("status: %d, body: %s", archivedResp.StatusCode(), string(archivedResp.Body)))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CatalogueItemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CatalogueItemResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	archivedResp, archivedErr := r.client.PutAPICatalogueItemsArchivedWithResponse(ctx, nil, remsclient.ArchivedCommand{
		ID:       state.Id.ValueInt64(),
		Archived: true,
	})
	if archivedErr != nil {
		resp.Diagnostics.AddError("Error Archiving Catalogue Item", archivedErr.Error())
		return
	}
	if archivedResp.StatusCode() != 200 || archivedResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error Archiving Catalogue Item", fmt.Sprintf("status: %d, body: %s", archivedResp.StatusCode(), string(archivedResp.Body)))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Archived catalogue item with ID: %d", state.Id.ValueInt64()))
}

func (r *CatalogueItemResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idInt, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric ID, got: %s", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idInt)...)
}
