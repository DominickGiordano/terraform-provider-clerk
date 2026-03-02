package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccOrganizationPermissionResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccOrganizationPermissionConfig("TF Acc Test Permission", "org:tf_acc_test:read", "Test permission for acceptance tests"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_organization_permission.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("TF Acc Test Permission"),
					),
					statecheck.ExpectKnownValue(
						"clerk_organization_permission.test",
						tfjsonpath.New("key"),
						knownvalue.StringExact("org:tf_acc_test:read"),
					),
					statecheck.ExpectKnownValue(
						"clerk_organization_permission.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test permission for acceptance tests"),
					),
				},
			},
			// Import
			{
				ResourceName:      "clerk_organization_permission.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: testAccOrganizationPermissionConfig("TF Acc Test Permission Updated", "org:tf_acc_test:read", "Updated description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_organization_permission.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("TF Acc Test Permission Updated"),
					),
					statecheck.ExpectKnownValue(
						"clerk_organization_permission.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated description"),
					),
				},
			},
		},
	})
}

func testAccOrganizationPermissionConfig(name, key, description string) string {
	return fmt.Sprintf(`
resource "clerk_organization_permission" "test" {
  name        = %[1]q
  key         = %[2]q
  description = %[3]q
}
`, name, key, description)
}
