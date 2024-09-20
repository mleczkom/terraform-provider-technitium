package provider

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"terraform-provider-technitium/internal/provider/technitium"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &dnsZoneResource{}
	_ resource.ResourceWithConfigure = &dnsZoneResource{}
)

func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// NewDnsZoneResource is a helper function to simplify the provider implementation.
func NewDnsZoneResource() resource.Resource {
	return &dnsZoneResource{}
}

// dnsZoneResource is the resource implementation.

type dnsZoneResource struct {
	client *technitium.APIClient
}

// orderResourceModel maps the resource schema data.
type dnsZoneResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

// Metadata returns the resource type name.
func (r *dnsZoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone"
}

// Schema defines the schema for the resource.
func (r *dnsZoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"type": schema.StringAttribute{
				Required: true,
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *dnsZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan dnsZoneResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new dns zone
	zone := r.client.DnsZoneAPI.CreateDnsZone(ctx)
	zone = zone.Zone(plan.Name.ValueString())
	zone = zone.Type_(plan.Type.ValueString())
	answ, _, err := zone.Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating dns zone",
			"Could not create dns zone, unexpected error: "+err.Error(),
		)
		return
	}

	if answ.GetStatus() != "ok" {
		resp.Diagnostics.AddError(
			"Error creating dns zone",
			"Could not create dns zone, unexpected error: "+answ.GetErrorMessage(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	ID := GetMD5Hash(plan.Name.ValueString())
	plan.ID = types.StringValue(ID)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *dnsZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *dnsZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *dnsZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Retrieve values from state
	var state dnsZoneResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing dns zone
	zone := r.client.DnsZoneAPI.DeleteDnsZone(ctx)
	zone = zone.Zone(state.Name.ValueString())
	answ, _, err := zone.Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting dns zone",
			"Could not delete dns zone, unexpected error: "+err.Error(),
		)
		return
	}

	if answ.GetStatus() != "ok" {
		resp.Diagnostics.AddError(
			"Error creating dns zone",
			"Could not create dns zone, unexpected error: "+answ.GetErrorMessage(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *dnsZoneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client
}
