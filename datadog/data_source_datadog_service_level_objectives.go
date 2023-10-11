package datadog

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/terraform-providers/terraform-provider-datadog/datadog/internal/utils"
)

func dataSourceDatadogServiceLevelObjectives() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to retrieve information about multiple SLOs for use in other resources.",
		ReadContext: dataSourceDatadogServiceLevelObjectivesRead,

		SchemaFunc: func() map[string]*schema.Schema {
			return map[string]*schema.Schema{
				"query": {
					Description: "The query string to filter results based on SLO names. Some examples of queries include service:<service-name> and <slo-name>.",
					Type:        schema.TypeString,
					Optional:    true,
				},

				// Computed values
				"slos": {
					Description: "List of SLOs",
					Type:        schema.TypeList,
					Computed:    true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"id": {
								Description: "ID of the Datadog service level objective",
								Type:        schema.TypeString,
								Computed:    true,
							},
							"name": {
								Description: "Name of the Datadog service level objective",
								Type:        schema.TypeString,
								Computed:    true,
							},
							"type": {
								Description: "The type of the service level objective. The mapping from these types to the types found in the Datadog Web UI can be found in the Datadog API [documentation page](https://docs.datadoghq.com/api/v1/service-level-objectives/#create-a-slo-object). Available options to choose from are: `metric` and `monitor`.",
								Type:        schema.TypeString,
								Computed:    true,
							},
						},
					},
				},
			}
		},
	}
}

func dataSourceDatadogServiceLevelObjectivesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConf := meta.(*ProviderConfiguration)
	apiInstances := providerConf.DatadogApiInstances
	auth := providerConf.Auth

	var queryPtr *string

	reqParams := datadogV1.NewSearchSLOOptionalParameters()
	if v, ok := d.GetOk("query"); ok {
		query := v.(string)
		queryPtr = &query
		reqParams.WithQuery(query)
	}

	reqParams.WithPageSize(100)

	pageNumber := int64(0)
	numberOfSLOs := int64(0)

	allSlosPages := make([][]map[string]interface{}, 0)

	diags := diag.Diagnostics{}

	for {
		reqParams.WithPageNumber(pageNumber)
		slosResp, httpresp, err := apiInstances.GetServiceLevelObjectivesApiV1().SearchSLO(auth, *reqParams)
		if err != nil {
			return utils.TranslateClientErrorDiag(err, httpresp, "error querying service level objectives")
		}

		slosPage := make([]map[string]interface{}, 0, len(slosResp.GetData().Attributes.Slos))
		for _, slo := range slosResp.GetData().Attributes.Slos {
			if err := utils.CheckForUnparsed(slo); err != nil {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  fmt.Sprintf("skipping service level objective with id: %s", *slo.GetData().Id),
					Detail:   fmt.Sprintf("service level objective contains unparsed object: %v", err),
				})
				continue
			}

			slosPage = append(slosPage, map[string]interface{}{
				"id":   *slo.GetData().Id,
				"name": slo.GetData().Attributes.GetName(),
				"type": slo.GetData().Attributes.GetSloType(),
			})
		}

		allSlosPages = append(allSlosPages, slosPage)
		numberOfSLOs += int64(len(slosPage))

		if *slosResp.GetMeta().Pagination.LastNumber <= pageNumber {
			break
		}
		pageNumber++
	}

	// Flatten all pages of slos into a single list
	slos := make([]map[string]interface{}, 0, numberOfSLOs)
	for _, slosPage := range allSlosPages {
		slos = append(slos, slosPage...)
	}

	if err := d.Set("slos", slos); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(computeSLOsDataSourceID(queryPtr))

	return diags
}

func computeSLOsDataSourceID(query *string) string {
	// Key for hashing
	h := sha256.New()
	log.Println("HASHKEY", query)
	h.Write([]byte(*query))

	return fmt.Sprintf("%x", h.Sum(nil))
}
