# Clerk Terraform Provider — Extension Plan

## Context

Arete completed a Clerk POC spike (see `../clerk-poc-spike/docs/FINDINGS.md`) with a **Go recommendation**. Two apps (Phoenix/LiveView/Ash + Next.js) authenticate through Clerk successfully. Now we need to manage Clerk configuration as infrastructure-as-code instead of clicking through the dashboard.

This forked provider (from buildwithdeck, MPL-2.0) already has 3 resources. We're adding 6 more to cover:
- Microsoft SSO via Entra ID (SAML)
- Role-based access control (internal team vs external clients)
- Per-app permissions
- Multi-tenant organizations for external clients
- Email domain restrictions
- Multi-environment deployment (dev/prod)

**Key discovery:** The Clerk Go SDK (`clerk-sdk-go/v2`) has full CRUD for SAML connections via the `samlconnection` package — Microsoft Entra ID SSO can be fully managed via Terraform.

---

## Arete's Setup (What We're Modeling)

```
Internal Team (Arete)
  → Sign in via Microsoft Entra ID (SAML SSO)
  → Full access to all apps
  → Roles: internal_admin, internal_user

External Clients (per-company orgs)
  → Sign in via email/password, Gmail, or their own SSO
  → Scoped access — each org gets specific app permissions
  → Roles: client_admin, client_viewer

Apps:
  → Phoenix app (Elixir/LiveView/Ash) — main product
  → Admin panel — internal only
  → Future: more apps on the same auth

Permissions:
  → app:phoenix:read, app:phoenix:admin
  → app:admin:read, app:admin:admin
  → Roles map to sets of permissions
```

---

## New Resources to Build (6 total)

### Phase 1: Permissions + Roles (~5 hours)

Foundation for RBAC. Permissions first, then roles that reference them.

#### 1a. `clerk_organization_permission`

**File:** `internal/provider/resource_organization_permission.go` + `_test.go`

**Schema:**

| Attribute | Type | Required | Notes |
|-----------|------|----------|-------|
| `id` | String | Computed | From API |
| `name` | String | Required | Display name (e.g. "Phoenix App Read") |
| `key` | String | Required | Unique key (e.g. "app:phoenix:read") |
| `description` | String | Optional | |
| `type` | String | Computed | "system" or "user" |
| `created_at` | String | Computed | RFC3339 |
| `updated_at` | String | Computed | RFC3339 |

**SDK:** `organizationpermission.Create/Get/Update/Delete` — standard full CRUD.

**CRUD:** Straightforward. Follow `resource_organization.go` pattern exactly.

#### 1b. `clerk_organization_role`

**File:** `internal/provider/resource_organization_role.go` + `_test.go`

**Schema:**

| Attribute | Type | Required | Notes |
|-----------|------|----------|-------|
| `id` | String | Computed | |
| `name` | String | Required | e.g. "Internal Admin" |
| `key` | String | Required | e.g. "internal_admin" |
| `description` | String | Optional | |
| `permissions` | Set(String) | Optional | Set of permission keys |
| `created_at` | String | Computed | |
| `updated_at` | String | Computed | |

**SDK:** `organizationrole.Create/Get/Update/Delete`
- `CreateParams.Permissions` accepts `*[]string` of permission keys — roles can be created with permissions inline
- `UpdateParams.Permissions` also accepts `*[]string` — full replacement on update
- On Read, the API returns `[]*OrganizationPermission` embedded in the role — extract `.Key` values into the Set

**Important:** Use `schema.SetAttribute` with `ElementType: types.StringType` for the permissions attribute (avoids ordering issues in plans). The SDK also has `AssignPermission`/`RemovePermission` if full replacement proves problematic, but inline `Permissions` field should work.

---

### Phase 2: Access Control Lists (~3.5 hours)

#### 2a. `clerk_allowlist_identifier`

**File:** `internal/provider/resource_allowlist_identifier.go` + `_test.go`

**Schema:**

| Attribute | Type | Required | Notes |
|-----------|------|----------|-------|
| `id` | String | Computed | |
| `identifier` | String | Required | ForceNew. e.g. "@aretecp.com" |
| `notify` | Bool | Optional | ForceNew. Send invite email |
| `identifier_type` | String | Computed | email_address, phone_number, domain, web3_wallet |
| `created_at` | String | Computed | |
| `updated_at` | String | Computed | |

**SDK quirk — No Get endpoint, No Update:**
- **Read** must call `allowlistidentifier.List(ctx, &ListParams{})` and scan results for matching `ID`. If not found → `resp.State.RemoveResource(ctx)` (drift detection).
- **No Update** — all mutable fields use `RequiresReplace()`. Any change destroys and recreates.
- `identifier` and `notify` both use `RequiresReplace()`.

#### 2b. `clerk_blocklist_identifier`

**File:** `internal/provider/resource_blocklist_identifier.go` + `_test.go`

Same pattern as allowlist minus `notify` field. No Get, no Update. Read via List scan.

---

### Phase 3: SAML Connection — Microsoft Entra ID (~4 hours)

#### `clerk_saml_connection`

**File:** `internal/provider/resource_saml_connection.go` + `_test.go`

**Schema:**

| Attribute | Type | Required | Notes |
|-----------|------|----------|-------|
| `id` | String | Computed | |
| `name` | String | Required | e.g. "Microsoft Entra ID" |
| `domain` | String | Required | e.g. "aretecp.com" |
| `provider` | String | Required | ForceNew. "saml_microsoft", "saml_custom" |
| `organization_id` | String | Optional | Tie to a Clerk org |
| `idp_entity_id` | String | Optional | From Entra ID |
| `idp_sso_url` | String | Optional | From Entra ID |
| `idp_certificate` | String | Optional | Sensitive. From Entra ID |
| `idp_metadata_url` | String | Optional | Auto-config from Entra federation metadata URL |
| `idp_metadata` | String | Optional | Raw XML metadata |
| `active` | Bool | Optional | Enable/disable |
| `sync_user_attributes` | Bool | Optional | |
| `allow_subdomains` | Bool | Optional | |
| `allow_idp_initiated` | Bool | Optional | |
| `attribute_mapping` | Object | Optional | SingleNestedAttribute (see below) |
| `acs_url` | String | Computed | **Output** — configure in Entra ID |
| `sp_entity_id` | String | Computed | **Output** — configure in Entra ID |
| `sp_metadata_url` | String | Computed | **Output** — configure in Entra ID |
| `user_count` | Int64 | Computed | |
| `created_at` | String | Computed | |
| `updated_at` | String | Computed | |

**`attribute_mapping` nested attribute:**
```go
"attribute_mapping": schema.SingleNestedAttribute{
    Optional: true, Computed: true,
    Attributes: map[string]schema.Attribute{
        "user_id":       schema.StringAttribute{Optional: true, Computed: true},
        "email_address": schema.StringAttribute{Optional: true, Computed: true},
        "first_name":    schema.StringAttribute{Optional: true, Computed: true},
        "last_name":     schema.StringAttribute{Optional: true, Computed: true},
    },
},
```

**SDK:** `samlconnection.Create/Get/Update/Delete` — full CRUD. `provider` field only on `CreateParams` (not `UpdateParams`), hence `RequiresReplace()`.

**Key outputs:** After `terraform apply`, the computed `acs_url`, `sp_entity_id`, `sp_metadata_url` values are used to configure the Entra ID app registration. These can be output from the Terraform module.

---

### Phase 4: Organization Invitations — Deferrable (~3 hours)

#### `clerk_organization_invitation`

**File:** `internal/provider/resource_organization_invitation.go` + `_test.go`

| Attribute | Type | Required | Notes |
|-----------|------|----------|-------|
| `id` | String | Computed | |
| `organization_id` | String | Required | ForceNew |
| `email_address` | String | Required | ForceNew |
| `role` | String | Required | ForceNew. Role key |
| `redirect_url` | String | Optional | ForceNew |
| `inviter_user_id` | String | Optional | ForceNew |
| `status` | String | Computed | pending/accepted/revoked |
| `public_metadata` | String | Optional | ForceNew |
| `private_metadata` | String | Optional | ForceNew, Sensitive |
| `created_at` | String | Computed | |
| `updated_at` | String | Computed | |

**No Update** — all fields ForceNew. Delete maps to `organizationinvitation.Revoke()`. Import uses composite key `org_id/invitation_id` parsed in a custom `ImportState` handler.

---

### Phase 5: Provider Registration + Terraform Module (~2 hours)

#### 5a. Provider Registration

Modify `internal/provider/provider.go` → `Resources()`:
```go
func (p *ClerkProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewJWTTemplateResource,
        NewOrganizationResource,
        NewOrganizationSettingsResource,
        NewOrganizationPermissionResource,   // Phase 1
        NewOrganizationRoleResource,          // Phase 1
        NewAllowlistIdentifierResource,       // Phase 2
        NewBlocklistIdentifierResource,       // Phase 2
        NewSAMLConnectionResource,            // Phase 3
        NewOrganizationInvitationResource,    // Phase 4
    }
}
```

#### 5b. Terraform Module Structure

```
terraform/
  modules/
    clerk_instance/
      main.tf           # All resource definitions
      variables.tf      # Inputs
      outputs.tf        # Org IDs, SAML SP metadata URLs
      versions.tf       # Provider constraint
  environments/
    dev/
      main.tf           # Module call with dev config
      terraform.tfvars  # Dev API key ref, dev-specific values
    prod/
      main.tf           # Module call with prod config
      terraform.tfvars  # Prod-specific values
  MANUAL_SETUP.md       # Dashboard-only steps checklist
```

---

## Example Terraform Config (What the Module Produces)

```hcl
# ─── Permissions (per-app access) ───
resource "clerk_organization_permission" "phoenix_read" {
  name        = "Phoenix App Read"
  key         = "app:phoenix:read"
  description = "Read access to the Phoenix application"
}

resource "clerk_organization_permission" "phoenix_admin" {
  name        = "Phoenix App Admin"
  key         = "app:phoenix:admin"
  description = "Admin access to the Phoenix application"
}

resource "clerk_organization_permission" "admin_panel_read" {
  name        = "Admin Panel Read"
  key         = "app:admin:read"
  description = "Read access to the admin panel"
}

resource "clerk_organization_permission" "admin_panel_admin" {
  name        = "Admin Panel Admin"
  key         = "app:admin:admin"
  description = "Admin access to the admin panel"
}

# ─── Roles ───
resource "clerk_organization_role" "internal_admin" {
  name        = "Internal Admin"
  key         = "internal_admin"
  description = "Arete internal team - full access to all apps"
  permissions = ["app:phoenix:read", "app:phoenix:admin", "app:admin:read", "app:admin:admin"]
}

resource "clerk_organization_role" "internal_user" {
  name        = "Internal User"
  key         = "internal_user"
  description = "Arete internal team - read access"
  permissions = ["app:phoenix:read", "app:admin:read"]
}

resource "clerk_organization_role" "client_admin" {
  name        = "Client Admin"
  key         = "client_admin"
  description = "Client organization admin - phoenix only"
  permissions = ["app:phoenix:read", "app:phoenix:admin"]
}

resource "clerk_organization_role" "client_viewer" {
  name        = "Client Viewer"
  key         = "client_viewer"
  description = "Client organization read-only"
  permissions = ["app:phoenix:read"]
}

# ─── Organizations ───
resource "clerk_organization" "arete" {
  name = "Arete Capital Partners"
  slug = "arete-internal"
}

resource "clerk_organization" "client_acme" {
  name = "Acme Corp"
  slug = "acme"
}

# ─── Allowlist: Internal Domain ───
resource "clerk_allowlist_identifier" "aretecp" {
  identifier = "@aretecp.com"
}

# ─── JWT Template for Phoenix ───
resource "clerk_jwt_template" "phoenix" {
  name = "phoenix"
  claims = jsonencode({
    email           = "{{user.primary_email_address}}"
    name            = "{{user.full_name}}"
    org_id          = "{{org.id}}"
    org_slug        = "{{org.slug}}"
    org_role        = "{{org.role}}"
    org_permissions = "{{org.permissions}}"
  })
  lifetime           = 60
  allowed_clock_skew = 5
  signing_algorithm  = "RS256"
}

# ─── SAML Connection (Microsoft Entra ID) ───
resource "clerk_saml_connection" "microsoft" {
  name            = "Microsoft Entra ID"
  domain          = "aretecp.com"
  provider        = "saml_microsoft"
  organization_id = clerk_organization.arete.id
  active          = true

  idp_metadata_url     = "https://login.microsoftonline.com/TENANT_ID/federationmetadata/2007-06/federationmetadata.xml"
  sync_user_attributes = true
}

# After apply, use these outputs to configure the Entra ID app registration:
# - clerk_saml_connection.microsoft.acs_url
# - clerk_saml_connection.microsoft.sp_entity_id
```

---

## What Stays Manual (Dashboard-Only)

These have no Backend API — document in `terraform/MANUAL_SETUP.md`:

1. **Sign-in method toggles** — enable/disable email/password, phone, passkeys (Dashboard → Configure → Email, phone, username)
2. **Social OAuth provider credentials** — Google, GitHub, etc. client ID + secret (Dashboard → Configure → Social connections)
3. **MFA settings** — enable/disable authenticator app, SMS
4. **Custom branding** — logo, colors, theme for hosted sign-in
5. **Webhook endpoints** — configure callback URLs
6. **Custom domain** — set up custom auth domain

---

## Clerk Go SDK Sub-packages Reference

All packages exist in `clerk-sdk-go/v2` (v2.5.1). No go.mod changes needed.

| Package | Import | Confirmed |
|---------|--------|-----------|
| `organizationpermission` | `github.com/clerk/clerk-sdk-go/v2/organizationpermission` | Create/Get/Update/Delete |
| `organizationrole` | `github.com/clerk/clerk-sdk-go/v2/organizationrole` | Create/Get/Update/Delete + AssignPermission/RemovePermission |
| `allowlistidentifier` | `github.com/clerk/clerk-sdk-go/v2/allowlistidentifier` | Create/List/Delete (no Get, no Update) |
| `blocklistidentifier` | `github.com/clerk/clerk-sdk-go/v2/blocklistidentifier` | Create/List/Delete (no Get, no Update) |
| `samlconnection` | `github.com/clerk/clerk-sdk-go/v2/samlconnection` | Create/Get/Update/Delete |
| `organizationinvitation` | `github.com/clerk/clerk-sdk-go/v2/organizationinvitation` | Create/Get/Revoke (no Update, no Delete) |

---

## Verification

1. `make build` — provider compiles
2. `make test` — unit tests pass
3. `CLERK_API_KEY=sk_test_... make testacc` — acceptance tests pass against dev Clerk
4. `cd terraform/environments/dev && terraform init && terraform plan` — shows expected resources
5. `terraform apply` — creates orgs, roles, permissions, SAML in dev Clerk
6. Verify in Clerk Dashboard: orgs exist, roles have correct permissions, SAML SP metadata populated

---

## Effort Estimate

| Phase | Hours |
|---|---|
| 1: Permissions + Roles | ~5h |
| 2: Allowlist + Blocklist | ~3.5h |
| 3: SAML Connection | ~4h |
| 4: Invitations (deferrable) | ~3h |
| 5: Module + environments | ~2h |
| Docs + examples | ~1.5h |
| **Total** | **~19h (~2.5 days)** |

**Recommended order:** Ship phases 1-3 + 5 first (~14.5h). Phase 4 can wait.
