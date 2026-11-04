package provider

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const testAccSubBalancerRemarkPrefix = "acc-sub-balancer"

// testAccCheckSubBalancerDestroyed also catches balancers that never reached
// state, e.g. a create that failed after the panel stored the row.
func testAccCheckSubBalancerDestroyed(state *terraform.State) error {
	client, err := testAccClientFromEnv()
	if err != nil {
		return fmt.Errorf("client init failed: %w", err)
	}
	balancers, err := client.ListSubBalancers(context.Background())
	if err != nil {
		return err
	}
	for _, b := range balancers {
		if strings.HasPrefix(b.Remark, testAccSubBalancerRemarkPrefix) {
			return fmt.Errorf("sub balancer %d (%q) still exists", b.Id, b.Remark)
		}
	}
	return testAccCheckInboundDestroyed(state)
}

func testAccSubBalancerInboundsConfig() string {
	return `
resource "threexui_inbound" "sb_a" {
  port     = 26301
  protocol = "vless"
  remark   = "acc-sub-balancer-inbound-a"
  enable   = true
  vless_settings {
    decryption = "none"
  }
}

resource "threexui_inbound" "sb_b" {
  port     = 26302
  protocol = "vless"
  remark   = "acc-sub-balancer-inbound-b"
  enable   = true
  vless_settings {
    decryption = "none"
  }
}
`
}

// testAccCheckSubBalancerOnPanel reads the balancer back from the panel, so
// the checks see what was stored rather than what the provider put in state.
func testAccCheckSubBalancerOnPanel(name string, check func(s *terraform.State, b *SubBalancer) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("%s not found in state", name)
		}
		id, err := parseSubBalancerID(rs.Primary.ID)
		if err != nil {
			return err
		}
		client, err := testAccClientFromEnv()
		if err != nil {
			return fmt.Errorf("client init failed: %w", err)
		}
		b, err := client.GetSubBalancer(context.Background(), id)
		if err != nil {
			return err
		}
		if b == nil {
			return fmt.Errorf("sub balancer %d not found on the panel", id)
		}
		return check(s, b)
	}
}

func testAccInboundIDs(s *terraform.State, names ...string) ([]int64, error) {
	ids := make([]int64, 0, len(names))
	for _, name := range names {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return nil, fmt.Errorf("%s not found in state", name)
		}
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids, nil
}

func testAccSubBalancerHasInbounds(names ...string) func(*terraform.State, *SubBalancer) error {
	return func(s *terraform.State, b *SubBalancer) error {
		want, err := testAccInboundIDs(s, names...)
		if err != nil {
			return err
		}
		got := slices.Clone(b.InboundIds)
		slices.Sort(got)
		if !slices.Equal(got, want) {
			return fmt.Errorf("panel inboundIds = %v, want %v", got, want)
		}
		return nil
	}
}

func TestAccSubBalancerBasic(t *testing.T) {
	requireMinVersion(t, "v3.7.0")
	name := "threexui_sub_balancer.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckSubBalancerDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + testAccSubBalancerInboundsConfig() + `
resource "threexui_sub_balancer" "test" {
  remark      = "acc-sub-balancer-basic"
  inbound_ids = [threexui_inbound.sb_a.id, threexui_inbound.sb_b.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(name, "id"),
					resource.TestCheckResourceAttr(name, "remark", "acc-sub-balancer-basic"),
					resource.TestCheckResourceAttr(name, "strategy", "random"),
					resource.TestCheckResourceAttr(name, "sort_order", "1"),
					resource.TestCheckResourceAttr(name, "enabled", "true"),
					resource.TestCheckResourceAttr(name, "inbound_ids.#", "2"),
					resource.TestCheckNoResourceAttr(name, "member_weights.%"),
					testAccCheckSubBalancerOnPanel(name, testAccSubBalancerHasInbounds("threexui_inbound.sb_a", "threexui_inbound.sb_b")),
					testAccCheckSubBalancerOnPanel(name, func(_ *terraform.State, b *SubBalancer) error {
						if !b.Enabled || b.Strategy != "random" || b.SortOrder != 1 {
							return fmt.Errorf("panel row = %+v", b)
						}
						return nil
					}),
				),
			},
			// Members follow the subscription order, so inbound_ids is a set:
			// listing them the other way round is not a change.
			{
				Config: testAccProviderConfig() + testAccSubBalancerInboundsConfig() + `
resource "threexui_sub_balancer" "test" {
  remark      = "acc-sub-balancer-basic"
  inbound_ids = [threexui_inbound.sb_b.id, threexui_inbound.sb_a.id]
}
`,
				PlanOnly: true,
			},
			{
				Config: testAccProviderConfig() + testAccSubBalancerInboundsConfig() + `
resource "threexui_sub_balancer" "test" {
  remark      = "acc-sub-balancer-basic-2"
  strategy    = "roundRobin"
  sort_order  = 3
  enabled     = false
  inbound_ids = [threexui_inbound.sb_a.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "remark", "acc-sub-balancer-basic-2"),
					resource.TestCheckResourceAttr(name, "strategy", "roundRobin"),
					resource.TestCheckResourceAttr(name, "sort_order", "3"),
					resource.TestCheckResourceAttr(name, "enabled", "false"),
					resource.TestCheckResourceAttr(name, "inbound_ids.#", "1"),
					testAccCheckSubBalancerOnPanel(name, testAccSubBalancerHasInbounds("threexui_inbound.sb_a")),
					testAccCheckSubBalancerOnPanel(name, func(_ *terraform.State, b *SubBalancer) error {
						if b.Enabled || b.Strategy != "roundRobin" || b.SortOrder != 3 || b.Remark != "acc-sub-balancer-basic-2" {
							return fmt.Errorf("panel row = %+v", b)
						}
						return nil
					}),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSubBalancerMemberWeights(t *testing.T) {
	requireMinVersion(t, "v3.8.0")
	name := "threexui_sub_balancer.test"
	balancer := func(weights string) string {
		return testAccProviderConfig() + testAccSubBalancerInboundsConfig() + fmt.Sprintf(`
resource "threexui_sub_balancer" "test" {
  remark      = "acc-sub-balancer-weights"
  strategy    = "leastLoad"
  inbound_ids = [threexui_inbound.sb_a.id, threexui_inbound.sb_b.id]
  %s
}
`, weights)
	}
	panelWeights := func(want map[string]float64) resource.TestCheckFunc {
		return testAccCheckSubBalancerOnPanel(name, func(s *terraform.State, b *SubBalancer) error {
			got := map[string]float64{}
			for id, weight := range b.MemberWeights {
				for _, inbound := range []string{"threexui_inbound.sb_a", "threexui_inbound.sb_b"} {
					if s.RootModule().Resources[inbound].Primary.ID == strconv.FormatInt(id, 10) {
						got[inbound] = weight
					}
				}
			}
			if len(got) != len(want) || len(b.MemberWeights) != len(want) {
				return fmt.Errorf("panel memberWeights = %v, want %v", b.MemberWeights, want)
			}
			for inbound, weight := range want {
				if got[inbound] != weight {
					return fmt.Errorf("panel weight of %s = %v, want %v", inbound, got[inbound], weight)
				}
			}
			return nil
		})
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckSubBalancerDestroyed,
		Steps: []resource.TestStep{
			// Known ids: the panel would silently drop the weight, so plan fails.
			{
				Config: testAccProviderConfig() + `
resource "threexui_sub_balancer" "test" {
  remark         = "acc-sub-balancer-weights"
  strategy       = "leastLoad"
  inbound_ids    = [1, 2]
  member_weights = { "3" = 2 }
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Inbound 3 is not a member of this balancer`),
			},
			// Ids of inbounds created in the same apply are unknown at plan time;
			// Terraform validates again during apply, still before the create.
			{
				Config:      balancer(`member_weights = { "999999" = 2 }`),
				ExpectError: regexp.MustCompile(`Inbound 999999 is not a member of this balancer`),
			},
			{
				Config: balancer(`member_weights = { (threexui_inbound.sb_a.id) = 0.3, (threexui_inbound.sb_b.id) = 2.5 }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "member_weights.%", "2"),
					panelWeights(map[string]float64{"threexui_inbound.sb_a": 0.3, "threexui_inbound.sb_b": 2.5}),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// 1/3 has more digits than the float64 the panel stores; it must
			// neither fail the apply nor show a diff afterwards.
			{
				Config: balancer(`member_weights = { (threexui_inbound.sb_a.id) = 0.3, (threexui_inbound.sb_b.id) = 1 / 3 }`),
				Check:  panelWeights(map[string]float64{"threexui_inbound.sb_a": 0.3, "threexui_inbound.sb_b": 1.0 / 3}),
			},
			{
				Config:   balancer(`member_weights = { (threexui_inbound.sb_a.id) = 0.3, (threexui_inbound.sb_b.id) = 1 / 3 }`),
				PlanOnly: true,
			},
			// The panel clears weights the update does not send.
			{
				Config: balancer(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(name, "member_weights.%"),
					panelWeights(map[string]float64{}),
				),
			},
		},
	})
}

// Deleting an inbound on the panel strips its id from every balancer
// (3x-ui-3.11.0/internal/web/service/inbound.go, DelInbound); the refresh has
// to show that, otherwise Terraform believes the balancer is still complete.
func TestAccSubBalancerInboundDeletedOnPanel(t *testing.T) {
	requireMinVersion(t, "v3.7.0")
	testAccPreCheck(t)
	client, err := testAccClientFromEnv()
	if err != nil {
		t.Fatalf("client init failed: %v", err)
	}
	ctx := context.Background()
	external, err := client.AddInbound(ctx, &Inbound{
		Remark: "acc-sub-balancer-external", Enable: true, Port: 26303, Protocol: "vless",
		Settings:       `{"clients":[],"decryption":"none"}`,
		StreamSettings: `{"network":"tcp","security":"none"}`,
		Sniffing:       `{"enabled":false}`,
		TrafficReset:   "never", TrafficResetDay: 1,
	})
	if err != nil {
		t.Fatalf("creating the external inbound: %v", err)
	}
	externalDeleted := false
	t.Cleanup(func() {
		if !externalDeleted {
			_ = client.DeleteInbound(ctx, external.ID)
		}
	})

	name := "threexui_sub_balancer.test"
	config := testAccProviderConfig() + testAccSubBalancerInboundsConfig() + fmt.Sprintf(`
resource "threexui_sub_balancer" "test" {
  remark      = "acc-sub-balancer-cleanup"
  inbound_ids = [threexui_inbound.sb_a.id, %d]
}
`, external.ID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckSubBalancerDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr(name, "inbound_ids.#", "2"),
			},
			{
				PreConfig: func() {
					if err := client.DeleteInbound(ctx, external.ID); err != nil {
						t.Fatalf("deleting the external inbound: %v", err)
					}
					externalDeleted = true
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(name, plancheck.ResourceActionUpdate)},
				},
				Check: resource.TestCheckResourceAttr(name, "inbound_ids.#", "2"),
			},
		},
	})
}

// v3.7.x has the API but not memberWeights: the apply must fail and leave no
// balancer behind (CheckDestroy lists every balancer, tracked or not).
func TestAccSubBalancerMemberWeightsUnsupported(t *testing.T) {
	requireMinVersion(t, "v3.7.0")
	requireBelowVersion(t, "v3.8.0")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckSubBalancerDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + testAccSubBalancerInboundsConfig() + `
resource "threexui_sub_balancer" "test" {
  remark         = "acc-sub-balancer-weights-v37"
  strategy       = "leastLoad"
  inbound_ids    = [threexui_inbound.sb_a.id, threexui_inbound.sb_b.id]
  member_weights = { (threexui_inbound.sb_b.id) = 2 }
}
`,
				ExpectError: regexp.MustCompile(`member_weights requires 3x-ui v3\.8\.0 or later`),
			},
		},
	})
}

func TestAccSubBalancerUnsupportedPanel(t *testing.T) {
	requireBelowVersion(t, "v3.7.0")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + testAccSubBalancerInboundsConfig() + `
resource "threexui_sub_balancer" "test" {
  remark      = "acc-sub-balancer-unsupported"
  inbound_ids = [threexui_inbound.sb_a.id]
}
`,
				ExpectError: regexp.MustCompile(`requires 3x-ui v3\.7\.0 or later`),
			},
		},
	})
}
