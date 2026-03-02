package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/organizationinvitation"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = &OrganizationInvitationResource{}
	_ resource.ResourceWithConfigure = &OrganizationInvitationResource{}
	_ resource.ResourceWithImportState = &OrganizationInvitationResource{}
)

func NewOrganizationInvitationResource() resource.Resource {
	return &OrganizationInvitationResource{}
}

type OrganizationInvitationResource struct {
	configured bool
}

type OrganizationInvitationResourceModel struct {
	ID              types.String `tfsdk:"id"`
	OrganizationID  types.String `tfsdk:"organization_id"`
	EmailAddress    types.String `tfsdk:"email_address"`
	Role            types.String `tfsdk:"role"`
	RedirectURL     types.String `tfsdk:"redirect_url"`
	InviterUserID   types.String `tfsdk:"inviter_user_id"`
	PublicMetadata  types.String `tfsdk:"public_metadata"`
	PrivateMetadata types.String `tfsdk:"private_metadata"`
	Status          types.String `tfsdk:"status"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
}

func (r *OrganizationInvitationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationInvitationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_invitation"
}

func (r *OrganizationInvitationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk organization invitation. Invitations are immutable — any change requires replacement.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the invitation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the organization to invite the user to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email_address": schema.StringAttribute{
				Required:    true,
				Description: "Email address to send the invitation to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Required:    true,
				Description: "Role key to assign to the invited user (e.g. \"internal_admin\").",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"redirect_url": schema.StringAttribute{
				Optional:    true,
				Description: "URL to redirect the user to after accepting the invitation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"inviter_user_id": schema.StringAttribute{
				Optional:    true,
				Description: "User ID of the person sending the invitation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_metadata": schema.StringAttribute{
				Optional:    true,
				Description: "Public metadata as a JSON string.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"private_metadata": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Private metadata as a JSON string.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the invitation: pending, accepted, or revoked.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the invitation was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the invitation was last updated.",
			},
		},
	}
}

func (r *OrganizationInvitationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationInvitationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &organizationinvitation.CreateParams{
		OrganizationID: plan.OrganizationID.ValueString(),
		EmailAddress:   clerkgo.String(plan.EmailAddress.ValueString()),
		Role:           clerkgo.String(plan.Role.ValueString()),
	}

	if !plan.RedirectURL.IsNull() && !plan.RedirectURL.IsUnknown() {
		params.RedirectURL = clerkgo.String(plan.RedirectURL.ValueString())
	}
	if !plan.InviterUserID.IsNull() && !plan.InviterUserID.IsUnknown() {
		params.InviterUserID = clerkgo.String(plan.InviterUserID.ValueString())
	}
	if !plan.PublicMetadata.IsNull() && !plan.PublicMetadata.IsUnknown() {
		raw := json.RawMessage(plan.PublicMetadata.ValueString())
		params.PublicMetadata = &raw
	}
	if !plan.PrivateMetadata.IsNull() && !plan.PrivateMetadata.IsUnknown() {
		raw := json.RawMessage(plan.PrivateMetadata.ValueString())
		params.PrivateMetadata = &raw
	}

	inv, err := organizationinvitation.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create organization invitation", err.Error())
		return
	}

	mapOrgInvitationResponseToModel(inv, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created organization invitation", map[string]any{"id": inv.ID})
}

func (r *OrganizationInvitationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationInvitationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	inv, err := organizationinvitation.Get(ctx, &organizationinvitation.GetParams{
		OrganizationID: state.OrganizationID.ValueString(),
		ID:             state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read organization invitation",
			fmt.Sprintf("Could not read invitation ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	mapOrgInvitationResponseToModel(inv, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *OrganizationInvitationResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No update — all fields use RequiresReplace()
	resp.Diagnostics.AddError(
		"Update not supported",
		"Organization invitations cannot be updated. All changes require replacement.",
	)
}

func (r *OrganizationInvitationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationInvitationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete maps to Revoke for invitations
	_, err := organizationinvitation.Revoke(ctx, &organizationinvitation.RevokeParams{
		OrganizationID: state.OrganizationID.ValueString(),
		ID:             state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to revoke organization invitation",
			fmt.Sprintf("Could not revoke invitation ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Revoked organization invitation", map[string]any{"id": state.ID.ValueString()})
}

func (r *OrganizationInvitationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import uses composite key: org_id/invitation_id
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: organization_id/invitation_id, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func mapOrgInvitationResponseToModel(inv *clerkgo.OrganizationInvitation, model *OrganizationInvitationResourceModel) {
	model.ID = types.StringValue(inv.ID)
	model.OrganizationID = types.StringValue(inv.OrganizationID)
	model.EmailAddress = types.StringValue(inv.EmailAddress)
	model.Role = types.StringValue(inv.Role)
	model.Status = types.StringValue(inv.Status)
	model.CreatedAt = types.StringValue(time.UnixMilli(inv.CreatedAt).UTC().Format(time.RFC3339))
	model.UpdatedAt = types.StringValue(time.UnixMilli(inv.UpdatedAt).UTC().Format(time.RFC3339))

	if len(inv.PublicMetadata) > 0 && string(inv.PublicMetadata) != "{}" {
		model.PublicMetadata = types.StringValue(normalizeJSON(string(inv.PublicMetadata)))
	}
	if len(inv.PrivateMetadata) > 0 && string(inv.PrivateMetadata) != "{}" {
		model.PrivateMetadata = types.StringValue(normalizeJSON(string(inv.PrivateMetadata)))
	}
}
