package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccSAMLConnectionResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccSAMLConnectionConfig("TF Acc Test SAML", "tf-acc-test-saml.example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_saml_connection.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("TF Acc Test SAML"),
					),
					statecheck.ExpectKnownValue(
						"clerk_saml_connection.test",
						tfjsonpath.New("domain"),
						knownvalue.StringExact("tf-acc-test-saml.example.com"),
					),
					statecheck.ExpectKnownValue(
						"clerk_saml_connection.test",
						tfjsonpath.New("provider"),
						knownvalue.StringExact("saml_custom"),
					),
				},
			},
			// Import
			{
				ResourceName:      "clerk_saml_connection.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: testAccSAMLConnectionConfig("TF Acc Test SAML Updated", "tf-acc-test-saml.example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_saml_connection.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("TF Acc Test SAML Updated"),
					),
				},
			},
		},
	})
}

func testAccSAMLConnectionConfig(name, domain string) string {
	return fmt.Sprintf(`
resource "clerk_saml_connection" "test" {
  name     = %[1]q
  domain   = %[2]q
  provider = "saml_custom"
}
`, name, domain)
}
