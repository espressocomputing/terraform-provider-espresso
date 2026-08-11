package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type apiClient struct{ endpoint, key string }

type operation func(context.Context, *schema.ResourceData, *apiClient) error

type kindOperation func(context.Context, *schema.ResourceData, *apiClient, string) error

type snowflakeCredentials struct {
	Account     string `json:"account"`
	Host        string `json:"host"`
	User        string `json:"user"`
	Role        string `json:"role"`
	Warehouse   string `json:"warehouse"`
	KeypairAuth bool   `json:"keypairAuth"`
}

type snowflakePublicKey struct {
	PublicKey string `json:"publicKey"`
}

type databricksCredentials struct {
	WorkspaceURL         string `json:"workspaceUrl"`
	WorkspaceID          string `json:"workspaceId"`
	WorkspaceName        string `json:"workspaceName"`
	ClientID             string `json:"clientId"`
	ClientSecret         string `json:"clientSecret,omitempty"`
	ServicePrincipalID   string `json:"servicePrincipalId"`
	ServicePrincipalName string `json:"servicePrincipalName"`
	WarehouseID          string `json:"warehouseId"`
	WarehouseName        string `json:"warehouseName"`
}

type warehouseAgentConfig struct {
	Enabled    bool                      `json:"enabled"`
	AutoOptIn  bool                      `json:"autoOptIn"`
	Warehouses []warehouseAgentWarehouse `json:"warehouses"`
}

type warehouseAgentWarehouse struct {
	Name          string  `json:"name"`
	Enabled       bool    `json:"enabled"`
	MinClusters   *int    `json:"minClusters"`
	MaxClusters   *int    `json:"maxClusters"`
	ScalingPolicy *string `json:"scalingPolicy"`
}

var notFound = errors.New("not found")

func (c *apiClient) do(ctx context.Context, method, path string, body, result any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	if body == nil {
		payload = nil
	}
	request, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+c.key)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return notFound
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Espresso API returned %s", response.Status)
	}
	if result == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(result)
}

func diagnose(op operation) func(context.Context, *schema.ResourceData, any) diag.Diagnostics {
	return func(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
		err := op(ctx, data, meta.(*apiClient))
		if errors.Is(err, notFound) && data.Id() != "" {
			data.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
}

func managed(ops [4]operation, fields map[string]*schema.Schema) *schema.Resource {
	return &schema.Resource{
		CreateContext: diagnose(ops[0]), ReadContext: diagnose(ops[1]),
		UpdateContext: diagnose(ops[2]), DeleteContext: diagnose(ops[3]), Schema: fields,
	}
}

func noop(context.Context, *schema.ResourceData, *apiClient) error { return nil }

func forKind(kind string, op kindOperation) operation {
	return func(ctx context.Context, data *schema.ResourceData, client *apiClient) error {
		return op(ctx, data, client, kind)
	}
}

func canonicalAccountSlug(slug, product string) string {
	if product == "databricks" && !strings.HasPrefix(slug, "databricks_") {
		return "databricks_" + slug
	}
	return slug
}

func accountSlug(data *schema.ResourceData) string {
	return canonicalAccountSlug(data.Get("slug").(string), data.Get("product").(string))
}

func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"endpoint": {Type: schema.TypeString, Optional: true, DefaultFunc: schema.EnvDefaultFunc("ESPRESSO_ENDPOINT", nil), ValidateFunc: validation.StringIsNotEmpty},
			"api_key":  {Type: schema.TypeString, Optional: true, Sensitive: true, DefaultFunc: schema.EnvDefaultFunc("ESPRESSO_API_KEY", nil), ValidateFunc: validation.StringIsNotEmpty},
		},
		ResourcesMap: map[string]*schema.Resource{
			"espresso_account": managed([4]operation{createOrAdoptAccount, readAccount, updateAccount, noop}, map[string]*schema.Schema{
				"slug": {Type: schema.TypeString, Required: true, ForceNew: true, DiffSuppressFunc: func(_ string, old string, new string, data *schema.ResourceData) bool {
					return old == canonicalAccountSlug(new, data.Get("product").(string))
				}}, "display_name": {Type: schema.TypeString, Required: true}, "product": {Type: schema.TypeString, Required: true, ForceNew: true},
			}),
			"espresso_snowflake_credentials": managed([4]operation{applySnowflakeCredentials, readSnowflakeCredentials, applySnowflakeCredentials, noop}, map[string]*schema.Schema{
				"account": {Type: schema.TypeString, Required: true, ForceNew: true}, "snowflake_account": {Type: schema.TypeString, Required: true, StateFunc: func(value any) string { return strings.ReplaceAll(value.(string), "_", "-") }}, "host": {Type: schema.TypeString, Optional: true, Computed: true, StateFunc: func(value any) string { return strings.ReplaceAll(value.(string), "_", "-") }}, "username": {Type: schema.TypeString, Required: true}, "role": {Type: schema.TypeString, Required: true}, "warehouse": {Type: schema.TypeString, Required: true},
			}),
			"espresso_databricks_credentials": managed([4]operation{applyDatabricksCredentials, readDatabricksCredentials, applyDatabricksCredentials, noop}, map[string]*schema.Schema{
				"account": {Type: schema.TypeString, Required: true, ForceNew: true}, "workspace_url": {Type: schema.TypeString, Required: true}, "workspace_id": {Type: schema.TypeString, Required: true}, "workspace_name": {Type: schema.TypeString, Optional: true}, "client_id": {Type: schema.TypeString, Required: true}, "client_secret": {Type: schema.TypeString, Required: true, Sensitive: true, WriteOnly: true}, "service_principal_id": {Type: schema.TypeString, Required: true}, "service_principal_name": {Type: schema.TypeString, Required: true}, "warehouse_id": {Type: schema.TypeString, Required: true}, "warehouse_name": {Type: schema.TypeString, Optional: true},
			}),
			"espresso_snowflake_warehouse_agent": warehouseAgentResource("autoscaler"),
			"espresso_snowflake_warehouse_agent_warehouse": warehouseAgentWarehouseResource("autoscaler", map[string]*schema.Schema{
				"enabled": {Type: schema.TypeBool, Optional: true, Computed: true}, "min_clusters": {Type: schema.TypeInt, Optional: true, Computed: true}, "max_clusters": {Type: schema.TypeInt, Optional: true, Computed: true}, "scaling_policy": {Type: schema.TypeString, Optional: true, Computed: true},
			}),
			"espresso_databricks_warehouse_agent": warehouseAgentResource("databricks"),
			"espresso_databricks_warehouse_agent_warehouse": warehouseAgentWarehouseResource("databricks", map[string]*schema.Schema{
				"enabled": {Type: schema.TypeBool, Optional: true, Computed: true}, "min_clusters": {Type: schema.TypeInt, Optional: true, Computed: true}, "max_clusters": {Type: schema.TypeInt, Optional: true, Computed: true},
			}),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"espresso_snowflake_public_key": {ReadContext: diagnose(readSnowflakePublicKey), Schema: map[string]*schema.Schema{
				"account": {Type: schema.TypeString, Required: true}, "public_key": {Type: schema.TypeString, Computed: true},
			}},
		},
		ConfigureContextFunc: func(_ context.Context, data *schema.ResourceData) (any, diag.Diagnostics) {
			return &apiClient{strings.TrimRight(data.Get("endpoint").(string), "/"), data.Get("api_key").(string)}, nil
		},
	}
}

func warehouseAgentResource(kind string) *schema.Resource {
	fields := map[string]*schema.Schema{
		"account": {Type: schema.TypeString, Required: true, ForceNew: true}, "enabled": {Type: schema.TypeBool, Required: true}, "auto_opt_in": {Type: schema.TypeBool, Optional: true, Default: false}, "notes": {Type: schema.TypeString, Optional: true},
	}
	return managed([4]operation{forKind(kind, applyWarehouseAgent), forKind(kind, readWarehouseAgent), forKind(kind, applyWarehouseAgent), noop}, fields)
}

func warehouseAgentWarehouseResource(kind string, fields map[string]*schema.Schema) *schema.Resource {
	fields["account"] = &schema.Schema{Type: schema.TypeString, Required: true, ForceNew: true}
	fields["name"] = &schema.Schema{Type: schema.TypeString, Required: true, ForceNew: true}
	fields["notes"] = &schema.Schema{Type: schema.TypeString, Optional: true}
	return managed([4]operation{forKind(kind, applyWarehouseAgent), forKind(kind, readWarehouseAgent), forKind(kind, applyWarehouseAgent), noop}, fields)
}

func configuredValue(data *schema.ResourceData, field string) (any, bool) {
	config := data.GetRawConfig()
	if config.IsNull() || !config.Type().HasAttribute(field) || config.GetAttr(field).IsNull() {
		return nil, false
	}
	return data.Get(field), true
}

func applySnowflakeCredentials(ctx context.Context, data *schema.ResourceData, client *apiClient) error {
	host := ""
	if value, ok := configuredValue(data, "host"); ok {
		host = value.(string)
	}
	value := snowflakeCredentials{
		Account: data.Get("snowflake_account").(string), Host: host, User: data.Get("username").(string), Role: data.Get("role").(string), Warehouse: data.Get("warehouse").(string), KeypairAuth: true,
	}
	if err := client.do(ctx, http.MethodPut, "/api/customers/"+data.Get("account").(string)+"/snowflake-credentials", value, &value); err != nil {
		return err
	}
	data.SetId(data.Get("account").(string) + "/snowflake-credentials")
	return setSnowflakeCredentials(data, value)
}

func readSnowflakeCredentials(ctx context.Context, data *schema.ResourceData, client *apiClient) error {
	var value snowflakeCredentials
	if err := client.do(ctx, http.MethodGet, "/api/customers/"+data.Get("account").(string)+"/snowflake-credentials", nil, &value); err != nil {
		return err
	}
	return setSnowflakeCredentials(data, value)
}

func setSnowflakeCredentials(data *schema.ResourceData, value snowflakeCredentials) error {
	return errors.Join(data.Set("snowflake_account", value.Account), data.Set("host", value.Host), data.Set("username", value.User), data.Set("role", value.Role), data.Set("warehouse", value.Warehouse))
}

func applyDatabricksCredentials(ctx context.Context, data *schema.ResourceData, client *apiClient) error {
	value := databricksCredentials{
		WorkspaceURL: data.Get("workspace_url").(string), WorkspaceID: data.Get("workspace_id").(string), WorkspaceName: data.Get("workspace_name").(string), ClientID: data.Get("client_id").(string), ClientSecret: data.Get("client_secret").(string), ServicePrincipalID: data.Get("service_principal_id").(string), ServicePrincipalName: data.Get("service_principal_name").(string), WarehouseID: data.Get("warehouse_id").(string), WarehouseName: data.Get("warehouse_name").(string),
	}
	if err := client.do(ctx, http.MethodPut, "/api/customers/"+data.Get("account").(string)+"/databricks-credentials", value, nil); err != nil {
		return err
	}
	data.SetId(data.Get("account").(string) + "/databricks-credentials")
	return nil
}

func readDatabricksCredentials(ctx context.Context, data *schema.ResourceData, client *apiClient) error {
	var value databricksCredentials
	if err := client.do(ctx, http.MethodGet, "/api/customers/"+data.Get("account").(string)+"/databricks-credentials", nil, &value); err != nil {
		return err
	}
	return errors.Join(data.Set("workspace_url", value.WorkspaceURL), data.Set("workspace_id", value.WorkspaceID), data.Set("workspace_name", value.WorkspaceName), data.Set("client_id", value.ClientID), data.Set("service_principal_id", value.ServicePrincipalID), data.Set("service_principal_name", value.ServicePrincipalName), data.Set("warehouse_id", value.WarehouseID), data.Set("warehouse_name", value.WarehouseName))
}

func readSnowflakePublicKey(ctx context.Context, data *schema.ResourceData, client *apiClient) error {
	var value snowflakePublicKey
	if err := client.do(ctx, http.MethodGet, "/api/customers/"+data.Get("account").(string)+"/snowflake-credentials/public-key", nil, &value); err != nil {
		return err
	}
	data.SetId(data.Get("account").(string) + "/snowflake-public-key")
	return data.Set("public_key", value.PublicKey)
}

func applyWarehouseAgent(ctx context.Context, data *schema.ResourceData, client *apiClient, kind string) error {
	name, warehouseResource := data.GetOk("name")
	if !warehouseResource {
		body := map[string]any{"enabled": data.Get("enabled"), "auto_opt_in": data.Get("auto_opt_in"), "notes": data.Get("notes")}
		if err := client.do(ctx, http.MethodPut, "/api/customers/"+data.Get("account").(string)+"/settings/"+kind+"/global", body, nil); err != nil {
			return err
		}
		data.SetId(data.Get("account").(string) + "/" + kind)
		return nil
	}
	fields := []string{"enabled", "min_clusters", "max_clusters", "scaling_policy"}
	change := map[string]any{}
	for _, field := range fields {
		if value, ok := configuredValue(data, field); ok {
			change[field] = value
		}
	}
	body := map[string]any{"changes": map[string]any{name.(string): change}, "notes": data.Get("notes")}
	if err := client.do(ctx, http.MethodPut, "/api/customers/"+data.Get("account").(string)+"/settings/"+kind+"/warehouses", body, nil); err != nil {
		return err
	}
	data.SetId(data.Get("account").(string) + "/" + kind + "/" + name.(string))
	return nil
}

func readWarehouseAgent(ctx context.Context, data *schema.ResourceData, client *apiClient, kind string) error {
	var value warehouseAgentConfig
	if err := client.do(ctx, http.MethodGet, "/api/customers/"+data.Get("account").(string)+"/settings/"+kind, nil, &value); err != nil {
		return err
	}
	name, warehouseResource := data.GetOk("name")
	if !warehouseResource {
		return errors.Join(data.Set("enabled", value.Enabled), data.Set("auto_opt_in", value.AutoOptIn))
	}
	for _, warehouse := range value.Warehouses {
		if warehouse.Name != name.(string) {
			continue
		}
		errs := []error{data.Set("enabled", warehouse.Enabled), data.Set("min_clusters", warehouse.MinClusters), data.Set("max_clusters", warehouse.MaxClusters)}
		if kind == "autoscaler" {
			errs = append(errs, data.Set("scaling_policy", warehouse.ScalingPolicy))
		}
		return errors.Join(errs...)
	}
	data.SetId("")
	return nil
}

func createOrAdoptAccount(ctx context.Context, data *schema.ResourceData, client *apiClient) error {
	slug := accountSlug(data)
	var existing struct {
		Name    string `json:"name"`
		Product string `json:"product"`
	}
	err := client.do(ctx, http.MethodGet, "/api/customers/"+slug, nil, &existing)
	if err == nil {
		if existing.Product != data.Get("product").(string) {
			return fmt.Errorf("existing account %q has product %q, expected %q", slug, existing.Product, data.Get("product"))
		}
		if existing.Name != data.Get("display_name").(string) {
			if err := client.do(ctx, http.MethodPut, "/api/customers/"+slug, map[string]any{"displayName": data.Get("display_name")}, nil); err != nil {
				return err
			}
		}
		if err := data.Set("slug", slug); err != nil {
			return err
		}
		data.SetId(slug)
		return nil
	}
	if !errors.Is(err, notFound) {
		return err
	}
	body := map[string]any{"slug": slug, "displayName": data.Get("display_name"), "product": data.Get("product")}
	if err := client.do(ctx, http.MethodPost, "/api/customers", body, nil); err != nil {
		return err
	}
	if err := data.Set("slug", slug); err != nil {
		return err
	}
	data.SetId(slug)
	return nil
}

func readAccount(ctx context.Context, data *schema.ResourceData, client *apiClient) error {
	var value struct {
		Name    string `json:"name"`
		Product string `json:"product"`
	}
	if err := client.do(ctx, http.MethodGet, "/api/customers/"+data.Id(), nil, &value); err != nil {
		return err
	}
	return errors.Join(data.Set("slug", data.Id()), data.Set("display_name", value.Name), data.Set("product", value.Product))
}

func updateAccount(ctx context.Context, data *schema.ResourceData, client *apiClient) error {
	return client.do(ctx, http.MethodPut, "/api/customers/"+data.Id(), map[string]any{"displayName": data.Get("display_name")}, nil)
}
