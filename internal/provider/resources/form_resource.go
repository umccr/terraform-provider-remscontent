// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/umccr/terraform-provider-remscontent/internal/remsclient"
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
	Id          types.String `tfsdk:"id"`
	Type        types.String `tfsdk:"type"`
	Title       types.Map    `tfsdk:"title"`
	Info        types.String `tfsdk:"info"`
	Placeholder types.String `tfsdk:"placeholder"`
	Optional    types.Bool   `tfsdk:"optional"`
}

var fieldSchema = schema.NestedAttributeObject{
	Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Optional: true,
		},
		"type": schema.StringAttribute{
			Required: true,
		},
		"title": schema.MapAttribute{
			ElementType: types.StringType,
			Optional:    true,
		},
		"info": schema.StringAttribute{
			Optional: true,
		},
		"placeholder": schema.StringAttribute{
			Optional: true,
		},
		"optional": schema.BoolAttribute{
			Optional: true,
		},
	},
}

// FormResourceModel describes the resource data model.
type FormResourceModel struct {
	Id             types.Int64  `tfsdk:"id"`
	OrganizationId types.String `tfsdk:"organization_id"`
	Title          types.String `tfsdk:"title"`
	Fields         types.List   `tfsdk:"fields"`
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
				MarkdownDescription: "The ID of the organization to associate this license with.",
				Required:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Form title attribute",
				Required:            true,
			},
			"fields": schema.ListNestedAttribute{
				NestedObject: fieldSchema,
				Required:     true,
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
		Organization: remsclient.OrganizationId{
			OrganizationId: plan.OrganizationId.ValueString(),
		},
		FormTitle: plan.Title.ValueString(),
	}

	// Convert our resource model map with error checking
	modelFields := make([]FormFieldResourceModel, len(resourceModel.Fields.Elements()))
	modelFieldDiagnostics := resourceModel.Fields.ElementsAs(ctx, &modelFields, false)

	if len(modelFieldDiagnostics) > 0 {
		resp.Diagnostics.Append(modelFieldDiagnostics...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// construct the API input models in reverse order (the generated openapi Go client is a bit wierd)
	orgId := remsclient.NewOrganizationId(resourceModel.OrganizationId.ValueString())

	formConfig := remsclient.NewCreateFormCommandWithDefaults()
	formConfig.SetOrganization(*orgId)

	// convert resource model data into API model
	if resourceModel.Title.IsNull() {
		formConfig.SetFormTitleNil()
	} else {
		formConfig.SetFormTitle(resourceModel.Title.ValueString())
	}

	newFields := make([]remsclient.NewwFieldTemplate, 0)

	for _, modelFieldValue := range modelFields {

		if !modelFieldValue.Title.IsNull() && !modelFieldValue.Title.IsUnknown() {
			var titleMap map[string]string
			resp.Diagnostics.Append(modelFieldValue.Title.ElementsAs(ctx, &titleMap, false)...)
			if resp.Diagnostics.HasError() {
				return
			}

			newField := remsclient.NewNewFieldTemplate(
				titleMap,
				modelFieldValue.Type.ValueString(),
				modelFieldValue.Optional.ValueBool())

			if !modelFieldValue.Id.IsNull() {
				newField.SetFieldId(modelFieldValue.Id.ValueString())
			}

			newFields = append(newFields, *newField)
		}
	}

	formConfig.SetFormFields(newFields)

	createResult, createResponse, createErr := r.client.FormsAPI.
		ApiFormsCreatePost(context.Background()).
		CreateFormCommand(*formConfig).
		Execute()

	if createErr != nil {
		resp.Diagnostics.AddError(
			"Failure to create form",
			fmt.Sprintf("Could not create form: %s %v", createErr.Error(), createResponse),
		)
		return
	}

	if !createResult.Success {
		resp.Diagnostics.AddError(
			"Failure to create form",
			fmt.Sprintf("Could not create form: %v", createResult.GetErrors()),
		)
		return
	}

	tflog.Info(ctx, createResponse.Status)

	resourceModel.Id = types.Int64Value(createResult.GetId())

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save resourceModel into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &resourceModel)...)
}

func (r *FormResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FormResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FormResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FormResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FormResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FormResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }
}

func (r *FormResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

/*
{
    "archived": false,
    "organization": {
        "organization/id": "Garvan Institute of Medical Research",
        "organization/short-name": {
            "en": "Garvan"
        },
        "organization/name": {
            "en": "Garvan Institute of Medical Research"
        }
    },
    "enabled": true
    "form/id": 7,
    "form/title": "MGRB Form v2.0",
    "form/errors": null,
    "form/external-title": {
        "en": "MGRB Form v2.0"
    },
    "form/internal-name": "MGRB Form v2.0",
    "form/fields": [
        {
            "field/title": {
                "en": "SECTION 1 - BACKGROUND"
            },
            "field/type": "header",
            "field/id": "fld1",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "1.1 The Medical Genome Reference Bank"
            },
            "field/type": "header",
            "field/id": "fld2",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "The Medical Genome Reference Bank (MGRB) is compromised of whole genome sequencing (WGS) data and phenotypic information from 4010 healthy Australians over 70 years of age. MGRB participants, consented through contributing studies, 45 and Up (Sax Institute, Sydney), and the ASPirin in Reducing Events in the Elderly (ASPREE) clinical trial (Monash University, Melbourne), are free from cardiovascular disease, degenerative neurological disorders and of a history of cancer at the time of consent into the study. The dataset can function as a powerful filter to distinguish between causal genetic and population-based genetic variation and be a resource to maximise the efficiency of genomic discovery in both the research and clinical setting.\nWGS was performed on the Illumina HiSeq X-Ten platform at the Garvan Institute of Medical Research (Sydney, Australia) under clinically accredited conditions (ISO 15189). Data was aligned and variant files were generated using best practice (BWA, GATK) pipelines to make results comparable with other cohorts. Joint-called variants were loaded into the Sydney Genomics Collaborative data portal built on the Garvan Institute’s Vectis platform. Key features of Vectis (a lever in Latin) include the support of both summary statistics (Summary and Explore tabs) as well as a scale-out Variant Store that supports the filtering of patients based on genomic features (genes and chromosome coordinates) as well as clinical information. Access to the Clinical Filtering features of Vectis are restricted to users whose Data Access Applications have been approved. Researchers are invited to complete this Data Access Application to gain access to comprehensive genotypic and clinical information, to support high-level integrative analysis.\nInformation on the comprehensive clinical data that is available through ASPREE and 45 and UP (pending a successful application) can be found in the following documents: ASPREE ASPREE Protocol AUS Version# 9 Nov 2014 45 and Up Download the Baseline Questionnaire for Women (PDF 537KB) Download the Baseline Questionnaire for Men (PDF 537KB)"
            },
            "field/type": "label",
            "field/id": "fld3",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "1.2 Data Access Policy"
            },
            "field/type": "header",
            "field/id": "fld4",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "To maintain participant privacy and confidentiality, whilst maximising MGRB utility, we have deployed a tiered data management system that determines the depth of data that is made available to researchers (as summarised in the schematic below). This consists of 3 access tiers; Open access, Controlled access and Restricted access. Completion of the MGRB DAA (Section 2), and execution of the Data Transfer Agreement (Sections 3, 4 & 5) is required before access to Controlled and Restricted access data will be granted. The MGRB Data Access Committee (DAC) will review completed applications within 6 weeks. Upon approval, Controlled access can then be immediately made available to the applicant. Should Restricted access data be requested, the application will be immediately forwarded to the governing body of the participating study, and will be considered in parallel with the MGRB DAC review. The study governing body may insist on an independent application process, the time-line for which dependent solely on the governing body – more information can be found here for ASPREE and 45 and UP. A copy of the full MGRB Data Access Policy can be found here."
            },
            "field/type": "label",
            "field/id": "fld5",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "1.3 Data Access Policy Schematic"
            },
            "field/type": "header",
            "field/id": "fld6",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "https://sgc.garvan.org.au/_files/data_revised_open_tier.png"
            },
            "field/type": "label",
            "field/id": "fld7",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "1.4 Data Access"
            },
            "field/type": "header",
            "field/id": "fld8",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "All approved users will be granted immediate access to the MGRB on the Sydney Genomics Collaborative Vectis Platform. Applicants wishing to access data files can access these from the European Genome-phenome Archive (EGA) as well as NCI-Australia. The EGA resource is intended for approved MGRB users who wish to download the data as lossless CRAM files (approximately 150 Terabytes). Access to the MGRB at NCI -Australia is intended for users of this Australian Teir 1 supercomputing facility but does not support the downloading of data."
            },
            "field/type": "label",
            "field/id": "fld9",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Please indicate where you would like to access the MGRB data"
            },
            "field/type": "multiselect",
            "field/id": "fld10",
            "field/options": [
                {
                    "key": "nci",
                    "label": {
                        "en": "NCI - Australia"
                    }
                },
                {
                    "key": "ega",
                    "label": {
                        "en": "EGA - https://ega-archive.org/studies/EGAS00001003511"
                    }
                }
            ],
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Associated phenotype data are also available: pyega3 -c 1000 -cf ~/.ega2 -d fetch EGAF00002440978"
            },
            "field/type": "label",
            "field/id": "fld11",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Once downloaded, minimum protection measures are required to protect all MGRB data"
            },
            "field/type": "label",
            "field/id": "fld12",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "SECTION 2 -DATA ACCESS APPLICATION"
            },
            "field/type": "header",
            "field/id": "fld13",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "2.1 Project Overview"
            },
            "field/type": "header",
            "field/id": "fld14",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Level of Data Required"
            },
            "field/type": "multiselect",
            "field/id": "fld15",
            "field/options": [
                {
                    "key": "controlled",
                    "label": {
                        "en": "Controlled"
                    }
                },
                {
                    "key": "restricted",
                    "label": {
                        "en": "Restricted"
                    }
                }
            ],
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Provide a brief overview of the proposed project; with specific emphasis on what level of information is being reuqested and how MGRB data will be used (500 words maximum)"
            },
            "field/type": "texta",
            "field/id": "fld16",
            "field/max-length": null,
            "field/optional": false
        },
        {
            "field/title": {
                "en": "2.2 Genomic Data"
            },
            "field/type": "header",
            "field/id": "fld17",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Select the MGRB data format required for the proposed project (tick both if required)"
            },
            "field/type": "multiselect",
            "field/id": "fld18",
            "field/options": [
                {
                    "key": "45nup",
                    "label": {
                        "en": "45 and Up"
                    }
                },
                {
                    "key": "aspree",
                    "label": {
                        "en": "ASPREE"
                    }
                }
            ],
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Provide detail as to how MGRB data will be stored and what security measures are in place to ensure conformity with the conditions stipulated in Section 1.4"
            },
            "field/type": "texta",
            "field/id": "fld19",
            "field/max-length": null,
            "field/optional": false
        },
        {
            "field/title": {
                "en": "2.3 Clinical Information"
            },
            "field/type": "header",
            "field/id": "fld20",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Clinical Data (Controlled Access) for 45 and Up and ASPREE are listed below - select the data required for the proposed project. (Should additional clinical information be required from either study, please state what data are required and justify the release of this data - release of this information is at the discretion of the external governing board for the study in question and maybe subject to an independent proposal)"
            },
            "field/type": "label",
            "field/id": "fld21",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "45 and Up"
            },
            "field/type": "multiselect",
            "field/id": "fld22",
            "field/options": [
                {
                    "key": "sex",
                    "label": {
                        "en": "Sex"
                    }
                },
                {
                    "key": "yob",
                    "label": {
                        "en": "Year of Birth"
                    }
                },
                {
                    "key": "height",
                    "label": {
                        "en": "Height"
                    }
                },
                {
                    "key": "weight",
                    "label": {
                        "en": "Weight"
                    }
                }
            ],
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Additional clinical information required from 45 and Up and justification thereof;"
            },
            "field/type": "texta",
            "field/id": "fld23",
            "field/max-length": null,
            "field/optional": true
        },
        {
            "field/title": {
                "en": "ASPREE"
            },
            "field/type": "multiselect",
            "field/id": "fld24",
            "field/options": [
                {
                    "key": "sex",
                    "label": {
                        "en": "Sex"
                    }
                },
                {
                    "key": "yob",
                    "label": {
                        "en": "Year of Birth"
                    }
                },
                {
                    "key": "height",
                    "label": {
                        "en": "Height"
                    }
                },
                {
                    "key": "weight",
                    "label": {
                        "en": "Weight"
                    }
                },
                {
                    "key": "systolic-blood-pressure",
                    "label": {
                        "en": "Systolic Blood Pressure"
                    }
                },
                {
                    "key": "evidence-of-macular-degeneration",
                    "label": {
                        "en": "Evidence of Macular Degeneration"
                    }
                },
                {
                    "key": "resting-glucose",
                    "label": {
                        "en": "Resting Glucose"
                    }
                },
                {
                    "key": "abdominal-circumference",
                    "label": {
                        "en": "Abdominal Circumference"
                    }
                }
            ],
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Additional clinical information required from ASPREE and justification thereof;"
            },
            "field/type": "texta",
            "field/id": "fld25",
            "field/max-length": null,
            "field/optional": true
        },
        {
            "field/title": {
                "en": "SECTION 3  - DATA TRANSFER AGREEMENT TERMS AND CONDITIONS"
            },
            "field/type": "header",
            "field/id": "fld26",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "The MGRB Research Governance Committee (RGC) has been established by the three organisations that generated and collected the clinical and genomic data that comprises the MGRB: the Garvan Institute of Medical Research, Monash University and the Sax Institute. The RGC has delegated responsibility for assessing applications to access data to the MGRB Data Access Committee (DAC). Garvan has been delegated by the DAC to sign the Data Transfer Agreement on behalf of the MGRB Partners. Garvan cannot amend the terms and conditions of this agreement."
            },
            "field/type": "label",
            "field/id": "fld27",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "The User Institution (named in Section 5.1) agrees with the Garvan Institute of Medical Research of 384 Victoria St, Darlinghurst, New South Wales, 2010 Australia ABN 62 330 391 937 (Garvan), that access to and use of the MGRB dataset specified in Sections 2.2 and 2.3 will be governed by the following terms and conditions."
            },
            "field/type": "label",
            "field/id": "fld28",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "The User Institution agrees to be bound by these terms and conditions."
            },
            "field/type": "label",
            "field/id": "fld29",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "3.1 Definitions"
            },
            "field/type": "header",
            "field/id": "fld30",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Authorised Personnel means the individuals at the User Institution to whom the MGRB Data Access Committee grants access to the Data including the User listed in Section 5.2, the individuals listed in Section 5.3 and any other individuals for whom the User Institution subsequently requests access to the Data."
            },
            "field/type": "label",
            "field/id": "fld31",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Data means the managed access datasets to which the User Institution has requested access as set out in Sections 2.2 and 2.3."
            },
            "field/type": "label",
            "field/id": "fld32",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "External Collaborator means a collaborator of the User, working for an institution other than the User Institution."
            },
            "field/type": "label",
            "field/id": "fld33",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "MGRB Partners means the Garvan Institute of Medical Research, the Sax Institute and Monash University."
            },
            "field/type": "label",
            "field/id": "fld34",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "MGRB Data means the open and controlled access genomic and clinical datasets generated by the MGRB Partners."
            },
            "field/type": "label",
            "field/id": "fld35",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Project means the project for which the User Institution has requested access to these Data as set out in Section 2.1."
            },
            "field/type": "label",
            "field/id": "fld36",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Publication means without limitation, articles published in print journals, electronic journals, reviews, books, posters and other written and oral presentations of research."
            },
            "field/type": "label",
            "field/id": "fld37",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Recipient means the User Institution listed in Section 5.1."
            },
            "field/type": "label",
            "field/id": "fld38",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Research Participant means an individual whose data form part of the Data."
            },
            "field/type": "label",
            "field/id": "fld39",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Research Purpose means an academic research not for commercial application that is seeking to advance the understanding of genetics and genomics, including the treatment of disorders, and work on statistical methods that may be applied to such research."
            },
            "field/type": "label",
            "field/id": "fld40",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "User means the Principal Investigator for the Project listed in Section 5.2."
            },
            "field/type": "label",
            "field/id": "fld41",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "3.2 The User Institution agrees to only use the Data for the purpose of the Project and only for Research Purposes. The User Institution provides a warranty that the Project for which MGRB data are being sought, has been approved by a human research ethics committee."
            },
            "field/type": "label",
            "field/id": "fld42",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "3.3 The User Institution agrees to preserve, at all times, the confidentiality of the Data. In particular, it undertakes not to use, or attempt to use the Data to compromise or otherwise infringe the confidentiality of information on Research Participants. Without prejudice to the generality of the foregoing, the User Institution agrees to use at least the measures set out in Section 1.4 to protect the Data."
            },
            "field/type": "label",
            "field/id": "fld43",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "3.4 The User Institution agrees to protect the confidentiality of Research Participants in any research paper or publication that is prepared using the Data by taking all reasonable care to limit the possibility of identification."
            },
            "field/type": "label",
            "field/id": "fld44",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "3.5 The User Institution agrees not to link or combine the Data to other information or archived data available in a way that could re-identify the Research Participants, even if access to that data has been formally granted to the User Institution or is freely available without restriction."
            },
            "field/type": "label",
            "field/id": "fld45",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "3.6 The User Institution agrees only to transfer or disclose the Data, in whole or part, or any material derived from the Data, to the User and Authorised Personnel (as defined in Section 5). Should the User Institution wish to share the Data with an External Collaborator, the External Collaborator must complete a separate application for access to the Data.\n3.7 The User Institution agrees that the MGRB Partners, and all other parties involved in the creation, funding or protection of the Data: a) make no warranty or representation, express or implied as to the accuracy, quality or comprehensiveness of the Data; b) bear no responsibility for the unavailability of, or break in access to, the Data for whatever reason; and c) bear no responsibility for the further analysis or interpretation of these Data. Except to the extent prohibited by law, the User institution assumes all risk and liability for all damages, losses, expenses (including reasonable legal expenses), claims, demands, suits or other liability (Loss) arising from or in relation to the User’s use, storage or disposal of the Data. The MGRB Partners will not be liable to the User Institution for any Loss incurred by the User or User Institution, or made against the User or User Institution by any other party, due to or arising from the use, storage or disposal of the Data by the User or User Institution, except to the extent permitted by law when caused by the gross negligence or willful misconduct of the MGRB Partners.\n3.8 The User Institution agrees to follow the Fort Lauderdale Guidelines and the Toronto Statement. This includes but is not limited to recognising the contribution of the MGRB Partners and including a proper acknowledgement in all reports or publications resulting from the use of these Data.\n3.9 The User Institution agrees to follow the Publication Policy described in Section 4.\n3.10 None of the MGRB Partners will have any right, title or interest in any intellectual property (IP) that is developed, created or invented in the course of the Project unless one or more of the MGRB Partners has contributed to its development creation or invention. If that is the case the IP will be jointly owned by the contributing parties as tenants in common in shares proportionate to their respective intellectual contribution to the development creation or invention of that IP . The joint owners will consult and decide on measures to be taken to protect and exploit that IP .\n3.11 The User Institution agrees not to make intellectual property claims on the Data and not to use intellectual property protection in ways that would prevent or block access to, or use of, any element of the Data, or conclusions drawn directly from the Data or any of it.\n3.12 The User Institution must notify the MGRB Data Access Committee of any patent application filed in respect of the Project, at the time of filing and provide the MGRB Data Access Committee (or its nominee) with sufficient information to determine whether there has been a breach of clause 3.11. If there has been a breach of clause 3.11 then the User Institution must amend its patent application to eliminate the breach or withdraw it. The information provided to MGRB Data Access Committee pursuant to this clause will be treated as confidential to the User Institution.\n3.13 The User Institution agrees to destroy/discard the Data held, once it is no longer required for use in the Project, unless obliged to retain the Data for archival purposes in conformity with audit or legal requirements. The User will not be required to delete or destroy Data that resides on electronic back-up tapes or other electronic back-up files that have been created solely by the User Institution’s automatic or routine archiving and back-up procedures but agrees to preserve the confidentiality of such Data.\n3.14 The User Institution will notify the MGRB Data Access Committee within 30 days of any changes or departures of Authorised Personnel.\n3.15 The User Institution will notify the MGRB Data Access Committee as soon as it becomes aware of a breach of the terms or conditions of this agreement.\n3.16 The MGRB Data Access Committee may terminate this agreement by written notice to the User Institution. If this agreement terminates for any reason, the User Institution will be required to destroy any Data held, including copies. This clause does not prevent the User Institution from retaining the Data for archival purposes in conformity with audit or legal requirements. The User will not be required to delete or destroy Data that resides on electronic back-up tapes or other electronic back-up files that have been created solely by the User Institution’s automatic or routine archiving and back-up procedures but agrees to preserve the confidentiality of such Data.\n3.17 The User Institution accepts that it may be necessary for the MGRB Data Access Committee to alter the terms of this agreement from time to time. In the event that changes are required, the MGRB Data Access Committee or their appointed agent will contact the User Institution to inform it of the changes and the User Institution may elect to accept the changes or terminate the agreement by written notice to the other party to this agreement.\n3.18 If requested, the User Institution will allow data security and management documentation to be inspected by an agent of the other party to verify that it is complying with the terms of this agreement.\n3.19 The User Institution agrees to distribute a copy of these terms to the Authorised Personnel. The User Institution will procure that the Authorised Personnel comply with the terms of this agreement.\n3.20 The parties acknowledge that this agreement does not address the law that governs disputes arising out of this agreement or the subject matter of this Agreement."
            },
            "field/type": "label",
            "field/id": "fld46",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "SECTION 4 - PUBLICATION POLICY"
            },
            "field/type": "header",
            "field/id": "fld47",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "The User Institution is required to include an appropriate acknowledgement (and authorship if deemed necessary) in any Publication that makes use of or reference to an MGRB dataset in accordance with the following terms. Manuscripts or presentations may be submitted to MGRB@garvan.org.au for review."
            },
            "field/type": "label",
            "field/id": "fld48",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "4.1 Open Access and/or Controlled Access data"
            },
            "field/type": "header",
            "field/id": "fld49",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Publications that make use of Open Access data will be required to acknowledge the MGRB Partners in the acknowledgements section of the Publication. Should the User Institution wish to add any personnel from an MGRB Partner to the list of authors, the User Institution agrees to forward the Publication to the MGRB Data Access Committee for approval prior to submission. Manuscripts or presentations may be submitted to MGRB@garvan.org.au for review. Authors are also encouraged to recognise the contribution of the appropriate cohort convenors via the acknowledgements section in their Publication. An example of a proper attribution is: \"The results <published or shown> here are in whole or part based upon data generated by the MGRB Partners: https://sgc.garvan.org.au/initiatives/mgrb"
            },
            "field/type": "label",
            "field/id": "fld50",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "4.2 Restriction Access (or Tier 3) data"
            },
            "field/type": "header",
            "field/id": "fld51",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "When a User Institution applies to use Restricted Access data the arrangements for publication review, acknowledgement and/or authorship will be determined by the entity in the MGBR Partners that owns the relevant study. The MGRB Data Access Committee must be notified of any Publication arising from use of or containing Restricted Access data at least one month prior to submission (or presentation)."
            },
            "field/type": "label",
            "field/id": "fld52",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "SECTION 5 - USER INFORMATION"
            },
            "field/type": "header",
            "field/id": "fld53",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "5.1 USER INSTITUTION (\"Recipient\")"
            },
            "field/type": "header",
            "field/id": "fld54",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Institute Name:"
            },
            "field/type": "text",
            "field/id": "fld55",
            "field/max-length": null,
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Address:"
            },
            "field/type": "text",
            "field/id": "fld56",
            "field/max-length": null,
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Phone:"
            },
            "field/type": "phone-number",
            "field/id": "fld57",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "5.2 PRINCIPAL INVESTIGATOR (\"User\")"
            },
            "field/type": "header",
            "field/id": "fld58",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Name:"
            },
            "field/type": "text",
            "field/id": "fld59",
            "field/max-length": null,
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Address:"
            },
            "field/type": "text",
            "field/id": "fld60",
            "field/max-length": null,
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Phone:"
            },
            "field/type": "phone-number",
            "field/id": "fld61",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Email:"
            },
            "field/type": "email",
            "field/id": "fld62",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "5.3 AUTHORISED PERSONNEL"
            },
            "field/type": "header",
            "field/id": "fld63",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "(Personnel at the User Institution to be granted access to the Data)"
            },
            "field/type": "table",
            "field/id": "fld64",
            "field/optional": false,
            "field/columns": [
                {
                    "key": "name",
                    "label": {
                        "en": "Name"
                    }
                },
                {
                    "key": "position",
                    "label": {
                        "en": "Position/Job Title"
                    }
                },
                {
                    "key": "department",
                    "label": {
                        "en": "Department/Division"
                    }
                },
                {
                    "key": "contact",
                    "label": {
                        "en": "Email/Phone"
                    }
                },
                {
                    "key": "nci-user",
                    "label": {
                        "en": "NCI-Australia Username (for applicants accessing the MGRB at NCI-Australia)"
                    }
                }
            ]
        },
        {
            "field/title": {
                "en": "Executed as an Agreement"
            },
            "field/type": "label",
            "field/id": "fld65",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Both parties confirm that they have read, understood and accept the terms and conditions outlined in this Application."
            },
            "field/type": "text",
            "field/id": "fld66",
            "field/max-length": null,
            "field/optional": false
        },
        {
            "field/title": {
                "en": "SIGNED on behalf of the USER INSTITUTION, by its duly authorised officer, in the presence of:"
            },
            "field/info-text": {
                "en": "Upload signed page ?"
            },
            "field/type": "attachment",
            "field/id": "fld67",
            "field/optional": false
        },
        {
            "field/title": {
                "en": "Electronically SIGNED and DATED on behalf of THE MGRB DATA ACCESS COMMITTEE which authorises access to the MGRB cohort by the Principal Investigator and named Authorised Personnel of this application. Authority to use this cohort is effective as of the date electronically stamped on each page of this document."
            },
            "field/type": "label",
            "field/id": "fld68",
            "field/optional": false
        }
    ],
}
*/
