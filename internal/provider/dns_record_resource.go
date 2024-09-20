package provider

import (
	"context"
	"fmt"
	"terraform-provider-technitium/internal/provider/technitium"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &dnsRecordResource{}
	_ resource.ResourceWithConfigure = &dnsRecordResource{}
)

// NewDnsRecordResource is a helper function to simplify the provider implementation.
func NewDnsRecordResource() resource.Resource {
	return &dnsRecordResource{}
}

// dnsRecordResource is the resource implementation.

type dnsRecordResource struct {
	client *technitium.APIClient
}

// orderResourceModel maps the resource schema data.
type dnsRecordResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Zone          types.String `tfsdk:"zone"`
	Domain        types.String `tfsdk:"domain"`
	IPAddress     types.String `tfsdk:"ip_address"`
	Type          types.String `tfsdk:"type"`
	Ttl           types.Int32  `tfsdk:"ttl"`
	Ptr           types.Bool   `tfsdk:"ptr"`
	CreatePtrZone types.Bool   `tfsdk:"create_ptr_zone"`
	LastUpdated   types.String `tfsdk:"last_updated"`
}

// Metadata returns the resource type name.
func (r *dnsRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

// Schema defines the schema for the resource.
func (r *dnsRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"zone": schema.StringAttribute{
				Required: true,
			},
			"domain": schema.StringAttribute{
				Required: true,
			},
			"ip_address": schema.StringAttribute{
				Required: true,
			},
			"type": schema.StringAttribute{
				Required: true,
			},
			"ttl": schema.Int32Attribute{
				Optional: true,
			},
			"ptr": schema.BoolAttribute{
				Optional: true,
			},
			"create_ptr_zone": schema.BoolAttribute{
				Optional: true,
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *dnsRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var (
		plan dnsRecordResourceModel
	)

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new dns record

	record := r.client.DnsRecordAPI.CreateDnsRecord(ctx)
	record = record.Zone(plan.Zone.ValueString())
	record = record.Type_(plan.Type.ValueString())
	record = record.Domain(plan.Domain.ValueString())
	record = record.IpAddress(plan.IPAddress.ValueString())
	if !plan.Ttl.IsNull() {
		record = record.Ttl(plan.Ttl.ValueInt32())
	}
	if !plan.Ptr.IsNull() {
		record = record.Ptr(plan.Ptr.ValueBool())
	}
	if !plan.CreatePtrZone.IsNull() {
		record = record.CreatePtrZone(plan.CreatePtrZone.ValueBool())
	}

	answ, _, err := record.Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating dns record",
			"Could not create dns record, unexpected error: "+err.Error(),
		)
		return
	}

	if answ.GetStatus() != "ok" {
		resp.Diagnostics.AddError(
			"Error creating dns record",
			"Could not create dns record, unexpected error: "+answ.GetErrorMessage(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	fqdn := plan.Domain.ValueString() + "." + plan.Zone.ValueString()
	ID := GetMD5Hash(fqdn)
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
func (r *dnsRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *dnsRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan
	var (
		plan  dnsRecordResourceModel
		state dnsRecordResourceModel
	)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	record := r.client.DnsRecordAPI.UpdateDnsRecord(ctx)
	record = record.Zone(state.Zone.ValueString())
	record = record.Type_(state.Type.ValueString())
	record = record.Domain(state.Domain.ValueString())
	record = record.IpAddress(state.IPAddress.ValueString())

	record = record.NewDomain(plan.Domain.ValueString())
	record = record.NewIpAddress(plan.IPAddress.ValueString())
	if !plan.Ttl.IsNull() {
		record = record.Ttl(plan.Ttl.ValueInt32())
	}

	// Update existing dns record
	answ, _, err := record.Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating dns record",
			"Could not update dns record, unexpected error: "+err.Error(),
		)
		return
	}

	if answ.GetStatus() != "ok" {
		resp.Diagnostics.AddError(
			"Error updating dns record",
			"Could not create dns record, unexpected error: "+answ.GetErrorMessage(),
		)
		return
	}

	fqdn := plan.Domain.ValueString() + "." + plan.Zone.ValueString()
	ID := GetMD5Hash(fqdn)
	plan.ID = types.StringValue(ID)
	// // Update resource state with updated items and timestamp
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *dnsRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Retrieve values from state
	var state dnsRecordResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing dns record
	record := r.client.DnsRecordAPI.DeleteDnsRecord(ctx)
	record = record.Domain(state.Domain.ValueString())
	record = record.Type_(state.Type.ValueString())
	record = record.IpAddress(state.IPAddress.ValueString())
	answ, _, err := record.Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting dns record",
			"Could not delete dns record, unexpected error: "+err.Error(),
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
}

// Configure adds the provider configured client to the resource.
func (r *dnsRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
