package solidserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/url"
	"strconv"
)

func resourceapplicationpool() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceapplicationpoolCreate,
		ReadContext:   resourceapplicationpoolRead,
		UpdateContext: resourceapplicationpoolUpdate,
		DeleteContext: resourceapplicationpoolDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceapplicationpoolImportState,
		},

		Description: heredoc.Doc(`
			Application Pool allows to create and manage a pool that implement a traffic policy.
			Application Pools are groups of nodes serving the same application and monitored by the GSLB(s) DNS servers
		`),

		Schema: map[string]*schema.Schema{
			"application": {
				Type:        schema.TypeString,
				Description: "The name of the application associated to the pool.",
				Required:    true,
				ForceNew:    true,
			},
			"fqdn": {
				Type:        schema.TypeString,
				Description: "The fqdn of the application associated to the pool.",
				Required:    true,
				ForceNew:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the application pool to create.",
				Required:    true,
				ForceNew:    true,
			},
			"ip_version": {
				Type:        schema.TypeString,
				Description: "The IP protocol version used by the application pool to create (Supported: ipv4, ipv6; Default: ipv4).",
				Optional:    true,
				ForceNew:    true,
				Default:     "ipv4",
			},
			"lb_mode": {
				Type:         schema.TypeString,
				Description:  "The load balancing mode of the application pool to create (Supported: weighted,round-robin,latency; Default: round-robin).",
				ValidateFunc: validation.StringInSlice([]string{"weighted", "round-robin", "latency"}, false),
				Optional:     true,
				Default:      "round-robin",
			},
			"affinity": {
				Type:        schema.TypeBool,
				Description: "Enable session affinity for the application pool.",
				Optional:    true,
				Default:     false,
			},
			"affinity_session_duration": {
				Type:        schema.TypeInt,
				Description: "The time each session is maintained in sec (Default: 300).",
				Optional:    true,
				Default:     300,
			},
			"best_active_nodes": {
				Type:         schema.TypeInt,
				Description:  "Number of best active nodes when lb_mode is set to latency.",
				ValidateFunc: validation.IntAtLeast(1),
				Optional:     true,
				Default:      1,
			},
		},
	}
}

func resourceapplicationpoolCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("add_flag", "new_only")
	parameters.Add("name", d.Get("name").(string))
	parameters.Add("appapplication_name", d.Get("application").(string))
	parameters.Add("appapplication_fqdn", d.Get("fqdn").(string))
	parameters.Add("type", d.Get("ip_version").(string))
	parameters.Add("lb_mode", d.Get("lb_mode").(string))

	// Building affinity_state mode
	if d.Get("affinity").(bool) == false {
		parameters.Add("affinity_state", "0")
	} else {
		parameters.Add("affinity_state", "1")
		parameters.Add("affinity_session_time", strconv.Itoa(d.Get("affinity_session_duration").(int)))
	}

	if d.Get("lb_mode").(string) == "latency" {
		parameters.Add("best_active_nodes", strconv.Itoa(d.Get("best_active_nodes").(int)))
	}

	if s.Version < 710 {
		// Reporting a failure
		return diag.Errorf("Object not supported in this SOLIDserver version")
	}

	// Sending creation request
	resp, body, err := s.Request("post", "rest/app_pool_add", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			if oid, oidExist := buf[0]["ret_oid"].(string); oidExist {
				tflog.Debug(ctx, fmt.Sprintf("Created application pool (oid): %s\n", oid))
				d.SetId(oid)
				return nil
			}
		}

		// Reporting a failure
		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				return diag.Errorf("Unable to create application pool: %s (%s)", d.Get("name").(string), errMsg)
			}
		}

		return diag.Errorf("Unable to create application pool: %s\n", d.Get("name").(string))
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourceapplicationpoolUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("apppool_id", d.Id())
	parameters.Add("add_flag", "edit_only")
	parameters.Add("name", d.Get("name").(string))
	parameters.Add("appapplication_name", d.Get("application").(string))
	parameters.Add("appapplication_fqdn", d.Get("fqdn").(string))
	parameters.Add("type", d.Get("ip_version").(string))
	parameters.Add("lb_mode", d.Get("lb_mode").(string))

	// Building affinity_state mode
	if d.Get("affinity").(bool) == false {
		parameters.Add("affinity_state", "0")
	} else {
		parameters.Add("affinity_state", "1")
		parameters.Add("affinity_session_time", strconv.Itoa(d.Get("affinity_session_duration").(int)))
	}

	if d.Get("lb_mode").(string) == "latency" {
		parameters.Add("best_active_nodes", strconv.Itoa(d.Get("best_active_nodes").(int)))
	}

	if s.Version < 710 {
		// Reporting a failure
		return diag.Errorf("Object not supported in this SOLIDserver version")
	}

	// Sending the update request
	resp, body, err := s.Request("put", "rest/app_pool_add", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			if oid, oidExist := buf[0]["ret_oid"].(string); oidExist {
				tflog.Debug(ctx, fmt.Sprintf("Updated application pool (oid): %s\n", oid))
				d.SetId(oid)
				return nil
			}
		}

		// Reporting a failure
		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				return diag.Errorf("Unable to update application pool: %s (%s)", d.Get("name").(string), errMsg)
			}
		}

		return diag.Errorf("Unable to update application pool: %s\n", d.Get("name").(string))
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourceapplicationpoolDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("apppool_id", d.Id())

	if s.Version < 710 {
		// Reporting a failure
		return diag.Errorf("Object not supported in this SOLIDserver version")
	}

	// Sending the deletion request
	resp, body, err := s.Request("delete", "rest/app_pool_delete", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode != 200 && resp.StatusCode != 204 {
			// Reporting a failure
			if len(buf) > 0 {
				if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
					return diag.Errorf("Unable to delete application pool: %s (%s)", d.Get("name").(string), errMsg)
				}
			}

			return diag.Errorf("Unable to delete application pool: %s", d.Get("name").(string))
		}

		// Log deletion
		tflog.Debug(ctx, fmt.Sprintf("Deleted application (oid) pool: %s\n", d.Id()))

		// Unset local ID
		d.SetId("")

		// Reporting a success
		return nil
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourceapplicationpoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("apppool_id", d.Id())

	if s.Version < 710 {
		// Reporting a failure
		return diag.Errorf("Object not supported in this SOLIDserver version")
	}

	// Sending the read request
	resp, body, err := s.Request("get", "rest/app_pool_info", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode == 200 && len(buf) > 0 {
			d.Set("name", buf[0]["apppool_name"].(string))
			d.Set("application", buf[0]["appapplication_name"].(string))
			d.Set("fqdn", buf[0]["appapplication_fqdn"].(string))
			d.Set("lb_mode", buf[0]["apppool_lb_mode"].(string))

			// Updating affinity_state mode
			if buf[0]["apppool_affinity_state"].(string) == "0" {
				d.Set("affinity", false)
			} else {
				d.Set("affinity", true)

				sessionTime, _ := strconv.Atoi(buf[0]["apppool_affinity_session_time"].(string))
				d.Set("affinity_session_duration", sessionTime)
			}

			// Updating best active nodes value
			if buf[0]["apppool_best_active_nodes"].(string) != "" {
				bestActiveNodes, _ := strconv.Atoi(buf[0]["apppool_best_active_nodes"].(string))
				d.Set("best_active_nodes", bestActiveNodes)
			}

			return nil
		}

		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				// Log the error
				tflog.Debug(ctx, fmt.Sprintf("Unable to find application pool: %s (%s)\n", d.Get("name"), errMsg))
			}
		} else {
			// Log the error
			tflog.Debug(ctx, fmt.Sprintf("Unable to find application pool (oid): %s\n", d.Id()))
		}

		// Do not unset the local ID to avoid inconsistency

		// Reporting a failure
		return diag.Errorf("Unable to find application pool: %s\n", d.Get("name").(string))
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourceapplicationpoolImportState(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("apppool_id", d.Id())

	if s.Version < 710 {
		// Reporting a failure
		return nil, fmt.Errorf("SOLIDServer - Object not supported in this SOLIDserver version")
	}

	// Sending the read request
	resp, body, err := s.Request("get", "rest/app_pool_info", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode == 200 && len(buf) > 0 {
			d.Set("name", buf[0]["apppool_name"].(string))
			d.Set("application", buf[0]["appapplication_name"].(string))
			d.Set("fqdn", buf[0]["appapplication_fqdn"].(string))
			d.Set("lb_mode", buf[0]["apppool_lb_mode"].(string))

			// Updating affinity_state mode
			if buf[0]["apppool_affinity_state"].(string) == "0" {
				d.Set("affinity_state", false)
			} else {
				d.Set("affinity_state", true)

				sessionTime, _ := strconv.Atoi(buf[0]["apppool_affinity_session_time"].(string))
				d.Set("affinity_session_duration", sessionTime)
			}

			// Updating best active nodes value
			if buf[0]["apppool_best_active_nodes"].(string) != "" {
				bestActiveNodes, _ := strconv.Atoi(buf[0]["apppool_best_active_nodes"].(string))
				d.Set("best_active_nodes", bestActiveNodes)
			} else {
				d.Set("best_active_nodes", 0)
			}

			return []*schema.ResourceData{d}, nil
		}

		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				tflog.Debug(ctx, fmt.Sprintf("Unable to import application pool (oid): %s (%s)\n", d.Id(), errMsg))
			}
		} else {
			tflog.Debug(ctx, fmt.Sprintf("Unable to find and import application pool (oid): %s\n", d.Id()))
		}

		// Reporting a failure
		return nil, fmt.Errorf("SOLIDServer - Unable to find and import application pool (oid): %s\n", d.Id())
	}

	// Reporting a failure
	return nil, err
}
