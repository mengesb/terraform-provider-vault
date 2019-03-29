package vault

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/vault/api"
)

func TestAccidentityGroupPolicies(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testProviders,
		CheckDestroy: testAccCheckidentityGroupPoliciesDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccidentityGroupPoliciesConfig(),
				Check:  testAccidentityGroupPoliciesCheckAttrs(),
			},
		},
	})
}

func TestAccidentityGroupPoliciesUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testProviders,
		CheckDestroy: testAccCheckidentityGroupPoliciesDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccidentityGroupPoliciesConfig(),
				Check:  testAccidentityGroupPoliciesCheckAttrs(),
			},
			{
				Config: testAccidentityGroupPoliciesConfigUpdate(),
				Check: resource.ComposeTestCheckFunc(
					testAccidentityGroupPoliciesCheckAttrs(),
					resource.TestCheckResourceAttr("vault_identity_group_policies.policies", "policies.#", "2"),
					resource.TestCheckResourceAttr("vault_identity_group_policies.policies", "policies.326271447", "dev"),
					resource.TestCheckResourceAttr("vault_identity_group_policies.policies", "policies.1785148924", "test"),
				),
			},
		},
	})
}

func testAccCheckidentityGroupPoliciesDestroy(s *terraform.State) error {
	client := testProvider.Meta().(*api.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vault_identity_group" {
			continue
		}
		secret, err := client.Logical().Read(identityGroupIDPath(rs.Primary.ID))
		if err != nil {
			return fmt.Errorf("error checking for identity group %q: %s", rs.Primary.ID, err)
		}
		if secret != nil {
			return fmt.Errorf("identity group role %q still exists", rs.Primary.ID)
		}
	}
	return nil
}

func testAccidentityGroupPoliciesCheckAttrs() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState := s.Modules[0].Resources["vault_identity_group_policies.policies"]
		if resourceState == nil {
			return fmt.Errorf("resource not found in state")
		}

		instanceState := resourceState.Primary
		if instanceState == nil {
			return fmt.Errorf("resource not found in state")
		}

		id := instanceState.ID

		path := identityGroupIDPath(id)
		client := testProvider.Meta().(*api.Client)
		resp, err := client.Logical().Read(path)
		if err != nil {
			return fmt.Errorf("%q doesn't exist", path)
		}

		attrs := map[string]string{
			"group_id":   "id",
			"group_name": "name",
			"policies":   "policies",
		}
		for stateAttr, apiAttr := range attrs {
			if resp.Data[apiAttr] == nil && instanceState.Attributes[stateAttr] == "" {
				continue
			}
			var match bool
			switch resp.Data[apiAttr].(type) {
			case json.Number:
				apiData, err := resp.Data[apiAttr].(json.Number).Int64()
				if err != nil {
					return fmt.Errorf("expected API field %s to be an int, was %q", apiAttr, resp.Data[apiAttr])
				}
				stateData, err := strconv.ParseInt(instanceState.Attributes[stateAttr], 10, 64)
				if err != nil {
					return fmt.Errorf("expected state field %s to be an int, was %q", stateAttr, instanceState.Attributes[stateAttr])
				}
				match = apiData == stateData
			case bool:
				if _, ok := resp.Data[apiAttr]; !ok && instanceState.Attributes[stateAttr] == "" {
					match = true
				} else {
					stateData, err := strconv.ParseBool(instanceState.Attributes[stateAttr])
					if err != nil {
						return fmt.Errorf("expected state field %s to be a bool, was %q", stateAttr, instanceState.Attributes[stateAttr])
					}
					match = resp.Data[apiAttr] == stateData
				}
			case []interface{}:
				apiData := resp.Data[apiAttr].([]interface{})
				length := instanceState.Attributes[stateAttr+".#"]
				if length == "" {
					if len(resp.Data[apiAttr].([]interface{})) != 0 {
						return fmt.Errorf("expected state field %s to have %d entries, had 0", stateAttr, len(apiData))
					}
					match = true
				} else {
					count, err := strconv.Atoi(length)
					if err != nil {
						return fmt.Errorf("expected %s.# to be a number, got %q", stateAttr, instanceState.Attributes[stateAttr+".#"])
					}
					if count != len(apiData) {
						return fmt.Errorf("expected %s to have %d entries in state, has %d", stateAttr, len(apiData), count)
					}

					for i := 0; i < count; i++ {
						found := false
						for stateKey, stateValue := range instanceState.Attributes {
							if strings.HasPrefix(stateKey, stateAttr) {
								if apiData[i] == stateValue {
									found = true
								}
							}
						}
						if !found {
							return fmt.Errorf("Expected item %d of %s (%s in state) of %q to be in state but wasn't", i, apiAttr, stateAttr, apiData[i])
						}
					}
					match = true
				}
			default:
				match = resp.Data[apiAttr] == instanceState.Attributes[stateAttr]
			}
			if !match {
				return fmt.Errorf("expected %s (%s in state) of %q to be %q, got %q", apiAttr, stateAttr, path, instanceState.Attributes[stateAttr], resp.Data[apiAttr])
			}
		}
		return nil
	}
}

func testAccidentityGroupPoliciesConfig() string {
	return fmt.Sprintf(`
resource "vault_identity_group" "group" {
  external_policies = true
}

resource "vault_identity_group_policies" "policies" {
  group_id = "${vault_identity_group.group.id}"
  policies = ["test"]
}`)
}

func testAccidentityGroupPoliciesConfigUpdate() string {
	return fmt.Sprintf(`
resource "vault_identity_group" "group" {
  external_policies = true
}

resource "vault_identity_group_policies" "policies" {
  group_id = "${vault_identity_group.group.id}"
  policies = ["dev", "test"]
}`)
}