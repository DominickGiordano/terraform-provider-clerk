package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccOrganizationRoleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with permission
			{
				Config: testAccOrganizationRoleConfig(
					"TF Acc Test Role",
					"org:tf_acc_test_role",
					"Test role for acceptance tests",
					`["org:tf_acc_test_perm:read"]`,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_organization_role.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("TF Acc Test Role"),
					),
					statecheck.ExpectKnownValue(
						"clerk_organization_role.test",
						tfjsonpath.New("key"),
						knownvalue.StringExact("org:tf_acc_test_role"),
					),
				},
			},
			// Import
			{
				ResourceName:      "clerk_organization_role.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: testAccOrganizationRoleConfig(
					"TF Acc Test Role Updated",
					"org:tf_acc_test_role",
					"Updated description",
					`["org:tf_acc_test_perm:read", "org:tf_acc_test_perm:write"]`,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_organization_role.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("TF Acc Test Role Updated"),
					),
				},
			},
		},
	})
}

func testAccOrganizationRoleConfig(name, key, description, permissions string) string {
	return fmt.Sprintf(`
resource "clerk_organization_permission" "test_read" {
  name = "TF Acc Test Perm Read"
  key  = "org:tf_acc_test_perm:read"
}

resource "clerk_organization_permission" "test_write" {
  name = "TF Acc Test Perm Write"
  key  = "org:tf_acc_test_perm:write"
}

resource "clerk_organization_role" "test" {
  name        = %[1]q
  key         = %[2]q
  description = %[3]q
  permissions = %[4]s

  depends_on = [
    clerk_organization_permission.test_read,
    clerk_organization_permission.test_write,
  ]
}
`, name, key, description, permissions)
}
