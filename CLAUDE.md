# CLAUDE.md

## Project Purpose

Forked Terraform provider for [Clerk](https://clerk.com) (from buildwithdeck/terraform-provider-clerk, MPL-2.0). Being extended to manage Arete's full Clerk infrastructure-as-code across dev and prod environments.

## Current State

The provider has 3 working resources:
- `clerk_jwt_template` — JWT template CRUD (most complete reference pattern)
- `clerk_organization` — Organization CRUD
- `clerk_organization_settings` — Singleton settings (no Delete, synthetic ID)

## Build / Test / Run

```bash
make build          # Compile provider
make install        # Build + install to $GOPATH/bin
make test           # Unit tests
make testacc        # Acceptance tests (requires CLERK_API_KEY env var)
make lint           # golangci-lint
make generate       # Generate docs via tfplugindocs
make fmt            # Format code
```

Acceptance tests run against a live Clerk instance:
```bash
export CLERK_API_KEY="sk_test_..."
make testacc
```

## Tech Stack

- Go 1.24+
- Terraform Plugin Framework v1.17 (`hashicorp/terraform-plugin-framework`)
- Clerk Go SDK v2.5.1 (`clerk/clerk-sdk-go/v2`)
- Global API key: `clerkgo.SetKey(apiKey)` in provider.go

## Architecture & Patterns

### Resource Pattern (follow for all new resources)

Every resource follows this exact pattern — reference `resource_organization.go` as the canonical example:

1. **Interface assertions** at top of file:
```go
var (
    _ resource.Resource                = &MyResource{}
    _ resource.ResourceWithConfigure   = &MyResource{}
    _ resource.ResourceWithImportState = &MyResource{}
)
```

2. **Struct** with `configured bool` field
3. **Model** with `tfsdk` tags using `types.String`, `types.Int64`, `types.Bool`, `types.Set`
4. **Configure()** — validates `req.ProviderData.(string)`, sets `r.configured = true`
5. **Metadata()** — returns `req.ProviderTypeName + "_suffix"`
6. **Schema()** — computed fields use `UseStateForUnknown()`, write-only fields preserved in mapper
7. **Create/Read/Update/Delete** — use SDK sub-packages directly
8. **ImportState** — `resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)`
9. **mapResponseToModel()** — converts API response to TF model. Timestamps: `time.UnixMilli(x).UTC().Format(time.RFC3339)`

### Optional Field Pattern
```go
if !plan.Field.IsNull() && !plan.Field.IsUnknown() {
    params.Field = clerkgo.String(plan.Field.ValueString())
}
```

### SDK Sub-packages
Import and use directly — no client object needed:
```go
import "github.com/clerk/clerk-sdk-go/v2/organization"
result, err := organization.Create(ctx, &organization.CreateParams{...})
```

### Test Pattern
Three-step acceptance tests: Create → Import → Update
```go
func TestAccMyResource(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            { Config: ..., ConfigStateChecks: [...] },  // Create
            { ResourceName: ..., ImportState: true },    // Import
            { Config: ..., ConfigStateChecks: [...] },  // Update
        },
    })
}
```

## Key Files

- `internal/provider/provider.go` — Provider config, resource registration (`Resources()` function)
- `internal/provider/resource_organization.go` — **Best reference pattern** for full CRUD resources
- `internal/provider/resource_jwt_template.go` — Reference for write-only fields (signing_key)
- `internal/provider/resource_organization_settings.go` — Reference for singleton/no-delete pattern
- `internal/provider/provider_test.go` — Test infrastructure (precheck, factory)
- `go.mod` — Dependencies (SDK v2.5.1 already has all needed sub-packages)

## Multi-Environment Strategy

Dev and prod use separate Clerk instances with different API keys:
- Dev: `CLERK_API_KEY=sk_test_...`
- Prod: `CLERK_API_KEY=sk_live_...`

Terraform modules in `terraform/` directory with per-environment configs.
