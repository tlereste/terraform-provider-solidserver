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
	"strings"
)

func resourcednsforwardzone() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcednsforwardzoneCreate,
		ReadContext:   resourcednsforwardzoneRead,
		UpdateContext: resourcednsforwardzoneUpdate,
		DeleteContext: resourcednsforwardzoneDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourcednsforwardzoneImportState,
		},

		Description: heredoc.Doc(`
			DNS Forward Zone allows to create and manage DNS forward zones.
		`),

		Schema: map[string]*schema.Schema{
			"dnsserver": {
				Type:        schema.TypeString,
				Description: "The managed SMART DNS server name, or DNS server name hosting the forward zone.",
				Required:    true,
				ForceNew:    true,
			},
			"dnsview": {
				Type:        schema.TypeString,
				Description: "The DNS view name hosting the forward zone.",
				Optional:    true,
				ForceNew:    true,
				Default:     "#",
			},
			"name": {
				Type:        schema.TypeString,
				Description: "The Domain Name served by the forward zone.",
				Required:    true,
				ForceNew:    true,
			},
			"forward": {
				Type:         schema.TypeString,
				Description:  "The forwarding mode of the forward zone (Supported: only, first; Default: only).",
				ValidateFunc: validation.StringInSlice([]string{"first", "only"}, true),
				Optional:     true,
				Default:      "only",
			},
			"forwarders": {
				Type:        schema.TypeList,
				Description: "The IP address list of the forwarder(s) to use for the forward zone.",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"class": {
				Type:        schema.TypeString,
				Description: "The class associated to the forward zone.",
				Optional:    true,
				ForceNew:    false,
				Default:     "",
			},
			"class_parameters": {
				Type:        schema.TypeMap,
				Description: "The class parameters associated to the forward zone.",
				Optional:    true,
				ForceNew:    false,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourcednsforwardzoneCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("add_flag", "new_only")
	parameters.Add("dns_name", d.Get("dnsserver").(string))
	if strings.Compare(d.Get("dnsview").(string), "#") != 0 {
		parameters.Add("dnsview_name", d.Get("dnsview").(string))
	}
	parameters.Add("dnszone_name", d.Get("name").(string))
	parameters.Add("dnszone_type", "forward")
	parameters.Add("dnszone_class_name", d.Get("class").(string))

	// Building forwarder list
	parameters.Add("dnszone_forward", strings.ToLower(d.Get("forward").(string)))

	fwdList := ""
	for _, fwd := range toStringArray(d.Get("forwarders").([]interface{})) {
		fwdList += fwd + ";"
	}

	parameters.Add("dnszone_forwarders", fwdList)

	// Building class_parameters
	classParameters := urlfromclassparams(d.Get("class_parameters"))
	parameters.Add("dnszone_class_parameters", classParameters.Encode())

	// Sending the creation request
	resp, body, err := s.Request("post", "rest/dns_zone_add", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			if oid, oidExist := buf[0]["ret_oid"].(string); oidExist {
				tflog.Debug(ctx, fmt.Sprintf("Created DNS forward zone (oid): %s\n", oid))
				d.SetId(oid)
				return nil
			}
		}

		// Reporting a failure
		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				return diag.Errorf("Unable to create DNS forward zone: %s (%s)", d.Get("name").(string), errMsg)
			}
		}

		return diag.Errorf("Unable to create DNS forward zone: %s\n", d.Get("name").(string))
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourcednsforwardzoneUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("dnszone_id", d.Id())
	parameters.Add("add_flag", "edit_only")
	parameters.Add("dnszone_class_name", d.Get("class").(string))

	// Building forwarder list
	parameters.Add("dnszone_forward", strings.ToLower(d.Get("forward").(string)))

	fwdList := ""
	for _, fwd := range toStringArray(d.Get("forwarders").([]interface{})) {
		fwdList += fwd + ";"
	}

	parameters.Add("dnszone_forwarders", fwdList)

	// Building class_parameters
	classParameters := urlfromclassparams(d.Get("class_parameters"))
	parameters.Add("dnszone_class_parameters", classParameters.Encode())

	// Sending the update request
	resp, body, err := s.Request("put", "rest/dns_zone_add", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			if oid, oidExist := buf[0]["ret_oid"].(string); oidExist {
				tflog.Debug(ctx, fmt.Sprintf("Updated DNS forward zone (oid): %s\n", oid))
				d.SetId(oid)
				return nil
			}
		}

		// Reporting a failure
		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				return diag.Errorf("Unable to update DNS forward zone: %s (%s)", d.Get("name").(string), errMsg)
			}
		}

		return diag.Errorf("Unable to update DNS forward zone: %s\n", d.Get("name").(string))
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourcednsforwardzoneDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("dnszone_id", d.Id())

	// Sending the deletion request
	resp, body, err := s.Request("delete", "rest/dns_zone_delete", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode != 200 && resp.StatusCode != 204 {
			// Reporting a failure
			if len(buf) > 0 {
				if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
					return diag.Errorf("Unable to delete DNS forward zone: %s (%s)", d.Get("name").(string), errMsg)
				}
			}

			return diag.Errorf("Unable to delete DNS forward zone: %s", d.Get("name").(string))
		}

		// Log deletion
		tflog.Debug(ctx, fmt.Sprintf("Deleted DNS forward zone (oid): %s\n", d.Id()))

		// Unset local ID
		d.SetId("")

		// Reporting a success
		return nil
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourcednsforwardzoneRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("dnszone_id", d.Id())

	// Sending the read request
	resp, body, err := s.Request("get", "rest/dns_zone_info", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode == 200 && len(buf) > 0 {
			d.Set("dnsserver", buf[0]["dns_name"].(string))
			d.Set("dnsview", buf[0]["dnsview_name"].(string))
			d.Set("name", buf[0]["dnszone_name"].(string))

			// Updating forward mode
			if buf[0]["dnszone_forward"].(string) == "" {
				d.Set("forward", "none")
			} else {
				d.Set("forward", strings.ToLower(buf[0]["dnszone_forward"].(string)))
			}

			// Updating forwarder information
			if buf[0]["dnszone_forwarders"].(string) != "" {
				d.Set("forwarders", toStringArrayInterface(strings.Split(strings.TrimSuffix(buf[0]["dnszone_forwarders"].(string), ";"), ";")))
			}

			d.Set("class", buf[0]["dnszone_class_name"].(string))

			// Updating local class_parameters
			currentClassParameters := d.Get("class_parameters").(map[string]interface{})
			retrievedClassParameters, _ := url.ParseQuery(buf[0]["dnszone_class_parameters"].(string))
			computedClassParameters := map[string]string{}

			if _, createptrExist := retrievedClassParameters["dnsptr"]; createptrExist {
				delete(retrievedClassParameters, "dnsptr")
			}

			for ck := range currentClassParameters {
				if rv, rvExist := retrievedClassParameters[ck]; rvExist {
					computedClassParameters[ck] = rv[0]
				} else {
					computedClassParameters[ck] = ""
				}
			}

			d.Set("class_parameters", computedClassParameters)

			return nil
		}

		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				// Log the error
				tflog.Debug(ctx, fmt.Sprintf("Unable to find DNS forward zone: %s (%s)\n", d.Get("name"), errMsg))
			}
		} else {
			// Log the error
			tflog.Debug(ctx, fmt.Sprintf("Unable to find DNS forward zone (oid): %s\n", d.Id()))
		}

		// Do not unset the local ID to avoid inconsistency

		// Reporting a failure
		return diag.Errorf("Unable to find DNS forward zone: %s\n", d.Get("name").(string))
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourcednsforwardzoneImportState(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("dnszone_id", d.Id())

	// Sending the read request
	resp, body, err := s.Request("get", "rest/dns_zone_info", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode == 200 && len(buf) > 0 {
			d.Set("dnsserver", buf[0]["dns_name"].(string))
			d.Set("dnsview", buf[0]["dnsview_name"].(string))
			d.Set("name", buf[0]["dnszone_name"].(string))
			d.Set("type", buf[0]["dnszone_type"].(string))

			// Updating forward mode
			if buf[0]["dnszone_forward"].(string) == "" {
				d.Set("forward", "none")
			} else {
				d.Set("forward", strings.ToLower(buf[0]["dnszone_forward"].(string)))
			}

			// Updating forwarder information
			if buf[0]["dnszone_forwarders"].(string) != "" {
				d.Set("forwarders", toStringArrayInterface(strings.Split(strings.TrimSuffix(buf[0]["dnszone_forwarders"].(string), ";"), ";")))
			}

			d.Set("class", buf[0]["dnszone_class_name"].(string))

			// Updating local class_parameters
			currentClassParameters := d.Get("class_parameters").(map[string]interface{})
			retrievedClassParameters, _ := url.ParseQuery(buf[0]["dnszone_class_parameters"].(string))
			computedClassParameters := map[string]string{}

			if _, createptrExist := retrievedClassParameters["dnsptr"]; createptrExist {
				delete(retrievedClassParameters, "dnsptr")
			}

			for ck := range currentClassParameters {
				if rv, rvExist := retrievedClassParameters[ck]; rvExist {
					computedClassParameters[ck] = rv[0]
				} else {
					computedClassParameters[ck] = ""
				}
			}

			d.Set("class_parameters", computedClassParameters)

			return []*schema.ResourceData{d}, nil
		}

		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				tflog.Debug(ctx, fmt.Sprintf("Unable to import DNS forward zone (oid): %s (%s)\n", d.Id(), errMsg))
			}
		} else {
			tflog.Debug(ctx, fmt.Sprintf("Unable to find and import DNS forward zone (oid): %s\n", d.Id()))
		}

		// Reporting a failure
		return nil, fmt.Errorf("SOLIDServer - Unable to find and import DNS forward zone (oid): %s\n", d.Id())
	}

	// Reporting a failure
	return nil, err
}
