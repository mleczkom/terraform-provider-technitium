package provider

import (
	"context"
	"net/http"
	"os"
	"terraform-provider-technitium/internal/provider/technitium"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &technitiumProvider{}
)

type CustomHTTPClient struct {
	client *http.Client
	token  string
}

func (c *CustomHTTPClient) RoundTrip(req *http.Request) (*http.Response, error) {
	query := req.URL.Query()
	query.Add("token", c.token)
	req.URL.RawQuery = query.Encode()

	if c.client.Transport == nil {
		c.client.Transport = http.DefaultTransport
	}
	return c.client.Transport.RoundTrip(req)
}

type technitiumProviderModel struct {
	Host  types.String `tfsdk:"host"`
	Token types.String `tfsdk:"token"`
}

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &technitiumProvider{
			version: version,
		}
	}
}

// technitiumProvider is the provider implementation.
type technitiumProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// Metadata returns the provider type name.
func (p *technitiumProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "technitium"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *technitiumProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional: true,
			},
			"token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

// Configure prepares a HashiCups API client for data sources and resources.
func (p *technitiumProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config technitiumProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Configuring Technitium client")

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown Technitium API Host",
			"The provider cannot create the Technitium API client as there is an unknown configuration value for the Technitium API host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the TECHNITIUM_HOST environment variable.",
		)
	}

	if config.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Unknown Technitium API Token",
			"The provider cannot create the Technitium API client as there is an unknown configuration value for the Technitium API token. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the TECHNITIUM_TOKEN environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	host := os.Getenv("TECHNITIUM_HOST")
	token := os.Getenv("TECHNITIUM_TOKEN")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing Technitium API Host",
			"The provider cannot create the Technitium API client as there is a missing or empty value for the Technitium API host. "+
				"Set the host value in the configuration or use the TECHNITIUM_HOST environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing Technitium API Token",
			"The provider cannot create the Technitium API client as there is a missing or empty value for the Technitium API token. "+
				"Set the token value in the configuration or use the TECHNITIUM_TOKEN environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create a new HashiCups client using the configuration values
	customClient := &CustomHTTPClient{
		client: http.DefaultClient,
		token:  token,
	}
	cfg := technitium.NewConfiguration()
	cfg.Servers = technitium.ServerConfigurations{
		{
			URL: host,
		},
	}
	cfg.HTTPClient = &http.Client{
		Transport: customClient,
	}
	apiClient := technitium.NewAPIClient(cfg)

	// if err != nil {
	// 	resp.Diagnostics.AddError(
	// 		"Unable to Create HashiCups API Client",
	// 		"An unexpected error occurred when creating the HashiCups API client. "+
	// 			"If the error is not clear, please contact the provider developers.\n\n"+
	// 			"HashiCups Client Error: "+err.Error(),
	// 	)
	// 	return
	// }

	// Make the HashiCups client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = apiClient
	resp.ResourceData = apiClient
}

// DataSources defines the data sources implemented in the provider.
func (p *technitiumProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDnsZonesDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *technitiumProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDnsZoneResource,
		NewDnsRecordResource,
	}
}
