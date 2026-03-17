// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/umccr/terraform-provider-remscontent/internal/provider/data_sources"
	"github.com/umccr/terraform-provider-remscontent/internal/provider/resources"
	"github.com/umccr/terraform-provider-remscontent/internal/provider/shared"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/rems-client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure RemsContentProvider satisfies various provider interfaces.
var _ provider.Provider = &RemsContentProvider{}
var _ provider.ProviderWithFunctions = &RemsContentProvider{}
var _ provider.ProviderWithEphemeralResources = &RemsContentProvider{}

// RemsContentProvider defines the provider implementation.
type RemsContentProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
	// For mocks in tests
	clientOverride *remsclient.ClientWithResponses
}

// contentTypeTransport strips charset from Content-Type headers
// e.g. "application/json; charset=utf-8" -> "application/json"
type contentTypeTransport struct {
	wrapped http.RoundTripper
}

func (t *contentTypeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.wrapped.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, ";") {
		resp.Header.Set("Content-Type", strings.TrimSpace(strings.Split(ct, ";")[0]))
	}
	return resp, nil
}

// RemsContentProviderModel describes the provider data model.
type RemsContentProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	ApiUser  types.String `tfsdk:"api_user"`
	ApiKey   types.String `tfsdk:"api_key"`
	Language types.String `tfsdk:"language"`
}

func (p *RemsContentProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "remscontent"
	resp.Version = p.version
}

func (p *RemsContentProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "REMS instance endpoint (DNS name only, not URI)",
				Optional:            true,
			},
			"api_user": schema.StringAttribute{
				MarkdownDescription: "REMS API user",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "REMS API key",
				Optional:            true,
				Sensitive:           true,
			},
			"language": schema.StringAttribute{
				MarkdownDescription: "Localization language key used for localized strings (e.g. \"en\", \"fi\"). Defaults to \"en\". Can also be set via REMS_LANGUAGE environment variable.",
				Optional:            true,
			},
		},
	}
}

func (p *RemsContentProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	if p.clientOverride != nil {
		// To allow mock responses when testing
		cfg := &shared.ProviderConfig{Client: p.clientOverride, Language: "en"}
		resp.ResourceData = cfg
		resp.DataSourceData = cfg
		return
	}

	var config RemsContentProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if config.Endpoint.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Unknown REMS API Endpoint",
			"The provider cannot create the REMS API client as there is an unknown configuration value for the REMS API endpoint. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the REMS_ENDPOINT environment variable.",
		)
	}

	if config.ApiUser.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_user"),
			"Unknown REMS API User",
			"The provider cannot create the REMS API client as there is an unknown configuration value for the REMS API user. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the REMS_API_USER environment variable.",
		)
	}

	if config.ApiKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown REMS API Key",
			"The provider cannot create the REMS API client as there is an unknown configuration value for the REMS API key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the REMS_API_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.
	endpoint := os.Getenv("REMS_ENDPOINT")
	api_user := os.Getenv("REMS_API_USER")
	api_key := os.Getenv("REMS_API_KEY")
	language := os.Getenv("REMS_LANGUAGE")
	if language == "" {
		language = "en"
	}

	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}

	if !config.ApiUser.IsNull() {
		api_user = config.ApiUser.ValueString()
	}

	if !config.ApiKey.IsNull() {
		api_key = config.ApiKey.ValueString()
	}

	if !config.Language.IsNull() {
		language = config.Language.ValueString()
	}

	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Missing REMS API Endpoint",
			"The provider cannot create the REMS API client without an endpoint. "+
				"Set the endpoint value in the provider configuration or use the REMS_ENDPOINT environment variable.",
		)
	}

	if api_user == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_user"),
			"Missing REMS API User",
			"The provider cannot create the REMS API client without an API user. "+
				"Set the api_user value in the provider configuration or use the REMS_API_USER environment variable.",
		)
	}

	if api_key == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing REMS API Key",
			"The provider cannot create the REMS API client without an API key. "+
				"Set the api_key value in the provider configuration or use the REMS_API_KEY environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := remsclient.NewClientWithResponses(
		fmt.Sprintf("https://%s", endpoint),
		remsclient.WithHTTPClient(&http.Client{
			Transport: &contentTypeTransport{
				wrapped: http.DefaultTransport,
			},
		}),
		remsclient.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("x-rems-api-key", api_key)
			req.Header.Set("x-rems-user-id", api_user)
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Content-Type", "application/json")
			return nil
		}),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create REMS API Client",
			"An unexpected error occurred when creating the REMS API client: "+err.Error(),
		)
		return
	}

	cfg := &shared.ProviderConfig{
		Client:   client,
		Language: language,
	}
	resp.DataSourceData = cfg
	resp.ResourceData = cfg
}

func (p *RemsContentProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewLicenseResource,
		resources.NewFormResource,
		resources.NewWorkflowResource,
		resources.NewResourceResource,
		resources.NewBlacklistResource,
		resources.NewCatalogueItemResource,
		resources.NewCategoryResource,
	}
}

func (p *RemsContentProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *RemsContentProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		data_sources.NewOrganizationDataSource,
		data_sources.NewLicenseDataSource,
		data_sources.NewActorDataSource,
		data_sources.NewBlacklistUserDataSource,
		data_sources.NewCatalogueItemDataSource,
		data_sources.NewCategoryDataSource,
		data_sources.NewFormDataSource,
		data_sources.NewResourceDataSource,
		data_sources.NewWorkflowDataSource,
	}
}

// :description :email :date :phone-number :table :header :texta :option :label :multiselect :ip-address :attachment :text

func (p *RemsContentProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RemsContentProvider{
			version: version,
		}
	}
}

// Add a test constructor
func NewWithClient(version string, c *remsclient.ClientWithResponses) func() provider.Provider {
	return func() provider.Provider {
		return &RemsContentProvider{version: version, clientOverride: c}
	}
}
