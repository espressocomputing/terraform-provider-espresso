package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestProviderSchema(t *testing.T) {
	if err := New().InternalValidate(); err != nil {
		t.Fatal(err)
	}
	account := New().ResourcesMap["espresso_account"]
	if account.Schema["display_name"].ForceNew || !account.Schema["slug"].ForceNew || !account.Schema["product"].ForceNew {
		t.Fatal("only display_name should be mutable")
	}
	accountData := schema.TestResourceDataRaw(t, account.Schema, map[string]any{"slug": "acme", "display_name": "Acme", "product": "databricks"})
	if accountSlug(accountData) != "databricks_acme" || !account.Schema["slug"].DiffSuppressFunc("slug", "databricks_acme", "acme", accountData) {
		t.Fatal("Databricks account slug was not normalized")
	}
}

func TestCreateOrAdoptAccount(t *testing.T) {
	tests := []struct {
		name            string
		existing        bool
		existingName    string
		existingProduct string
		expectedMethods []string
		expectedError   bool
	}{
		{name: "create", expectedMethods: []string{http.MethodGet, http.MethodPost}},
		{name: "adopt", existing: true, existingName: "Acme", existingProduct: "databricks", expectedMethods: []string{http.MethodGet}},
		{name: "reconcile name", existing: true, existingName: "Old name", existingProduct: "databricks", expectedMethods: []string{http.MethodGet, http.MethodPut}},
		{name: "reject product mismatch", existing: true, existingName: "Acme", existingProduct: "snowflake", expectedMethods: []string{http.MethodGet}, expectedError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var methods []string
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				methods = append(methods, request.Method)
				if request.Method == http.MethodGet {
					if !test.existing {
						response.WriteHeader(http.StatusNotFound)
						return
					}
					response.Header().Set("Content-Type", "application/json")
					json.NewEncoder(response).Encode(map[string]any{"name": test.existingName, "product": test.existingProduct})
				}
			}))
			defer server.Close()

			resource := New().ResourcesMap["espresso_account"]
			data := schema.TestResourceDataRaw(t, resource.Schema, map[string]any{"slug": "acme", "display_name": "Acme", "product": "databricks"})
			err := createOrAdoptAccount(context.Background(), data, &apiClient{endpoint: server.URL, key: "ok_test"})
			if (err != nil) != test.expectedError {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(methods) != len(test.expectedMethods) {
				t.Fatalf("unexpected methods: %v", methods)
			}
			for index, method := range methods {
				if method != test.expectedMethods[index] {
					t.Fatalf("unexpected methods: %v", methods)
				}
			}
			if !test.expectedError && (data.Id() != "databricks_acme" || data.Get("slug") != "databricks_acme") {
				t.Fatalf("unexpected account state: %#v", data.State())
			}
		})
	}
}

func withRawConfig(t *testing.T, fields map[string]*schema.Schema, data *schema.ResourceData, configured map[string]cty.Value) *schema.ResourceData {
	values := map[string]cty.Value{}
	for name, field := range fields {
		var valueType cty.Type
		switch field.Type {
		case schema.TypeString:
			valueType = cty.String
		case schema.TypeInt:
			valueType = cty.Number
		case schema.TypeBool:
			valueType = cty.Bool
		default:
			t.Fatalf("unsupported field type: %v", field.Type)
		}
		values[name] = cty.NullVal(valueType)
	}
	for name, value := range configured {
		values[name] = value
	}
	id := data.Id()
	if id == "" {
		data.SetId("test")
	}
	state := data.State()
	state.ID = id
	state.RawConfig = cty.ObjectVal(values)
	result, err := schema.InternalMap(fields).Data(state, nil)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestSnowflakeCredentials(t *testing.T) {
	var requests []snowflakeCredentials
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, httpRequest *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		if httpRequest.URL.Path == "/api/customers/acme/snowflake-credentials/public-key" {
			if httpRequest.Method != http.MethodGet {
				t.Errorf("unexpected public key method: %s", httpRequest.Method)
			}
			json.NewEncoder(response).Encode(snowflakePublicKey{PublicKey: "PUBLIC_KEY"})
			return
		}
		if httpRequest.URL.Path != "/api/customers/acme/snowflake-credentials" {
			t.Errorf("unexpected path: %s", httpRequest.URL.Path)
		}
		if httpRequest.Header.Get("Authorization") != "Bearer ok_test" {
			t.Error("missing API key")
		}
		if httpRequest.Method == http.MethodPut {
			var request snowflakeCredentials
			if err := json.NewDecoder(httpRequest.Body).Decode(&request); err != nil {
				t.Error(err)
			}
			requests = append(requests, request)
		}
		json.NewEncoder(response).Encode(snowflakeCredentials{Account: "acme-org-acme", Host: "acme-org-acme.snowflakecomputing.com", User: "ESPRESSO_AI_USER", Role: "ESPRESSO_AI_ROLE", Warehouse: "ESPRESSO_AI_WH", KeypairAuth: true})
	}))
	defer server.Close()

	resource := New().ResourcesMap["espresso_snowflake_credentials"]
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]any{
		"account": "acme", "snowflake_account": "acme-org_acme", "username": "ESPRESSO_AI_USER", "role": "ESPRESSO_AI_ROLE", "warehouse": "ESPRESSO_AI_WH",
	})
	data = withRawConfig(t, resource.Schema, data, nil)
	client := &apiClient{endpoint: server.URL, key: "ok_test"}
	publicKeyDataSource := New().DataSourcesMap["espresso_snowflake_public_key"]
	publicKeyData := schema.TestResourceDataRaw(t, publicKeyDataSource.Schema, map[string]any{"account": "acme"})
	if err := readSnowflakePublicKey(context.Background(), publicKeyData, client); err != nil {
		t.Fatal(err)
	}
	if publicKeyData.Get("public_key") != "PUBLIC_KEY" {
		t.Fatalf("unexpected public key: %q", publicKeyData.Get("public_key"))
	}
	if err := applySnowflakeCredentials(context.Background(), data, client); err != nil {
		t.Fatal(err)
	}
	if err := readSnowflakeCredentials(context.Background(), data, client); err != nil {
		t.Fatal(err)
	}
	if err := data.Set("snowflake_account", "next-account"); err != nil {
		t.Fatal(err)
	}
	data = withRawConfig(t, resource.Schema, data, map[string]cty.Value{"snowflake_account": cty.StringVal("next-account")})
	if err := applySnowflakeCredentials(context.Background(), data, client); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 || !requests[0].KeypairAuth || requests[0].Account != "acme-org-acme" || requests[1].Account != "next-account" || requests[1].Host != "" {
		t.Fatalf("unexpected requests: %+v", requests)
	}
	if data.Get("host") != "acme-org-acme.snowflakecomputing.com" {
		t.Fatalf("unexpected state: host=%v", data.Get("host"))
	}
}

func TestDatabricksCredentials(t *testing.T) {
	var request databricksCredentials
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, httpRequest *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		if httpRequest.URL.Path != "/api/customers/databricks_acme/databricks-credentials" {
			t.Errorf("unexpected path: %s", httpRequest.URL.Path)
		}
		if httpRequest.Method == http.MethodPut {
			if err := json.NewDecoder(httpRequest.Body).Decode(&request); err != nil {
				t.Error(err)
			}
			json.NewEncoder(response).Encode(map[string]any{"ok": true, "message": "Connection tested and saved."})
			return
		}
		json.NewEncoder(response).Encode(databricksCredentials{WorkspaceURL: "https://workspace.cloud.databricks.com", WorkspaceID: "123", WorkspaceName: "Production", ClientID: "client-id", ServicePrincipalID: "456", ServicePrincipalName: "espresso-ai-optimizer", WarehouseID: "warehouse-id", WarehouseName: "ESPRESSO_AI_WAREHOUSE"})
	}))
	defer server.Close()

	resource := New().ResourcesMap["espresso_databricks_credentials"]
	if !resource.Schema["client_secret"].WriteOnly || !resource.Schema["client_secret"].Sensitive {
		t.Fatal("client_secret must be sensitive and write-only")
	}
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]any{
		"account": "databricks_acme", "workspace_url": "https://workspace.cloud.databricks.com", "workspace_id": "123", "workspace_name": "Production", "client_id": "client-id", "client_secret": "secret-value", "service_principal_id": "456", "service_principal_name": "espresso-ai-optimizer", "warehouse_id": "warehouse-id", "warehouse_name": "ESPRESSO_AI_WAREHOUSE",
	})
	client := &apiClient{endpoint: server.URL, key: "ok_test"}
	if err := applyDatabricksCredentials(context.Background(), data, client); err != nil {
		t.Fatal(err)
	}
	if request.ClientSecret != "secret-value" || request.WarehouseID != "warehouse-id" {
		t.Fatalf("unexpected request: %+v", request)
	}
	if err := readDatabricksCredentials(context.Background(), data, client); err != nil {
		t.Fatal(err)
	}
	if data.Get("workspace_name") != "Production" || data.Get("warehouse_name") != "ESPRESSO_AI_WAREHOUSE" {
		t.Fatalf("unexpected state: %+v", data.State())
	}
}

func TestWarehouseAgents(t *testing.T) {
	cases := []struct {
		globalResource, warehouseResource, kind, productField, apiField, wireField string
		productValue                                                               any
		rawValue                                                                   cty.Value
	}{
		{"espresso_snowflake_warehouse_agent", "espresso_snowflake_warehouse_agent_warehouse", "autoscaler", "scaling_policy", "scaling_policy", "scalingPolicy", "STANDARD", cty.StringVal("STANDARD")},
		{"espresso_databricks_warehouse_agent", "espresso_databricks_warehouse_agent_warehouse", "databricks", "enabled", "enabled", "enabled", true, cty.BoolVal(true)},
	}
	for _, test := range cases {
		t.Run(test.kind, func(t *testing.T) {
			var paths []string
			var bodies []map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				paths = append(paths, request.Method+" "+request.URL.Path)
				if request.Method == http.MethodPut {
					var body map[string]any
					if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					bodies = append(bodies, body)
					return
				}
				response.Header().Set("Content-Type", "application/json")
				managed := map[string]any{"name": "MANAGED", "enabled": true, "minClusters": 1, "maxClusters": 4}
				managed[test.wireField] = test.productValue
				json.NewEncoder(response).Encode(map[string]any{
					"enabled": true, "autoOptIn": true,
					"warehouses": []any{
						managed,
						map[string]any{"name": "UNMANAGED", "enabled": true},
					},
				})
			}))
			defer server.Close()

			client := &apiClient{endpoint: server.URL, key: "ok_test"}
			global := New().ResourcesMap[test.globalResource]
			globalData := schema.TestResourceDataRaw(t, global.Schema, map[string]any{"account": "acme", "enabled": true, "auto_opt_in": true})
			if err := applyWarehouseAgent(context.Background(), globalData, client, test.kind); err != nil {
				t.Fatal(err)
			}
			if err := readWarehouseAgent(context.Background(), globalData, client, test.kind); err != nil {
				t.Fatal(err)
			}
			warehouse := New().ResourcesMap[test.warehouseResource]
			warehouseData := schema.TestResourceDataRaw(t, warehouse.Schema, map[string]any{"account": "acme", "name": "MANAGED", test.productField: test.productValue})
			warehouseData = withRawConfig(t, warehouse.Schema, warehouseData, map[string]cty.Value{test.productField: test.rawValue})
			if err := applyWarehouseAgent(context.Background(), warehouseData, client, test.kind); err != nil {
				t.Fatal(err)
			}
			if err := readWarehouseAgent(context.Background(), warehouseData, client, test.kind); err != nil {
				t.Fatal(err)
			}
			warehouseData = withRawConfig(t, warehouse.Schema, warehouseData, map[string]cty.Value{test.productField: test.rawValue})
			if err := applyWarehouseAgent(context.Background(), warehouseData, client, test.kind); err != nil {
				t.Fatal(err)
			}
			if len(paths) != 5 || paths[0] != "PUT /api/customers/acme/settings/"+test.kind+"/global" || paths[1] != "GET /api/customers/acme/settings/"+test.kind || paths[2] != "PUT /api/customers/acme/settings/"+test.kind+"/warehouses" || paths[3] != "GET /api/customers/acme/settings/"+test.kind || paths[4] != "PUT /api/customers/acme/settings/"+test.kind+"/warehouses" {
				t.Fatalf("unexpected requests: %v", paths)
			}
			for _, body := range bodies[1:] {
				changes := body["changes"].(map[string]any)["MANAGED"].(map[string]any)
				if len(changes) != 1 || changes[test.apiField] != test.productValue {
					t.Fatalf("unexpected changes: %v", changes)
				}
			}
			if _, found := bodies[0]["instance_cost_per_dbu"]; found {
				t.Fatalf("unexpected global settings: %v", bodies[0])
			}
			if warehouseData.Get(test.productField) != test.productValue || warehouseData.Get("enabled") != true {
				t.Fatalf("unexpected warehouse state: %v %v", warehouseData.Get(test.productField), warehouseData.Get("enabled"))
			}
		})
	}
}
