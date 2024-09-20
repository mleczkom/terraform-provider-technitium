package provider

import (
	"context"
	"fmt"
	"terraform-provider-technitium/internal/provider/technitium"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &dnsZonesDataSource{}
	_ datasource.DataSourceWithConfigure = &dnsZonesDataSource{}
)

// dnsZoneDataSourceModel maps the data source schema data.
type dnsZonesDataSourceModel struct {
	DnsZones []dnsZoneModel `tfsdk:"dns_zones"`
}

// dnsZoneModel maps coffees schema data.
type dnsZoneModel struct {
	Zone         types.String `tfsdk:"zone"`
	Type         types.String `tfsdk:"type"`
	DNSSEC       types.String `tfsdk:"dnssec"`
	Status       types.String `tfsdk:"status"`
	Serial       types.Int32  `tfsdk:"serial"`
	Expiry       types.String `tfsdk:"expiry"`
	LastModified types.String `tfsdk:"last_modified"`
}

type dnsZonesDataSource struct {
	client *technitium.APIClient
}

func NewDnsZonesDataSource() datasource.DataSource {
	return &dnsZonesDataSource{}
}

func (d *dnsZonesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zones"
}

// Schema defines the schema for the data source.
func (d *dnsZonesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"dns_zones": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"zone": schema.StringAttribute{
							Computed: true,
						},
						"type": schema.StringAttribute{
							Computed: true,
						},
						"dnssec": schema.StringAttribute{
							Computed: true,
						},
						"status": schema.StringAttribute{
							Computed: true,
						},
						"serial": schema.Int32Attribute{
							Computed: true,
						},
						"expiry": schema.StringAttribute{
							Computed: true,
						},
						"last_modified": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *dnsZonesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var (
		state  dnsZonesDataSourceModel
		Status string
		Type   string
	)

	answ, _, err := d.client.DnsZoneAPI.ListDnsZones(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read HashiCups Coffees",
			err.Error(),
		)
		return
	}

	if answ.GetStatus() != "ok" {
		resp.Diagnostics.AddError(
			"Error deleting dns record",
			"Could not create dns record, unexpected error: "+answ.GetErrorMessage(),
		)
		return
	}

	// Map response body to model
	for _, dnsZone := range answ.Response.Zones {
		if dnsZone.GetDisabled() {
			Status = "Disabled"
		} else {
			Status = "Enabled"
		}
		if dnsZone.GetInternal() {
			Type = "Internal"
		} else {
			Type = dnsZone.GetType()
		}

		dnsZoneState := dnsZoneModel{
			Zone:         types.StringValue(dnsZone.GetName()),
			Type:         types.StringValue(Type),
			DNSSEC:       types.StringValue(dnsZone.GetDnssecStatus()),
			Status:       types.StringValue(Status),
			Serial:       types.Int32Value(int32(dnsZone.GetSoaSerial())),
			Expiry:       types.StringValue(dnsZone.GetExpiry()),
			LastModified: types.StringValue(dnsZone.GetLastModified()),
		}

		state.DnsZones = append(state.DnsZones, dnsZoneState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *dnsZonesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*technitium.APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *technitiumclient.TechnitiumDNSClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}
