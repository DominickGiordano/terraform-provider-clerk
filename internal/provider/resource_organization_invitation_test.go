package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccOrganizationInvitationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccOrganizationInvitationConfig("tf-acc-test-invite@example.com", "org:member"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_organization_invitation.test",
						tfjsonpath.New("email_address"),
						knownvalue.StringExact("tf-acc-test-invite@example.com"),
					),
					statecheck.ExpectKnownValue(
						"clerk_organization_invitation.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("org:member"),
					),
					statecheck.ExpectKnownValue(
						"clerk_organization_invitation.test",
						tfjsonpath.New("status"),
						knownvalue.StringExact("pending"),
					),
				},
			},
			// Import with composite key
			{
				ResourceName:      "clerk_organization_invitation.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["clerk_organization_invitation.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["organization_id"], rs.Primary.Attributes["id"]), nil
				},
			},
		},
	})
}

func testAccOrganizationInvitationConfig(email, role string) string {
	return fmt.Sprintf(`
resource "clerk_organization" "test_invite_org" {
  name = "tf-acc-test-invite-org"
  slug = "tf-acc-test-invite-org"
}

resource "clerk_organization_invitation" "test" {
  organization_id = clerk_organization.test_invite_org.id
  email_address   = %[1]q
  role            = %[2]q
}
`, email, role)
}
