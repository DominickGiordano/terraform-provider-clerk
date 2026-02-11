package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccJWTTemplateResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccJWTTemplateConfig("tf-acc-test", `{"email":"{{user.primary_email_address}}"}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_jwt_template.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-test"),
					),
				},
			},
			// Import
			{
				ResourceName:            "clerk_jwt_template.test",
				ImportState:             true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore: []string{"signing_key"},
			},
			// Update
			{
				Config: testAccJWTTemplateConfig("tf-acc-test-updated", `{"email":"{{user.primary_email_address}}","name":"{{user.first_name}}"}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_jwt_template.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-test-updated"),
					),
				},
			},
		},
	})
}

func testAccJWTTemplateConfig(name, claims string) string {
	return fmt.Sprintf(`
resource "clerk_jwt_template" "test" {
  name   = %[1]q
  claims = %[2]q
}
`, name, claims)
}
