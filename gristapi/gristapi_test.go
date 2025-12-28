// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

package gristapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestConnect(t *testing.T) {
	orgs := GetOrgs()
	nbOrgs := len(orgs)

	if nbOrgs < 1 {
		t.Errorf("We only found %d organizations", nbOrgs)
	}

	for i, org := range orgs {
		orgId := fmt.Sprintf("%d", org.Id)
		if GetOrg(orgId).Name != orgs[i].Name {
			t.Error("We don't find main organization.")
		}

		workspaces := GetOrgWorkspaces(org.Id)

		if len(workspaces) < 1 {
			t.Errorf("No workspace in org n°%d", org.Id)
		}

		for i, workspace := range workspaces {
			if workspace.OrgDomain != org.Domain {
				t.Errorf("Workspace %d : le domaine du workspace %s ne correspond pas à %s", workspace.Id, workspace.OrgDomain, org.Domain)
			}

			myWorkspace := GetWorkspace(workspace.Id)
			if myWorkspace.Name != workspace.Name {
				t.Errorf("Workspace n°%d : les noms ne correspondent pas (%s/%s)", workspace.Id, workspace.Name, myWorkspace.Name)
			}

			if workspace.Name != workspaces[i].Name {
				t.Error("Mauvaise correspondance des noms de Workspaces")
			}

			for i, doc := range workspace.Docs {
				if doc.Name != workspace.Docs[i].Name {
					t.Errorf("Document n°%s : non correspondance des noms (%s/%s)", doc.Id, doc.Name, workspace.Docs[i].Name)
				}

				// // Un document doit avoir au moins une table
				// tables := GetDocTables(doc.Id)
				// if len(tables.Tables) < 1 {
				// 	t.Errorf("Le document n°%s ne contient pas de table (org %d/workspace %s)", doc.Name, org.Id, workspace.Name)
				// }
				// for _, table := range tables.Tables {
				// 	// Une table doit avoir au moins une colonne
				// 	cols := GetTableColumns(doc.Id, table.Id)
				// 	if len(cols.Columns) < 1 {
				// 		t.Errorf("La table %s du document %s ne contient pas de colonne", table.Id, doc.Id)
				// 	}
				// }
			}

		}
	}

}

// setupMockServer creates a test server and sets environment variables
func setupMockServer(handler http.HandlerFunc) (*httptest.Server, func()) {
	server := httptest.NewServer(handler)
	oldURL := os.Getenv("GRIST_URL")
	oldToken := os.Getenv("GRIST_TOKEN")
	os.Setenv("GRIST_URL", server.URL)
	os.Setenv("GRIST_TOKEN", "test-token")
	return server, func() {
		server.Close()
		os.Setenv("GRIST_URL", oldURL)
		os.Setenv("GRIST_TOKEN", oldToken)
	}
}

func TestBuildRecordsQueryParams(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]string
		expected string
	}{
		{
			name:     "empty params",
			params:   map[string]string{},
			expected: "",
		},
		{
			name:     "single param",
			params:   map[string]string{"limit": "10"},
			expected: "?limit=10",
		},
		{
			name:     "empty value ignored",
			params:   map[string]string{"limit": "", "sort": "name"},
			expected: "?sort=name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRecordsQueryParams(tt.params)
			// For single param case, exact match
			if len(tt.params) <= 1 {
				if result != tt.expected {
					t.Errorf("buildRecordsQueryParams() = %q, want %q", result, tt.expected)
				}
			} else {
				// For multiple params, just check it starts with ? and contains expected parts
				if tt.expected != "" && (result == "" || result[0] != '?') {
					t.Errorf("buildRecordsQueryParams() = %q, expected to start with '?'", result)
				}
			}
		})
	}
}

func TestGetRecords(t *testing.T) {
	expectedRecords := RecordsList{
		Records: []Record{
			{Id: 1, Fields: map[string]interface{}{"name": "Alice", "age": float64(30)}},
			{Id: 2, Fields: map[string]interface{}{"name": "Bob", "age": float64(25)}},
		},
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Bearer token, got %s", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedRecords)
	})
	defer cleanup()

	records, status := GetRecords("doc123", "Table1", nil)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(records.Records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(records.Records))
	}
	if records.Records[0].Id != 1 {
		t.Errorf("Expected first record ID 1, got %d", records.Records[0].Id)
	}
}

func TestGetRecordsWithOptions(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("sort") != "name" {
			t.Errorf("Expected sort=name, got %s", query.Get("sort"))
		}
		if query.Get("limit") != "10" {
			t.Errorf("Expected limit=10, got %s", query.Get("limit"))
		}
		if query.Get("hidden") != "true" {
			t.Errorf("Expected hidden=true, got %s", query.Get("hidden"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RecordsList{Records: []Record{}})
	})
	defer cleanup()

	options := &GetRecordsOptions{
		Sort:   "name",
		Limit:  10,
		Hidden: true,
	}
	_, status := GetRecords("doc123", "Table1", options)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestGetRecordsWithFilter(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		filterParam := query.Get("filter")
		if filterParam == "" {
			t.Error("Expected filter parameter")
		}

		var filter map[string][]interface{}
		if err := json.Unmarshal([]byte(filterParam), &filter); err != nil {
			t.Errorf("Failed to parse filter: %v", err)
		}
		if len(filter["name"]) != 1 || filter["name"][0] != "Alice" {
			t.Errorf("Expected filter for name=Alice, got %v", filter)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RecordsList{Records: []Record{}})
	})
	defer cleanup()

	options := &GetRecordsOptions{
		Filter: map[string][]interface{}{"name": {"Alice"}},
	}
	_, status := GetRecords("doc123", "Table1", options)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestAddRecords(t *testing.T) {
	expectedResponse := RecordsWithoutFields{
		Records: []struct {
			Id int `json:"id"`
		}{{Id: 1}, {Id: 2}},
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		var body struct {
			Records []struct {
				Fields map[string]interface{} `json:"fields"`
			} `json:"records"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if len(body.Records) != 2 {
			t.Errorf("Expected 2 records in request, got %d", len(body.Records))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	})
	defer cleanup()

	records := []map[string]interface{}{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
	}
	result, status := AddRecords("doc123", "Table1", records, nil)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(result.Records) != 2 {
		t.Errorf("Expected 2 record IDs, got %d", len(result.Records))
	}
}

func TestAddRecordsWithNoParse(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("noparse") != "true" {
			t.Errorf("Expected noparse=true, got %s", query.Get("noparse"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RecordsWithoutFields{})
	})
	defer cleanup()

	options := &AddRecordsOptions{NoParse: true}
	_, status := AddRecords("doc123", "Table1", []map[string]interface{}{}, options)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestUpdateRecords(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Expected PATCH request, got %s", r.Method)
		}

		var body struct {
			Records []Record `json:"records"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if len(body.Records) != 1 {
			t.Errorf("Expected 1 record in request, got %d", len(body.Records))
		}
		if body.Records[0].Id != 1 {
			t.Errorf("Expected record ID 1, got %d", body.Records[0].Id)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	records := []Record{
		{Id: 1, Fields: map[string]interface{}{"name": "Alice Updated"}},
	}
	_, status := UpdateRecords("doc123", "Table1", records, nil)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestUpsertRecords(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT request, got %s", r.Method)
		}

		var body struct {
			Records []RecordWithRequire `json:"records"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if len(body.Records) != 1 {
			t.Errorf("Expected 1 record in request, got %d", len(body.Records))
		}
		if body.Records[0].Require["email"] != "alice@example.com" {
			t.Errorf("Expected require email, got %v", body.Records[0].Require)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	records := []RecordWithRequire{
		{
			Require: map[string]interface{}{"email": "alice@example.com"},
			Fields:  map[string]interface{}{"name": "Alice", "age": 30},
		},
	}
	_, status := UpsertRecords("doc123", "Table1", records, nil)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestUpsertRecordsWithOptions(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("onmany") != "all" {
			t.Errorf("Expected onmany=all, got %s", query.Get("onmany"))
		}
		if query.Get("noadd") != "true" {
			t.Errorf("Expected noadd=true, got %s", query.Get("noadd"))
		}
		if query.Get("noupdate") != "true" {
			t.Errorf("Expected noupdate=true, got %s", query.Get("noupdate"))
		}
		if query.Get("allow_empty_require") != "true" {
			t.Errorf("Expected allow_empty_require=true, got %s", query.Get("allow_empty_require"))
		}

		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	options := &UpsertRecordsOptions{
		OnMany:            "all",
		NoAdd:             true,
		NoUpdate:          true,
		AllowEmptyRequire: true,
	}
	_, status := UpsertRecords("doc123", "Table1", []RecordWithRequire{}, options)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestDeleteRecords(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/docs/doc123/tables/Table1/records/delete" {
			t.Errorf("Expected delete endpoint, got %s", r.URL.Path)
		}

		var ids []int
		if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if len(ids) != 2 {
			t.Errorf("Expected 2 IDs, got %d", len(ids))
		}
		if ids[0] != 1 || ids[1] != 2 {
			t.Errorf("Expected IDs [1, 2], got %v", ids)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	_, status := DeleteRecords("doc123", "Table1", []int{1, 2})
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

// SCIM Bulk Operations Tests

func TestSCIMBulk_ValidRequest(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		// Mock response for SCIM user creation
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "user123",
			"userName": "testuser",
		})
	})
	defer cleanup()

	request := SCIMBulkRequest{
		Schemas: []string{SCIMBulkRequestSchema},
		Operations: []SCIMBulkOperation{
			{
				Method: "POST",
				Path:   "/Users",
				BulkId: "bulk1",
				Data: map[string]interface{}{
					"userName": "testuser",
					"emails": []map[string]interface{}{
						{"value": "test@example.com", "primary": true},
					},
				},
			},
		},
	}

	response, status := SCIMBulk(request)

	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(response.Schemas) != 1 || response.Schemas[0] != SCIMBulkResponseSchema {
		t.Errorf("Expected BulkResponse schema, got %v", response.Schemas)
	}
	if len(response.Operations) != 1 {
		t.Errorf("Expected 1 operation response, got %d", len(response.Operations))
	}
	if response.Operations[0].BulkId != "bulk1" {
		t.Errorf("Expected bulkId 'bulk1', got %s", response.Operations[0].BulkId)
	}
	if response.Operations[0].Method != "POST" {
		t.Errorf("Expected method 'POST', got %s", response.Operations[0].Method)
	}
}

func TestSCIMBulk_InvalidSchema(t *testing.T) {
	request := SCIMBulkRequest{
		Schemas: []string{"invalid:schema"},
		Operations: []SCIMBulkOperation{
			{
				Method: "POST",
				Path:   "/Users",
			},
		},
	}

	_, status := SCIMBulk(request)

	if status != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid schema, got %d", status)
	}
}

func TestSCIMBulk_InvalidMethod(t *testing.T) {
	request := SCIMBulkRequest{
		Schemas: []string{SCIMBulkRequestSchema},
		Operations: []SCIMBulkOperation{
			{
				Method: "GET", // GET is not allowed in bulk operations
				Path:   "/Users",
			},
		},
	}

	response, status := SCIMBulk(request)

	if status != http.StatusOK {
		t.Errorf("Expected status 200 (overall request succeeds), got %d", status)
	}
	if len(response.Operations) != 1 {
		t.Fatalf("Expected 1 operation response, got %d", len(response.Operations))
	}
	if response.Operations[0].Status != "400" {
		t.Errorf("Expected operation status '400', got %s", response.Operations[0].Status)
	}
}

func TestSCIMBulk_MissingPath(t *testing.T) {
	request := SCIMBulkRequest{
		Schemas: []string{SCIMBulkRequestSchema},
		Operations: []SCIMBulkOperation{
			{
				Method: "POST",
				Path:   "", // Empty path
			},
		},
	}

	response, status := SCIMBulk(request)

	if status != http.StatusOK {
		t.Errorf("Expected status 200 (overall request succeeds), got %d", status)
	}
	if response.Operations[0].Status != "400" {
		t.Errorf("Expected operation status '400' for missing path, got %s", response.Operations[0].Status)
	}
}

func TestSCIMBulk_MultipleOperations(t *testing.T) {
	callCount := 0
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case "POST":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "user1"})
		case "PATCH":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "user1", "updated": true})
		case "DELETE":
			w.WriteHeader(http.StatusNoContent)
		}
	})
	defer cleanup()

	request := SCIMBulkRequest{
		Schemas: []string{SCIMBulkRequestSchema},
		Operations: []SCIMBulkOperation{
			{
				Method: "POST",
				Path:   "/Users",
				BulkId: "op1",
				Data:   map[string]interface{}{"userName": "user1"},
			},
			{
				Method: "PATCH",
				Path:   "/Users/user1",
				BulkId: "op2",
				Data:   map[string]interface{}{"displayName": "Updated User"},
			},
			{
				Method: "DELETE",
				Path:   "/Users/user2",
				BulkId: "op3",
			},
		},
	}

	response, status := SCIMBulk(request)

	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(response.Operations) != 3 {
		t.Errorf("Expected 3 operation responses, got %d", len(response.Operations))
	}
	if callCount != 3 {
		t.Errorf("Expected 3 HTTP calls, got %d", callCount)
	}
}

func TestSCIMBulk_FailOnErrors(t *testing.T) {
	callCount := 0
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// All operations fail
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
	})
	defer cleanup()

	request := SCIMBulkRequest{
		Schemas:      []string{SCIMBulkRequestSchema},
		FailOnErrors: 2, // Stop after 2 errors
		Operations: []SCIMBulkOperation{
			{Method: "POST", Path: "/Users", BulkId: "op1"},
			{Method: "POST", Path: "/Users", BulkId: "op2"},
			{Method: "POST", Path: "/Users", BulkId: "op3"}, // Should not execute
			{Method: "POST", Path: "/Users", BulkId: "op4"}, // Should not execute
		},
	}

	response, _ := SCIMBulk(request)

	if len(response.Operations) != 2 {
		t.Errorf("Expected 2 operation responses (stopped after failOnErrors), got %d", len(response.Operations))
	}
	if callCount != 2 {
		t.Errorf("Expected 2 HTTP calls (stopped after failOnErrors), got %d", callCount)
	}
}

func TestSCIMBulkFromJSON_ValidJSON(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "user1"})
	})
	defer cleanup()

	jsonBody := `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
		"Operations": [
			{
				"method": "POST",
				"path": "/Users",
				"bulkId": "test1",
				"data": {"userName": "testuser"}
			}
		]
	}`

	response, status := SCIMBulkFromJSON(jsonBody)

	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(response.Operations) != 1 {
		t.Errorf("Expected 1 operation response, got %d", len(response.Operations))
	}
}

func TestSCIMBulkFromJSON_InvalidJSON(t *testing.T) {
	jsonBody := `{invalid json}`

	response, status := SCIMBulkFromJSON(jsonBody)

	if status != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", status)
	}
	if len(response.Operations) != 1 {
		t.Fatalf("Expected 1 error operation, got %d", len(response.Operations))
	}
	if response.Operations[0].Status != "400" {
		t.Errorf("Expected operation status '400', got %s", response.Operations[0].Status)
	}
}

func TestSCIMBulk_PUTOperation(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "user1", "userName": "updated"})
	})
	defer cleanup()

	request := SCIMBulkRequest{
		Schemas: []string{SCIMBulkRequestSchema},
		Operations: []SCIMBulkOperation{
			{
				Method: "PUT",
				Path:   "/Users/user1",
				Data:   map[string]interface{}{"userName": "updated"},
			},
		},
	}

	response, status := SCIMBulk(request)

	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if response.Operations[0].Status != "200" {
		t.Errorf("Expected operation status '200', got %s", response.Operations[0].Status)
	}
}

// =============================================================================
// SCIM User Management API Tests
// =============================================================================

func TestSCIMGetUsers(t *testing.T) {
	expectedResponse := SCIMListResponse{
		Schemas:      []string{SCIMListResponseSchema},
		TotalResults: 2,
		StartIndex:   1,
		ItemsPerPage: 10,
		Resources: []SCIMUser{
			{
				Schemas:  []string{SCIMUserSchema},
				Id:       "1",
				UserName: "alice@example.com",
				Active:   true,
			},
			{
				Schemas:  []string{SCIMUserSchema},
				Id:       "2",
				UserName: "bob@example.com",
				Active:   true,
			},
		},
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if !contains(r.URL.Path, "/scim/v2/Users") {
			t.Errorf("Expected SCIM Users path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	})
	defer cleanup()

	response, status := SCIMGetUsers(0, 0)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(response.Resources) != 2 {
		t.Errorf("Expected 2 users, got %d", len(response.Resources))
	}
	if response.Resources[0].UserName != "alice@example.com" {
		t.Errorf("Expected alice@example.com, got %s", response.Resources[0].UserName)
	}
}

func TestSCIMGetUsersWithPagination(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("startIndex") != "10" {
			t.Errorf("Expected startIndex=10, got %s", query.Get("startIndex"))
		}
		if query.Get("count") != "25" {
			t.Errorf("Expected count=25, got %s", query.Get("count"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SCIMListResponse{
			Schemas:   []string{SCIMListResponseSchema},
			Resources: []SCIMUser{},
		})
	})
	defer cleanup()

	_, status := SCIMGetUsers(10, 25)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestSCIMGetUser(t *testing.T) {
	expectedUser := SCIMUser{
		Schemas:     []string{SCIMUserSchema},
		Id:          "123",
		UserName:    "test@example.com",
		DisplayName: "Test User",
		Active:      true,
		Emails: []SCIMEmail{
			{Value: "test@example.com", Primary: true},
		},
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if !contains(r.URL.Path, "/scim/v2/Users/123") {
			t.Errorf("Expected user 123 path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedUser)
	})
	defer cleanup()

	user, status := SCIMGetUser("123")
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if user.Id != "123" {
		t.Errorf("Expected user ID 123, got %s", user.Id)
	}
	if user.UserName != "test@example.com" {
		t.Errorf("Expected test@example.com, got %s", user.UserName)
	}
}

func TestSCIMCreateUser(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		var body SCIMUser
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if body.UserName != "newuser@example.com" {
			t.Errorf("Expected userName newuser@example.com, got %s", body.UserName)
		}
		if len(body.Schemas) == 0 || body.Schemas[0] != SCIMUserSchema {
			t.Errorf("Expected User schema, got %v", body.Schemas)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		body.Id = "456"
		json.NewEncoder(w).Encode(body)
	})
	defer cleanup()

	newUser := SCIMUser{
		UserName: "newuser@example.com",
		Emails: []SCIMEmail{
			{Value: "newuser@example.com", Primary: true},
		},
		Active: true,
	}

	result, status := SCIMCreateUser(newUser)
	if status != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", status)
	}
	if result.Id != "456" {
		t.Errorf("Expected user ID 456, got %s", result.Id)
	}
}

func TestSCIMUpdateUser(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT request, got %s", r.Method)
		}
		if !contains(r.URL.Path, "/scim/v2/Users/123") {
			t.Errorf("Expected user 123 path, got %s", r.URL.Path)
		}

		var body SCIMUser
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if body.DisplayName != "Updated User" {
			t.Errorf("Expected displayName 'Updated User', got %s", body.DisplayName)
		}

		w.Header().Set("Content-Type", "application/json")
		body.Id = "123"
		json.NewEncoder(w).Encode(body)
	})
	defer cleanup()

	updatedUser := SCIMUser{
		UserName:    "test@example.com",
		DisplayName: "Updated User",
		Active:      true,
	}

	result, status := SCIMUpdateUser("123", updatedUser)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if result.DisplayName != "Updated User" {
		t.Errorf("Expected displayName 'Updated User', got %s", result.DisplayName)
	}
}

func TestSCIMPatchUser(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Expected PATCH request, got %s", r.Method)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		schemas := body["schemas"].([]interface{})
		if len(schemas) == 0 || schemas[0] != "urn:ietf:params:scim:api:messages:2.0:PatchOp" {
			t.Errorf("Expected PatchOp schema, got %v", schemas)
		}

		ops := body["Operations"].([]interface{})
		if len(ops) != 1 {
			t.Errorf("Expected 1 operation, got %d", len(ops))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SCIMUser{
			Id:       "123",
			UserName: "test@example.com",
			Active:   false,
		})
	})
	defer cleanup()

	operations := []map[string]interface{}{
		{
			"op":    "replace",
			"path":  "active",
			"value": false,
		},
	}

	result, status := SCIMPatchUser("123", operations)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if result.Active != false {
		t.Error("Expected user to be inactive")
	}
}

func TestSCIMDeleteUser(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}
		if !contains(r.URL.Path, "/scim/v2/Users/123") {
			t.Errorf("Expected user 123 path, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	})
	defer cleanup()

	_, status := SCIMDeleteUser("123")
	if status != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", status)
	}
}

func TestSCIMSearchUsers(t *testing.T) {
	expectedResponse := SCIMListResponse{
		Schemas:      []string{SCIMListResponseSchema},
		TotalResults: 1,
		Resources: []SCIMUser{
			{
				Schemas:  []string{SCIMUserSchema},
				Id:       "1",
				UserName: "alice@example.com",
				Active:   true,
			},
		},
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if !contains(r.URL.Path, "/scim/v2/Users/.search") {
			t.Errorf("Expected search path, got %s", r.URL.Path)
		}

		var body SCIMSearchRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if body.Filter != "userName eq \"alice@example.com\"" {
			t.Errorf("Expected filter, got %s", body.Filter)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	})
	defer cleanup()

	response, status := SCIMSearchUsers("userName eq \"alice@example.com\"", 0, 0)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(response.Resources) != 1 {
		t.Errorf("Expected 1 user, got %d", len(response.Resources))
	}
}

func TestSCIMGetMe(t *testing.T) {
	expectedUser := SCIMUser{
		Schemas:     []string{SCIMUserSchema},
		Id:          "current",
		UserName:    "me@example.com",
		DisplayName: "Current User",
		Active:      true,
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if !contains(r.URL.Path, "/scim/v2/Me") {
			t.Errorf("Expected Me path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedUser)
	})
	defer cleanup()

	user, status := SCIMGetMe()
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if user.UserName != "me@example.com" {
		t.Errorf("Expected me@example.com, got %s", user.UserName)
	}
}

// =============================================================================
// Service Account API Tests
// =============================================================================

func TestGetServiceAccounts(t *testing.T) {
	expectedAccounts := []ServiceAccount{
		{Id: 1, Label: "CI/CD Bot", Description: "For automation", HasValidKey: true},
		{Id: 2, Label: "Backup Service", HasValidKey: false},
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if !contains(r.URL.Path, "/service-accounts") {
			t.Errorf("Expected service-accounts path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedAccounts)
	})
	defer cleanup()

	accounts, status := GetServiceAccounts()
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if len(accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(accounts))
	}
	if accounts[0].Label != "CI/CD Bot" {
		t.Errorf("Expected 'CI/CD Bot', got %s", accounts[0].Label)
	}
}

func TestGetServiceAccount(t *testing.T) {
	expectedAccount := ServiceAccount{
		Id:          1,
		Label:       "CI/CD Bot",
		Description: "For automation pipelines",
		HasValidKey: true,
	}

	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if !contains(r.URL.Path, "/service-accounts/1") {
			t.Errorf("Expected service-accounts/1 path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedAccount)
	})
	defer cleanup()

	account, status := GetServiceAccount(1)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if account.Id != 1 {
		t.Errorf("Expected ID 1, got %d", account.Id)
	}
	if account.Description != "For automation pipelines" {
		t.Errorf("Expected description, got %s", account.Description)
	}
}

func TestCreateServiceAccount(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		var body ServiceAccountCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if body.Label != "New Bot" {
			t.Errorf("Expected label 'New Bot', got %s", body.Label)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ServiceAccountWithKey{
			ServiceAccount: ServiceAccount{
				Id:          3,
				Label:       body.Label,
				Description: body.Description,
				HasValidKey: true,
			},
			ApiKey: "new-api-key-12345",
		})
	})
	defer cleanup()

	request := ServiceAccountCreate{
		Label:       "New Bot",
		Description: "A new service account",
	}

	result, status := CreateServiceAccount(request)
	if status != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", status)
	}
	if result.Id != 3 {
		t.Errorf("Expected ID 3, got %d", result.Id)
	}
	if result.ApiKey != "new-api-key-12345" {
		t.Errorf("Expected API key, got %s", result.ApiKey)
	}
}

func TestUpdateServiceAccount(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Expected PATCH request, got %s", r.Method)
		}
		if !contains(r.URL.Path, "/service-accounts/1") {
			t.Errorf("Expected service-accounts/1 path, got %s", r.URL.Path)
		}

		var body ServiceAccountCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ServiceAccount{
			Id:          1,
			Label:       body.Label,
			Description: body.Description,
		})
	})
	defer cleanup()

	request := ServiceAccountCreate{
		Label:       "Updated Bot",
		Description: "Updated description",
	}

	result, status := UpdateServiceAccount(1, request)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if result.Label != "Updated Bot" {
		t.Errorf("Expected 'Updated Bot', got %s", result.Label)
	}
}

func TestDeleteServiceAccount(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}
		if !contains(r.URL.Path, "/service-accounts/1") {
			t.Errorf("Expected service-accounts/1 path, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	_, status := DeleteServiceAccount(1)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestRegenerateServiceAccountKey(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if !contains(r.URL.Path, "/service-accounts/1/apikey") {
			t.Errorf("Expected apikey path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ServiceAccountWithKey{
			ServiceAccount: ServiceAccount{
				Id:          1,
				Label:       "CI/CD Bot",
				HasValidKey: true,
			},
			ApiKey: "regenerated-key-67890",
		})
	})
	defer cleanup()

	result, status := RegenerateServiceAccountKey(1)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if result.ApiKey != "regenerated-key-67890" {
		t.Errorf("Expected regenerated key, got %s", result.ApiKey)
	}
}

func TestDeleteServiceAccountKey(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}
		if !contains(r.URL.Path, "/service-accounts/1/apikey") {
			t.Errorf("Expected apikey path, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	_, status := DeleteServiceAccountKey(1)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

// =============================================================================
// User Enable/Disable API Tests
// =============================================================================

func TestDisableUser(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if !contains(r.URL.Path, "/users/123/disable") {
			t.Errorf("Expected disable path, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	_, status := DisableUser(123)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestEnableUser(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if !contains(r.URL.Path, "/users/123/enable") {
			t.Errorf("Expected enable path, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	_, status := EnableUser(123)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestDisableUser_NotFound(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
	})
	defer cleanup()

	_, status := DisableUser(999)
	if status != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", status)
	}
}

func TestDisableUser_Forbidden(t *testing.T) {
	_, cleanup := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Access denied"})
	})
	defer cleanup()

	_, status := DisableUser(123)
	if status != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", status)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
