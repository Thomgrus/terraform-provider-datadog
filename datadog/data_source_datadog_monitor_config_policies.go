package datadog

import (
	"context"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/terraform-providers/terraform-provider-datadog/datadog/internal/utils"
	"github.com/terraform-providers/terraform-provider-datadog/datadog/internal/validators"
)

func dataSourceDatadogMonitorConfigPolicies() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to list several existing monitor config policies for use in other resources.",
		ReadContext: dataSourceDatadogMonitorConfigPoliciesRead,
		Schema: map[string]*schema.Schema{
			// Computed values
			"monitor_config_policies": {
				Description: "List of monitor config policies",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "ID of the monitor config policy",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						"policy_type": {
							Description:      "The monitor config policy type",
							Type:             schema.TypeString,
							Computed:         true,
							ValidateDiagFunc: validators.ValidateEnumValue(datadogV2.NewMonitorConfigPolicyTypeFromValue),
						},
						"tag_policy": {
							Description: "Config for a tag policy. Only set if `policy_type` is `tag`.",
							Type:        schema.TypeList,
							Computed:    true,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"tag_key": {
										Type:        schema.TypeString,
										Description: "The key of the tag",
									},
									"tag_key_required": {
										Type:        schema.TypeBool,
										Description: "If a tag key is required for monitor creation",
									},
									"valid_tag_values": {
										Type:        schema.TypeString,
										Description: "Valid values for the tag",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceDatadogMonitorConfigPoliciesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConf := meta.(*ProviderConfiguration)
	apiInstances := providerConf.DatadogApiInstances
	auth := providerConf.Auth

	monitorConfigPolicies, httpresp, err := apiInstances.GetMonitorsApiV2().ListMonitorConfigPolicies(auth)
	if err != nil {
		return utils.TranslateClientErrorDiag(err, httpresp, "error querying monitor config policies")
	}
	if err := utils.CheckForUnparsed(monitorConfigPolicies); err != nil {
		return diag.FromErr(err)
	}

	tfMonitorConfigPolicies := make([]map[string]interface{}, len(monitorConfigPolicies.Data))
	for i, mcp := range monitorConfigPolicies.Data {
		attributes := mcp.GetAttributes()
		tfMonitorConfigPolicies[i] = map[string]interface{}{
			"id":          mcp.GetId(),
			"policy_type": attributes.GetPolicyType(),
		}

		policy := attributes.GetPolicy()
		if policy.MonitorConfigPolicyTagPolicy != nil {
			tfMonitorConfigPolicies[i]["tag_policy"] = map[string]interface{}{
				"tag_key":          policy.MonitorConfigPolicyTagPolicy.GetTagKey(),
				"tag_key_required": policy.MonitorConfigPolicyTagPolicy.GetTagKeyRequired(),
				"valid_tag_values": policy.MonitorConfigPolicyTagPolicy.GetValidTagValues(),
			}
		}
	}

	d.SetId("monitor-config-policies")
	d.Set("monitor_config_policies", tfMonitorConfigPolicies)

	return nil
}