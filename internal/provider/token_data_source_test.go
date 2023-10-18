package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccTokenDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: `
data "kerberos_token" "test" {
  username = "Administrator"
  password = "Test1234!"
  realm = "TEST.LAN"
  service = "HTTP/test.lan"
  kdc = "localhost:1088"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.kerberos_token.test", "id"),
					resource.TestCheckResourceAttrSet("data.kerberos_token.test", "token"),
				),
			},
		},
	})
}
