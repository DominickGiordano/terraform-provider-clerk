package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDomainResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccDomainConfig("test-satellite.example.com", true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_domain.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-satellite.example.com"),
					),
					statecheck.ExpectKnownValue(
						"clerk_domain.test",
						tfjsonpath.New("is_satellite"),
						knownvalue.Bool(true),
					),
				},
			},
			// Import
			{
				ResourceName:      "clerk_domain.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name
			{
				Config: testAccDomainConfig("test-satellite-updated.example.com", true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"clerk_domain.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-satellite-updated.example.com"),
					),
				},
			},
		},
	})
}

func testAccDomainConfig(name string, isSatellite bool) string {
	return fmt.Sprintf(`
resource "clerk_domain" "test" {
  name         = %[1]q
  is_satellite = %[2]t
}
`, name, isSatellite)
}
