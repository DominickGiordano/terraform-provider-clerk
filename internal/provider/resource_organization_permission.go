package provider

import (
	"context"
	"fmt"
	"time"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/organizationpermission"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &OrganizationPermissionResource{}
	_ resource.ResourceWithConfigure   = &OrganizationPermissionResource{}
	_ resource.ResourceWithImportState = &OrganizationPermissionResource{}
)

func NewOrganizationPermissionResource() resource.Resource {
	return &OrganizationPermissionResource{}
}

type OrganizationPermissionResource struct {
	configured bool
}

type OrganizationPermissionResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Key         types.String `tfsdk:"key"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *OrganizationPermissionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationPermissionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_permission"
}

func (r *OrganizationPermissionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk organization permission.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the permission.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the permission (e.g. \"Phoenix App Read\").",
			},
			"key": schema.StringAttribute{
				Required:    true,
				Description: "Unique key for the permission (e.g. \"app:phoenix:read\").",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the permission.",
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "Type of the permission: \"system\" or \"user\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the permission was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the permission was last updated.",
			},
		},
	}
}

func (r *OrganizationPermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationPermissionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &organizationpermission.CreateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
		Key:  clerkgo.String(plan.Key.ValueString()),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = clerkgo.String(plan.Description.ValueString())
	}

	perm, err := organizationpermission.Create(ctx, params)
	if err != nil {
		// Already exists — find by key and adopt into state
		list, listErr := organizationpermission.List(ctx, &organizationpermission.ListParams{})
		if listErr == nil {
			for _, existing := range list.OrganizationPermissions {
				if existing.Key == plan.Key.ValueString() {
					mapOrgPermissionResponseToModel(existing, &plan)
					resp.State.Set(ctx, plan)
					tflog.Debug(ctx, "Adopted existing organization permission", map[string]any{"id": existing.ID})
					return
				}
			}
		}
		resp.Diagnostics.AddError("Unable to create organization permission", err.Error())
		return
	}

	mapOrgPermissionResponseToModel(perm, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created organization permission", map[string]any{"id": perm.ID})
}

func (r *OrganizationPermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationPermissionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	perm, err := organizationpermission.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read organization permission",
			fmt.Sprintf("Could not read permission ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	mapOrgPermissionResponseToModel(perm, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *OrganizationPermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationPermissionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrganizationPermissionResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &organizationpermission.UpdateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
		Key:  clerkgo.String(plan.Key.ValueString()),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = clerkgo.String(plan.Description.ValueString())
	}

	perm, err := organizationpermission.Update(ctx, state.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update organization permission", err.Error())
		return
	}

	mapOrgPermissionResponseToModel(perm, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated organization permission", map[string]any{"id": perm.ID})
}

func (r *OrganizationPermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationPermissionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := organizationpermission.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete organization permission",
			fmt.Sprintf("Could not delete permission ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted organization permission", map[string]any{"id": state.ID.ValueString()})
}

func (r *OrganizationPermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapOrgPermissionResponseToModel(perm *clerkgo.OrganizationPermission, model *OrganizationPermissionResourceModel) {
	model.ID = types.StringValue(perm.ID)
	model.Name = types.StringValue(perm.Name)
	model.Key = types.StringValue(perm.Key)
	model.Type = types.StringValue(perm.Type)
	model.CreatedAt = types.StringValue(time.UnixMilli(perm.CreatedAt).UTC().Format(time.RFC3339))
	model.UpdatedAt = types.StringValue(time.UnixMilli(perm.UpdatedAt).UTC().Format(time.RFC3339))

	if perm.Description != nil {
		model.Description = types.StringValue(*perm.Description)
	}
}
