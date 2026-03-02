package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccBlocklistIdentifierResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: `
resource "clerk_blocklist_identifier" "test" {
  identifier = "tf-acc-test-blocked@example.com"
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_blocklist_identifier.test",
						tfjsonpath.New("identifier"),
						knownvalue.StringExact("tf-acc-test-blocked@example.com"),
					),
					statecheck.ExpectKnownValue(
						"clerk_blocklist_identifier.test",
						tfjsonpath.New("identifier_type"),
						knownvalue.StringExact("email_address"),
					),
				},
			},
			// Import
			{
				ResourceName:      "clerk_blocklist_identifier.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
