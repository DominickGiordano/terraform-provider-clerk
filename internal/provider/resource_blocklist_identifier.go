package provider

import (
	"context"
	"fmt"
	"time"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/blocklistidentifier"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &BlocklistIdentifierResource{}
	_ resource.ResourceWithConfigure   = &BlocklistIdentifierResource{}
	_ resource.ResourceWithImportState = &BlocklistIdentifierResource{}
)

func NewBlocklistIdentifierResource() resource.Resource {
	return &BlocklistIdentifierResource{}
}

type BlocklistIdentifierResource struct {
	configured bool
}

type BlocklistIdentifierResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Identifier     types.String `tfsdk:"identifier"`
	IdentifierType types.String `tfsdk:"identifier_type"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (r *BlocklistIdentifierResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	_, ok := req.ProviderData.(string)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected string (API key), got something else. Please report this issue.",
		)
		return
	}
	r.configured = true
}

func (r *BlocklistIdentifierResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blocklist_identifier"
}

func (r *BlocklistIdentifierResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk blocklist identifier. Blocklist identifiers prevent sign-ups from specific email addresses, phone numbers, or domains.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the blocklist entry.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"identifier": schema.StringAttribute{
				Required:    true,
				Description: "The identifier to blocklist (e.g. \"@spam.com\" for a domain, or a specific email address).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"identifier_type": schema.StringAttribute{
				Computed:    true,
				Description: "Type of the identifier: email_address, phone_number, domain, or web3_wallet.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the blocklist entry was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the blocklist entry was last updated.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *BlocklistIdentifierResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BlocklistIdentifierResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &blocklistidentifier.CreateParams{
		Identifier: clerkgo.String(plan.Identifier.ValueString()),
	}

	entry, err := blocklistidentifier.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create blocklist identifier", err.Error())
		return
	}

	mapBlocklistResponseToModel(entry, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created blocklist identifier", map[string]any{"id": entry.ID})
}

func (r *BlocklistIdentifierResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BlocklistIdentifierResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No Get endpoint — must list all and find by ID
	list, err := blocklistidentifier.List(ctx, &blocklistidentifier.ListParams{})
	if err != nil {
		resp.Diagnostics.AddError("Unable to list blocklist identifiers", err.Error())
		return
	}

	var found *clerkgo.BlocklistIdentifier
	for _, entry := range list.BlocklistIdentifiers {
		if entry.ID == state.ID.ValueString() {
			found = entry
			break
		}
	}

	if found == nil {
		// Resource was deleted outside Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	mapBlocklistResponseToModel(found, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *BlocklistIdentifierResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No update — identifier uses RequiresReplace()
	resp.Diagnostics.AddError(
		"Update not supported",
		"Blocklist identifiers cannot be updated. All changes require replacement.",
	)
}

func (r *BlocklistIdentifierResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BlocklistIdentifierResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := blocklistidentifier.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete blocklist identifier",
			fmt.Sprintf("Could not delete blocklist identifier ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted blocklist identifier", map[string]any{"id": state.ID.ValueString()})
}

func (r *BlocklistIdentifierResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapBlocklistResponseToModel(entry *clerkgo.BlocklistIdentifier, model *BlocklistIdentifierResourceModel) {
	model.ID = types.StringValue(entry.ID)
	model.Identifier = types.StringValue(entry.Identifier)
	model.IdentifierType = types.StringValue(entry.IdentifierType)
	model.CreatedAt = types.StringValue(time.UnixMilli(entry.CreatedAt).UTC().Format(time.RFC3339))
	model.UpdatedAt = types.StringValue(time.UnixMilli(entry.UpdatedAt).UTC().Format(time.RFC3339))
}
