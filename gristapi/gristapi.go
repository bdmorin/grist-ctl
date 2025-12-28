// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

// Grist API operation
package gristapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Grist's user
type User struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Access       string `json:"access"`
	ParentAccess string `json:"parentAccess"`
}

// Grist's Organization
type Org struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	Domain    string `json:"domain"`
	CreatedAt string `json:"createdAt"`
}

// Grist's workspace
type Workspace struct {
	Id                 int    `json:"id"`
	Name               string `json:"name"`
	CreatedAt          string `json:"createdAt"`
	Docs               []Doc  `json:"docs"`
	IsSupportWorkspace string `json:"isSupportWorkspace"`
	OrgDomain          string `json:"orgDomain"`
	Org                Org    `json:"org"`
	Access             string `json:"access"`
}

type EntityAccess struct {
	MaxInheritedRole string `json:"maxInheritedRole"`
	Users            []User `json:"users"`
}

// Grist's document
type Doc struct {
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	IsPinned  bool      `json:"isPinned"`
	Workspace Workspace `json:"workspace"`
}

// Grist's table
type Table struct {
	Id string `json:"id"`
}

// List of Grist's tables
type Tables struct {
	Tables []Table `json:"tables"`
}

// Grist's table column
type TableColumn struct {
	Id string `json:"id"`
}

// List of Grist's table columns
type TableColumns struct {
	Columns []TableColumn `json:"columns"`
}

// Grist's table row
type TableRows struct {
	Id []uint `json:"id"`
}

// Record represents a single record with its fields
type Record struct {
	Id     int                    `json:"id,omitempty"`
	Fields map[string]interface{} `json:"fields"`
}

// RecordsList represents a list of records returned by GET /records
type RecordsList struct {
	Records []Record `json:"records"`
}

// RecordWithRequire represents a record for upsert operations
type RecordWithRequire struct {
	Require map[string]interface{} `json:"require"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

// RecordsWithRequire represents records for PUT (upsert) operations
type RecordsWithRequire struct {
	Records []RecordWithRequire `json:"records"`
}

// RecordsWithoutId represents records without IDs (for POST/add operations)
type RecordsWithoutId struct {
	Records []struct {
		Fields map[string]interface{} `json:"fields"`
	} `json:"records"`
}

// RecordsWithoutFields represents the response from adding records (IDs only)
type RecordsWithoutFields struct {
	Records []struct {
		Id int `json:"id"`
	} `json:"records"`
}

// RecordsDeleteRequest represents the request body for deleting records
type RecordsDeleteRequest []int

// GetRecordsOptions contains query parameters for fetching records
type GetRecordsOptions struct {
	Filter map[string][]interface{} // Filter by column values
	Sort   string                   // Column(s) to sort by, e.g. "name,-age"
	Limit  int                      // Maximum records to return
	Hidden bool                     // Include hidden columns
}

// AddRecordsOptions contains query parameters for adding records
type AddRecordsOptions struct {
	NoParse bool // Don't parse strings into column types
}

// UpdateRecordsOptions contains query parameters for updating records
type UpdateRecordsOptions struct {
	NoParse bool // Don't parse strings into column types
}

// UpsertRecordsOptions contains query parameters for upserting records
type UpsertRecordsOptions struct {
	OnMany            string // How to handle multiple matches: "first", "none", "all"
	NoAdd             bool   // Don't add new records
	NoUpdate          bool   // Don't update existing records
	AllowEmptyRequire bool   // Allow matching all records with empty require
	NoParse           bool   // Don't parse strings into column types
}

// Grist's user role
type UserRole struct {
	Email string
	Role  string
}

// Grist's organization usage
type OrgUsage struct {
	CountsByDataLimitStatus DataLimitStatus `json:"CountsByDataLimitStatus"`
	Attachments             Attachment      `json:"attachments"`
}

// Grist's data limit status
type DataLimitStatus struct {
	ApproachingLimit int
	GracePeriod      int
	DeleteOnly       int
}

// Grist's attachment
type Attachment struct {
	TotalBytes int `json:"totalBytes"`
}

// Apply config and return the config file path
func GetConfig() string {
	home := os.Getenv("HOME")
	configFile := filepath.Join(home, ".gristctl")
	if os.Getenv("GRIST_TOKEN") == "" || os.Getenv("GRIST_URL") == "" {
		err := godotenv.Load(configFile)
		if err != nil {
			fmt.Printf("Error reading configuration file : %s\n", err)
		}
	}
	return configFile
}

func init() {
	GetConfig()
}

// Sending an HTTP request to Grist's REST API
// Action: GET, POST, PATCH, DELETE
// Returns response body
func httpRequest(action string, myRequest string, data *bytes.Buffer) (string, int) {
	client := &http.Client{}
	url := fmt.Sprintf("%s/api/%s", os.Getenv("GRIST_URL"), myRequest)
	bearer := "Bearer " + os.Getenv("GRIST_TOKEN")

	req, err := http.NewRequest(action, url, data)
	if err != nil {
		log.Fatalf("Error creating request %s: %s", url, err)
	}
	req.Header.Add("Authorization", bearer)
	req.Header.Set("Content-Type", "application/json")

	// Send the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("Error sending request %s: %s", url, err)
		return errMsg, -10
	} else {
		defer resp.Body.Close()
		// Read the HTTP response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response %s: %s", url, err)
		}
		return string(body), resp.StatusCode
	}
}

// Send an HTTP GET request to Grist's REST API
// Returns the response body
func httpGet(myRequest string, data string) (string, int) {
	dataBody := bytes.NewBuffer([]byte(data))
	body, status := httpRequest("GET", myRequest, dataBody)
	// if status != http.StatusOK {
	// 	fmt.Printf("Return code from %s : %d (%s)\n", myRequest, status, body)
	// }
	return body, status
}

// Test Grist API connection
func TestConnection() bool {
	_, status := httpGet("orgs", "")
	return status == http.StatusOK
}

// Sends an HTTP POST request to Grist's REST API with a data load
// Return the response body
func httpPost(myRequest string, data string) (string, int) {
	dataBody := bytes.NewBuffer([]byte(data))
	body, status := httpRequest("POST", myRequest, dataBody)
	return body, status
}

// Sends an HTTP PATCH request to Grist's REST API with a data load
// Return the response body
func httpPatch(myRequest string, data string) (string, int) {
	dataBody := bytes.NewBuffer([]byte(data))
	body, status := httpRequest("PATCH", myRequest, dataBody)
	return body, status
}

// Send an HTTP DELETE request to Grist's REST API with a data load
// Return the response body
func httpDelete(myRequest string, data string) (string, int) {
	dataBody := bytes.NewBuffer([]byte(data))
	body, status := httpRequest("DELETE", myRequest, dataBody)
	return body, status
}

// Send an HTTP PUT request to Grist's REST API with a data load
// Return the response body
func httpPut(myRequest string, data string) (string, int) {
	dataBody := bytes.NewBuffer([]byte(data))
	body, status := httpRequest("PUT", myRequest, dataBody)
	return body, status
}

// Retrieves the list of organizations
func GetOrgs() []Org {
	myOrgs := []Org{}
	response, _ := httpGet("orgs", "")
	json.Unmarshal([]byte(response), &myOrgs)
	return myOrgs
}

// Retrieves the organization whose identifier is passed in parameter
func GetOrg(idOrg string) Org {
	myOrg := Org{}
	response, _ := httpGet("orgs/"+idOrg, "")
	json.Unmarshal([]byte(response), &myOrg)
	return myOrg
}

// Retrieves the list of users in the organization whose ID is passed in parameter
func GetOrgAccess(idOrg string) []User {
	var lstUsers EntityAccess
	url := fmt.Sprintf("orgs/%s/access", idOrg)
	response, _ := httpGet(url, "")
	json.Unmarshal([]byte(response), &lstUsers)
	return lstUsers.Users
}

// Retrieves information on a specific organization
func GetOrgWorkspaces(orgId int) []Workspace {
	lstWorkspaces := []Workspace{}
	response, _ := httpGet("orgs/"+strconv.Itoa(orgId)+"/workspaces", "")
	json.Unmarshal([]byte(response), &lstWorkspaces)
	return lstWorkspaces
}

// Get a workspace
func GetWorkspace(workspaceId int) Workspace {
	workspace := Workspace{}
	url := fmt.Sprintf("workspaces/%d", workspaceId)
	response, returnCode := httpGet(url, "")
	if returnCode == http.StatusOK {
		json.Unmarshal([]byte(response), &workspace)
	}
	return workspace
}

// Delete an organization
func DeleteOrg(orgId int, orgName string) {
	url := fmt.Sprintf("orgs/%d/%s", orgId, orgName)
	response, status := httpDelete(url, "")
	if status == http.StatusOK {
		fmt.Printf("Organization %d : %s deleted\t✅\n", orgId, orgName)
	} else {
		fmt.Printf("Unable to delete organization %d : %s : %s ❗️\n", orgId, orgName, response)
	}
}

// Delete a workspace
func DeleteWorkspace(workspaceId int) {
	url := fmt.Sprintf("workspaces/%d", workspaceId)
	response, status := httpDelete(url, "")
	if status == http.StatusOK {
		fmt.Printf("Workspace %d deleted\t✅\n", workspaceId)
	} else {
		fmt.Printf("Unable to delete workspace %d : %s ❗️\n", workspaceId, response)
	}
}

// Delete a document
func DeleteDoc(docId string) {
	url := fmt.Sprintf("docs/%s", docId)
	response, status := httpDelete(url, "")
	if status == http.StatusOK {
		fmt.Printf("Document %s deleted\t✅\n", docId)
	} else {
		fmt.Printf("Unable to delete document %s : %s ❗️", docId, response)
	}
}

// Delete a user
func DeleteUser(userId int) {
	url := fmt.Sprintf("users/%d", userId)
	response, status := httpDelete(url, `{"name": ""}`)

	var message string
	switch status {
	case 200:
		message = "The account has been deleted successfully"
	case 400:
		message = "The passed user name does not match the one retrieved from the database given the passed user id"
	case 403:
		message = "The caller is not allowed to delete this account"
	case 404:
		message = "The user is not found"
	}
	fmt.Println(message)
	if status != http.StatusOK {
		fmt.Printf("ERREUR: %s\n", response)
	}
}

// Workspace access rights query
func GetWorkspaceAccess(workspaceId int) EntityAccess {
	workspaceAccess := EntityAccess{}
	url := fmt.Sprintf("workspaces/%d/access", workspaceId)
	response, _ := httpGet(url, "")
	json.Unmarshal([]byte(response), &workspaceAccess)
	return workspaceAccess
}

// Retrieves information about a specific document
func GetDoc(docId string) Doc {
	doc := Doc{}
	url := "docs/" + docId
	response, _ := httpGet(url, "")
	json.Unmarshal([]byte(response), &doc)
	return doc
}

// Retrieves the list of tables contained in a document
func GetDocTables(docId string) Tables {
	tables := Tables{}
	url := "docs/" + docId + "/tables"
	response, _ := httpGet(url, "")
	json.Unmarshal([]byte(response), &tables)

	return tables
}

// Retrieves a list of table columns
func GetTableColumns(docId string, tableId string) TableColumns {
	columns := TableColumns{}
	url := "docs/" + docId + "/tables/" + tableId + "/columns"
	response, _ := httpGet(url, "")
	json.Unmarshal([]byte(response), &columns)

	return columns
}

// Retrieves records from a table
func GetTableRows(docId string, tableId string) TableRows {
	rows := TableRows{}
	url := "docs/" + docId + "/tables/" + tableId + "/data"
	response, _ := httpGet(url, "")
	json.Unmarshal([]byte(response), &rows)

	return rows
}

// Returns the list of users with access to the document
func GetDocAccess(docId string) EntityAccess {
	var lstUsers EntityAccess
	url := fmt.Sprintf("docs/%s/access", docId)
	response, _ := httpGet(url, "")
	json.Unmarshal([]byte(response), &lstUsers)
	return lstUsers
}

// Move all documents from a workspace to another
func MoveAllDocs(fromWorkspaceId int, toWorkspaceId int) {
	// Getting the workspaces
	from_ws := GetWorkspace(fromWorkspaceId)
	to_ws := GetWorkspace(toWorkspaceId)
	if from_ws.Id == 0 {
		fmt.Printf("❗️ Workspace %d not found ❗️\n", fromWorkspaceId)
	} else if to_ws.Id == 0 {
		fmt.Printf("❗️ Workspace %d not found ❗️\n", toWorkspaceId)
	} else {
		// Workspaces were found
		for _, doc := range from_ws.Docs {
			url := "docs/" + doc.Id + "/move"
			data := fmt.Sprintf(`{"workspace": "%d"}`, toWorkspaceId)
			_, status := httpPatch(url, data)
			if status == http.StatusOK {
				fmt.Printf("Document %s moved to workspace %d ✅\n", doc.Id, toWorkspaceId)
			} else {
				fmt.Printf("Unable to move document %s", doc.Id)
			}
		}
	}
}

// Move a document in a workspace
func MoveDoc(docId string, workspaceId int) {
	url := "docs/" + docId + "/move"
	data := fmt.Sprintf(`{"workspace": "%d"}`, workspaceId)
	_, status := httpPatch(url, data)
	if status == http.StatusOK {
		fmt.Printf("Document moved to workspace %d ✅\n", workspaceId)
	} else {
		fmt.Printf("Unable to move document")
	}
}

// Purge a document's history, to retain only the last modifications
func PurgeDoc(docId string, nbHisto int) {
	url := "docs/" + docId + "/states/remove"
	data := fmt.Sprintf(`{"keep": "%d"}`, nbHisto)
	_, status := httpPost(url, data)
	if status == http.StatusOK {
		fmt.Printf("History cleared (%d last states) ✅\n", nbHisto)
	}
}

// Import a list of user & role into a workspace
// Search workspace by name in org
func ImportUsers(orgId int, workspaceName string, users []UserRole) {
	lstWorkspaces := GetOrgWorkspaces(orgId)
	idWorkspace := 0
	for _, ws := range lstWorkspaces {
		if ws.Name == workspaceName {
			idWorkspace = ws.Id
		}
	}

	if idWorkspace == 0 {
		idWorkspace = CreateWorkspace(orgId, workspaceName)
	}
	if idWorkspace == 0 {
		fmt.Printf("Unable to create workspace %s\n", workspaceName)
	} else {
		url := fmt.Sprintf("workspaces/%d/access", idWorkspace)

		roleLine := []string{}
		for _, role := range users {
			roleLine = append(roleLine, fmt.Sprintf(`"%s": "%s"`, role.Email, role.Role))
		}
		patch := fmt.Sprintf(`{	"delta": { "users": {%s}}}`, strings.Join(roleLine, ","))

		body, status := httpPatch(url, patch)

		var result string
		if status == http.StatusOK {
			result = "✅"
		} else {
			result = fmt.Sprintf("❗️ (%s)", body)
		}
		fmt.Printf("Import %d users in workspace n°%d\t : %s\n", len(users), idWorkspace, result)
	}

}

// Create an organization
func CreateOrg(orgName string, orgDomain string) int {
	url := fmt.Sprintf("orgs")
	data := fmt.Sprintf(`{"name":"%s", "domain":"%s"}`, orgName, orgDomain)
	body, status := httpPost(url, data)
	idOrg := 0
	if status == http.StatusOK {
		id, err := strconv.Atoi(body)
		if err == nil {
			idOrg = id
		}
	}
	return idOrg
}

// Create a workspace in an organization
func CreateWorkspace(orgId int, workspaceName string) int {
	url := fmt.Sprintf("orgs/%d/workspaces", orgId)
	data := fmt.Sprintf(`{"name":"%s"}`, workspaceName)
	body, status := httpPost(url, data)
	idWorkspace := 0
	if status == http.StatusOK {
		id, err := strconv.Atoi(body)
		if err == nil {
			idWorkspace = id
		}
	}
	return idWorkspace
}

// Export doc in Grist format (Sqlite) in fileName file
func ExportDocGrist(docId string, fileName string) {
	url := fmt.Sprintf("docs/%s/download", docId)
	export, returnCode := httpGet(url, "")
	if returnCode == http.StatusOK {
		f, e := os.Create(fileName)
		if e != nil {
			panic(e)
		}
		defer f.Close()
		fmt.Fprintln(f, export)
	}
}

// Export doc in Excel format (XLSX) in fileName file
func ExportDocExcel(docId string, fileName string) {
	url := fmt.Sprintf("docs/%s/download/xlsx", docId)
	export, returnCode := httpGet(url, "")
	if returnCode == http.StatusOK {
		f, e := os.Create(fileName)
		if e != nil {
			panic(e)
		}
		defer f.Close()
		fmt.Fprintln(f, export)
	}
}

// Returns table content as Dataframe
func GetTableContent(docId string, tableName string) {
	url := fmt.Sprintf("docs/%s/download/csv?tableId=%s", docId, tableName)
	csvFile, _ := httpGet(url, "")
	fmt.Println(csvFile)
}

// Retrieves information on a specific organization
func GetOrgUsageSummary(orgId string) OrgUsage {
	usage := OrgUsage{}
	response, _ := httpGet("orgs/"+orgId+"/usage", "")
	json.Unmarshal([]byte(response), &usage)
	return usage
}

<<<<<<< HEAD

// buildRecordsQueryParams builds the query string for records API endpoints
func buildRecordsQueryParams(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	parts := []string{}
	for key, value := range params {
		if value != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", key, value))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "?" + strings.Join(parts, "&")
}

// GetRecords fetches records from a table
// GET /docs/{docId}/tables/{tableId}/records
func GetRecords(docId string, tableId string, options *GetRecordsOptions) (RecordsList, int) {
	records := RecordsList{}
	params := make(map[string]string)

	if options != nil {
		if options.Filter != nil {
			filterJSON, err := json.Marshal(options.Filter)
			if err == nil {
				params["filter"] = string(filterJSON)
			}
		}
		if options.Sort != "" {
			params["sort"] = options.Sort
		}
		if options.Limit > 0 {
			params["limit"] = strconv.Itoa(options.Limit)
		}
		if options.Hidden {
			params["hidden"] = "true"
		}
	}

	url := fmt.Sprintf("docs/%s/tables/%s/records%s", docId, tableId, buildRecordsQueryParams(params))
	response, status := httpGet(url, "")
	if status == http.StatusOK {
		json.Unmarshal([]byte(response), &records)
	}
	return records, status
}

// AddRecords adds records to a table
// POST /docs/{docId}/tables/{tableId}/records
func AddRecords(docId string, tableId string, records []map[string]interface{}, options *AddRecordsOptions) (RecordsWithoutFields, int) {
	result := RecordsWithoutFields{}
	params := make(map[string]string)

	if options != nil && options.NoParse {
		params["noparse"] = "true"
	}

	// Build request body
	body := struct {
		Records []struct {
			Fields map[string]interface{} `json:"fields"`
		} `json:"records"`
	}{}
	for _, fields := range records {
		body.Records = append(body.Records, struct {
			Fields map[string]interface{} `json:"fields"`
		}{Fields: fields})
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return result, -1
	}

	url := fmt.Sprintf("docs/%s/tables/%s/records%s", docId, tableId, buildRecordsQueryParams(params))
	response, status := httpPost(url, string(bodyJSON))
	if status == http.StatusOK {
		json.Unmarshal([]byte(response), &result)
	}
	return result, status
}

// UpdateRecords modifies records in a table
// PATCH /docs/{docId}/tables/{tableId}/records
func UpdateRecords(docId string, tableId string, records []Record, options *UpdateRecordsOptions) (string, int) {
	params := make(map[string]string)

	if options != nil && options.NoParse {
		params["noparse"] = "true"
	}

	// Build request body
	body := struct {
		Records []Record `json:"records"`
	}{Records: records}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", -1
	}

	url := fmt.Sprintf("docs/%s/tables/%s/records%s", docId, tableId, buildRecordsQueryParams(params))
	response, status := httpPatch(url, string(bodyJSON))
	return response, status
}

// UpsertRecords adds or updates records in a table (upsert)
// PUT /docs/{docId}/tables/{tableId}/records
func UpsertRecords(docId string, tableId string, records []RecordWithRequire, options *UpsertRecordsOptions) (string, int) {
	params := make(map[string]string)

	if options != nil {
		if options.OnMany != "" {
			params["onmany"] = options.OnMany
		}
		if options.NoAdd {
			params["noadd"] = "true"
		}
		if options.NoUpdate {
			params["noupdate"] = "true"
		}
		if options.AllowEmptyRequire {
			params["allow_empty_require"] = "true"
		}
		if options.NoParse {
			params["noparse"] = "true"
		}
	}

	// Build request body
	body := struct {
		Records []RecordWithRequire `json:"records"`
	}{Records: records}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", -1
	}

	url := fmt.Sprintf("docs/%s/tables/%s/records%s", docId, tableId, buildRecordsQueryParams(params))
	response, status := httpPut(url, string(bodyJSON))
	return response, status
}

// DeleteRecords deletes records from a table
// POST /docs/{docId}/tables/{tableId}/records/delete
func DeleteRecords(docId string, tableId string, recordIds []int) (string, int) {
	bodyJSON, err := json.Marshal(recordIds)
	if err != nil {
		return "", -1
	}

	url := fmt.Sprintf("docs/%s/tables/%s/records/delete", docId, tableId)
	response, status := httpPost(url, string(bodyJSON))
	return response, status
}
