package datadog

import (
	"context"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/terraform-providers/terraform-provider-datadog/datadog/internal/utils"
	"github.com/terraform-providers/terraform-provider-datadog/datadog/internal/validators"
)

func resourceDatadogMonitorConfigPolicy() *schema.Resource {
	return &schema.Resource{
		Description:   "Provides a Datadog monitor config policy resource. This can be used to create and manage Datadog monitor config policies.",
		CreateContext: resourceDatadogMonitorConfigPolicyCreate,
		ReadContext:   resourceDatadogMonitorConfigPolicyRead,
		UpdateContext: resourceDatadogMonitorConfigPolicyUpdate,
		DeleteContext: resourceDatadogMonitorConfigPolicyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"policy_type": {
				Description:      "The monitor config policy type",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validators.ValidateEnumValue(datadogV2.NewMonitorConfigPolicyTypeFromValue),
			},
			"tag_policy": {
				Description: "Config for a tag policy. Only set if `policy_type` is `tag`.",
				Type:        schema.TypeList,
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
	}
}

func buildCreateRequestPolicy(d *schema.ResourceData, policyType datadogV2.MonitorConfigPolicyType) *datadogV2.MonitorConfigPolicyPolicyCreateRequest {
	if policyType == datadogV2.MONITORCONFIGPOLICYTYPE_TAG {
		tagKey := d.Get("tag_key").(string)
		tagKeyRequired := d.Get("tag_key_required").(bool)
		var validTagValues []string
		for _, s := range d.Get("valid_tag_values").([]interface{}) {
			validTagValues = append(validTagValues, s.(string))
		}
		return &datadogV2.MonitorConfigPolicyPolicyCreateRequest{
			MonitorConfigPolicyTagPolicyCreateRequest: &datadogV2.MonitorConfigPolicyTagPolicyCreateRequest{
				TagKey:         tagKey,
				TagKeyRequired: tagKeyRequired,
				ValidTagValues: validTagValues,
			}}
	}
	return nil
}

func buildUpdateRequestPolicy(d *schema.ResourceData, policyType datadogV2.MonitorConfigPolicyType) *datadogV2.MonitorConfigPolicyPolicy {
	if policyType == datadogV2.MONITORCONFIGPOLICYTYPE_TAG {
		tagPolicy := datadogV2.NewMonitorConfigPolicyTagPolicy()

		if attr, ok := d.GetOk("tag_key"); ok {
			tagPolicy.SetTagKey(attr.(string))
		}
		if attr, ok := d.GetOk("tag_key"); ok {
			tagPolicy.SetTagKeyRequired(attr.(bool))
		}
		if attr, ok := d.GetOk("valid_tag_values"); ok {
			var validTagValues []string
			for _, s := range attr.([]interface{}) {
				validTagValues = append(validTagValues, s.(string))
			}
		}
		return &datadogV2.MonitorConfigPolicyPolicy{MonitorConfigPolicyTagPolicy: tagPolicy}
	}
	return nil
}

func buildMonitorConfigPolicyCreateV2Struct(d *schema.ResourceData) *datadogV2.MonitorConfigPolicyCreateRequest {
	policyType, _ := datadogV2.NewMonitorConfigPolicyTypeFromValue(d.Get("policy_type").(string))

	return datadogV2.NewMonitorConfigPolicyCreateRequest(
		datadogV2.MonitorConfigPolicyCreateData{
			Attributes: datadogV2.MonitorConfigPolicyAttributeCreateRequest{
				PolicyType: *policyType,
				Policy:     *buildCreateRequestPolicy(d, *policyType),
			},
			Type: datadogV2.MONITORCONFIGPOLICYRESOURCETYPE_MONITOR_CONFIG_POLICY,
		})
}

func buildMonitorConfigPolicyUpdateV2Struct(d *schema.ResourceData) *datadogV2.MonitorConfigPolicyEditRequest {
	id := d.Id()
	policyType, _ := datadogV2.NewMonitorConfigPolicyTypeFromValue(d.Get("policy_type").(string))
	return datadogV2.NewMonitorConfigPolicyEditRequest(
		datadogV2.MonitorConfigPolicyEditData{
			Attributes: datadogV2.MonitorConfigPolicyAttributeEditRequest{
				Policy:     *buildUpdateRequestPolicy(d, *policyType),
				PolicyType: *policyType,
			},
			Id:   id,
			Type: datadogV2.MONITORCONFIGPOLICYRESOURCETYPE_MONITOR_CONFIG_POLICY,
		},
	)
}

func resourceDatadogMonitorConfigPolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConf := meta.(*ProviderConfiguration)
	apiInstances := providerConf.DatadogApiInstances
	auth := providerConf.Auth

	var monitorConfigPolicyResponse datadogV2.MonitorConfigPolicyResponse
	monitorConfigPolicyResponse, httpresp, err := apiInstances.GetMonitorsApiV2().GetMonitorConfigPolicy(auth, d.Id())
	if err != nil {
		if httpresp != nil && httpresp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return utils.TranslateClientErrorDiag(err, httpresp, "error getting monitor config policy")
	}

	return updateMonitorConfigPolicyState(d, monitorConfigPolicyResponse.Data)
}

func resourceDatadogMonitorConfigPolicyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConf := meta.(*ProviderConfiguration)
	apiInstances := providerConf.DatadogApiInstances
	auth := providerConf.Auth

	m := buildMonitorConfigPolicyCreateV2Struct(d)
	mCreated, httpResponse, err := apiInstances.GetMonitorsApiV2().CreateMonitorConfigPolicy(auth, *m)
	if err != nil {
		return utils.TranslateClientErrorDiag(err, httpResponse, "error creating monitor config policy")
	}
	if err := utils.CheckForUnparsed(m); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(*mCreated.Data.Id)

	return updateMonitorConfigPolicyState(d, mCreated.Data)
}

func resourceDatadogMonitorConfigPolicyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConf := meta.(*ProviderConfiguration)
	apiInstances := providerConf.DatadogApiInstances
	auth := providerConf.Auth

	monitorConfigPolicy := buildMonitorConfigPolicyUpdateV2Struct(d)

	mUpdated, httpresp, err := apiInstances.GetMonitorsApiV2().UpdateMonitorConfigPolicy(
		auth, monitorConfigPolicy.Data.Id, *monitorConfigPolicy,
	)
	if err != nil {
		return utils.TranslateClientErrorDiag(err, httpresp, "error updating monitor config policy")
	}
	if err := utils.CheckForUnparsed(mUpdated); err != nil {
		return diag.FromErr(err)
	}

	return updateMonitorConfigPolicyState(d, mUpdated.Data)
}

func resourceDatadogMonitorConfigPolicyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConf := meta.(*ProviderConfiguration)
	apiInstances := providerConf.DatadogApiInstances
	auth := providerConf.Auth

	httpresp, err := apiInstances.GetMonitorsApiV2().DeleteMonitorConfigPolicy(auth, d.Id())
	if err != nil {
		return utils.TranslateClientErrorDiag(err, httpresp, "error deleting monitor config policy")
	}

	return nil
}

func updateMonitorConfigPolicyState(d *schema.ResourceData, m *datadogV2.MonitorConfigPolicyResponseData) diag.Diagnostics {
	d.SetId(m.GetId())
	attributes := m.GetAttributes()
	d.Set("policy_type", attributes.GetPolicyType())
	if attributes.GetPolicyType() == datadogV2.MONITORCONFIGPOLICYTYPE_TAG {
		d.Set("tag_policy", map[string]interface{}{
			"tag_key":          attributes.Policy.MonitorConfigPolicyTagPolicy.GetTagKey(),
			"tag_key_required": attributes.Policy.MonitorConfigPolicyTagPolicy.GetTagKeyRequired(),
			"valid_tag_values": attributes.Policy.MonitorConfigPolicyTagPolicy.GetValidTagValues(),
		})
	}
	return nil
}