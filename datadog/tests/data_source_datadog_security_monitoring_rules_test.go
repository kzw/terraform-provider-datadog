package test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/terraform-providers/terraform-provider-datadog/datadog"
)

const tfSecurityRulesSource = "data.datadog_security_monitoring_rules.acceptance_test"

var allRules *[]datadogV2.SecurityMonitoringRuleResponse

func TestAccDatadogSecurityMonitoringRuleDatasource(t *testing.T) {
	ctx, accProviders := testAccProviders(context.Background(), t)
	ruleName := uniqueEntityName(ctx, t)
	accProvider := testAccProvider(t, accProviders)

	securityMonitoringCreatedConfig := testAccCheckDatadogSecurityMonitoringCreatedConfig(ruleName)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: accProviders,
		CheckDestroy:      testAccCheckDatadogSecurityMonitoringRuleDestroy(accProvider),
		Steps: []resource.TestStep{
			{
				// Create a rule to make sure we have at least one non-default rule
				Config: testAccCheckDatadogSecurityMonitoringCreatedConfig(ruleName),
			},
			// Disable the test for now as the result is too huge
			//{
			//	Config: testAccDataSourceSecurityMonitoringRuleNoFilter(securityMonitoringCreatedConfig),
			//	Check: resource.ComposeTestCheckFunc(
			//		securityMonitoringCheckRuleCountNoFilter(accProvider),
			//	),
			//},
			{
				Config: testAccDataSourceSecurityMonitoringRuleNameFilter(securityMonitoringCreatedConfig, ruleName),
				Check: resource.ComposeTestCheckFunc(
					securityMonitoringCheckRuleCountNameFilter(accProvider, ruleName),
				),
			},
			{
				Config: testAccDataSourceSecurityMonitoringRuleTagsFilter(securityMonitoringCreatedConfig),
				Check: resource.ComposeTestCheckFunc(
					securityMonitoringCheckRuleCountTagsFilter(accProvider, "i:tomato"),
				),
			},
			// Disable the test for now as the result is too huge
			//{
			//	Config: testAccDataSourceSecurityMonitoringRuleDefaultFilter(ruleName),
			//	Check: resource.ComposeTestCheckFunc(
			//		securityMonitoringCheckRuleCountDefaultFilter(accProvider, true),
			//	),
			//},
			{
				Config: testAccDataSourceSecurityMonitoringRuleUserFilter(securityMonitoringCreatedConfig),
				Check: resource.ComposeTestCheckFunc(
					securityMonitoringCheckRuleCountDefaultFilter(accProvider, false),
				),
			},
		},
	})
}

//func securityMonitoringCheckRuleCountNoFilter(accProvider func() (*schema.Provider, error)) func(state *terraform.State) error {
//	return func(state *terraform.State) error {
//		provider, _ := accProvider()
//		providerConf := provider.Meta().(*datadog.ProviderConfiguration)
//		auth := providerConf.Auth
//		apiInstances := providerConf.DatadogApiInstances
//
//		rulesResponse, _, err := apiInstances.GetSecurityMonitoringApiV2().ListSecurityMonitoringRules(auth,
//			*datadogV2.NewListSecurityMonitoringRulesOptionalParameters().
//				WithPageNumber(0).
//				WithPageSize(1000))
//		if err != nil {
//			return err
//		}
//		return securityMonitoringCheckRuleCount(state, len(rulesResponse.Data))
//	}
//}

func securityMonitoringCheckRuleCountNameFilter(accProvider func() (*schema.Provider, error), name string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		if allRules == nil {
			err := getAllSecurityMonitoringRules(accProvider)
			if err != nil {
				return err
			}
		}

		ruleCount := 0
		for _, rule := range *allRules {
			if strings.Contains(rule.GetName(), name) {
				ruleCount++
			}
		}

		return securityMonitoringCheckRuleCount(state, ruleCount)
	}
}

func securityMonitoringCheckRuleCountTagsFilter(accProvider func() (*schema.Provider, error), filterTag string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		if allRules == nil {
			err := getAllSecurityMonitoringRules(accProvider)
			if err != nil {
				return err
			}
		}

		ruleCount := 0
		for _, rule := range *allRules {
			for _, tag := range rule.GetTags() {
				if strings.Contains(tag, filterTag) {
					ruleCount++
				}
			}
		}
		return securityMonitoringCheckRuleCount(state, ruleCount)
	}
}

func securityMonitoringCheckRuleCountDefaultFilter(accProvider func() (*schema.Provider, error), isDefault bool) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		if allRules == nil {
			err := getAllSecurityMonitoringRules(accProvider)
			if err != nil {
				return err
			}
		}

		ruleCount := 0
		for _, rule := range *allRules {
			if rule.GetIsDefault() == isDefault {
				ruleCount++
			}
		}
		return securityMonitoringCheckRuleCount(state, ruleCount)
	}

}

func securityMonitoringCheckRuleCount(state *terraform.State, responseRuleCount int) error {
	resourceAttributes := state.RootModule().Resources[tfSecurityRulesSource].Primary.Attributes
	ruleIDCount, _ := strconv.Atoi(resourceAttributes["rule_ids.#"])
	rulesCount, _ := strconv.Atoi(resourceAttributes["rules.#"])

	if rulesCount != responseRuleCount || ruleIDCount != responseRuleCount {
		return fmt.Errorf("expected %d rules got %d rules and %d rule ids",
			responseRuleCount, rulesCount, ruleIDCount)
	}
	return nil
}

//func testAccDataSourceSecurityMonitoringRuleNoFilter(existingRuleConfig string) string {
//	return existingRuleConfig + `
//data "datadog_security_monitoring_rules" "acceptance_test" {
//}
//`
//}

func getAllSecurityMonitoringRules(accProvider func() (*schema.Provider, error)) error {
	provider, _ := accProvider()
	providerConf := provider.Meta().(*datadog.ProviderConfiguration)
	auth := providerConf.Auth
	apiInstances := providerConf.DatadogApiInstances

	rulesResponse, _, err := apiInstances.GetSecurityMonitoringApiV2().ListSecurityMonitoringRules(auth,
		*datadogV2.NewListSecurityMonitoringRulesOptionalParameters().WithPageSize(1000))
	if err != nil {
		return err
	}

	allRules = &rulesResponse.Data
	return nil
}

func testAccDataSourceSecurityMonitoringRuleNameFilter(existingRuleConfig, ruleName string) string {
	return existingRuleConfig + fmt.Sprintf(`
data "datadog_security_monitoring_rules" "acceptance_test" {
    name_filter = "%s"
}
`, ruleName)
}

func testAccDataSourceSecurityMonitoringRuleTagsFilter(existingRuleConfig string) string {
	return existingRuleConfig + `
data "datadog_security_monitoring_rules" "acceptance_test" {
    tags_filter = ["i:tomato"]
}
`
}

//func testAccDataSourceSecurityMonitoringRuleDefaultFilter(ruleName string) string {
//	return testAccCheckDatadogSecurityMonitoringCreatedConfig(ruleName) + `
//data "datadog_security_monitoring_rules" "acceptance_test" {
//	default_only_filter = true
//}
//`
//}

func testAccDataSourceSecurityMonitoringRuleUserFilter(existingRuleConfig string) string {
	return existingRuleConfig + `
data "datadog_security_monitoring_rules" "acceptance_test" {
	user_only_filter = true
}
`
}
