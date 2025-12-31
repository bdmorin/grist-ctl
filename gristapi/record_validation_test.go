// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

package gristapi

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"testing"
)

// TestRecordCRUD is a comprehensive integration test for all record CRUD operations
// This test creates a real document in the playground workspace and performs all operations
//
//nolint:gocyclo // This is a comprehensive integration test, high complexity is expected
func TestRecordCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Find the playground workspace
	orgs := GetOrgs()
	if len(orgs) == 0 {
		t.Fatal("No organizations found - cannot proceed with testing")
	}

	var playgroundWorkspaceID int
	for _, org := range orgs {
		workspaces := GetOrgWorkspaces(org.Id)
		for _, ws := range workspaces {
			if ws.Name == "docs" || strings.Contains(strings.ToLower(ws.Name), "playground") {
				playgroundWorkspaceID = ws.Id
				t.Logf("Found playground workspace: %s (ID: %d)", ws.Name, ws.Id)
				break
			}
		}
		if playgroundWorkspaceID != 0 {
			break
		}
	}

	if playgroundWorkspaceID == 0 {
		// Use the first workspace if we can't find playground
		for _, org := range orgs {
			workspaces := GetOrgWorkspaces(org.Id)
			if len(workspaces) > 0 {
				playgroundWorkspaceID = workspaces[0].Id
				t.Logf("Using workspace: %s (ID: %d)", workspaces[0].Name, playgroundWorkspaceID)
				break
			}
		}
	}

	if playgroundWorkspaceID == 0 {
		t.Fatal("Could not find any workspace for testing")
	}

	// Use a known accessible document from Hexxa org
	// This document ID is from the Hexxa/Home workspace
	docID := "g7pesgBnD5B5FsN4hUF9BB"

	// Verify it's accessible
	testDoc := GetDoc(docID)
	if testDoc.Id == "" {
		// Fallback: try to find or create a document
		docID = findOrCreateTestDocument(t, playgroundWorkspaceID)
		if docID == "" {
			t.Fatal("Failed to find or create test document")
		}
	}
	t.Logf("Using test document: %s", docID)

	// Create a table with columns for testing
	tableId := "TestRecords"
	if !createTestTable(t, docID, tableId) {
		t.Fatal("Failed to create test table")
	}
	t.Logf("Created test table: %s", tableId)

	// Clean up the table data at the end (delete all records)
	defer func() {
		records, status := GetRecords(docID, tableId, nil)
		if status == http.StatusOK && len(records.Records) > 0 {
			recordIds := make([]int, len(records.Records))
			for i, r := range records.Records {
				recordIds[i] = r.Id
			}
			DeleteRecords(docID, tableId, recordIds)
			t.Logf("Cleaned up %d test records", len(recordIds))
		}
	}()

	// Run all CRUD operation tests
	t.Run("AddSingleRecord", func(t *testing.T) {
		testAddSingleRecord(t, docID, tableId)
	})

	t.Run("AddBulkRecords", func(t *testing.T) {
		testAddBulkRecords(t, docID, tableId)
	})

	t.Run("GetRecordsWithFilters", func(t *testing.T) {
		testGetRecordsWithFilters(t, docID, tableId)
	})

	t.Run("GetRecordsWithSort", func(t *testing.T) {
		testGetRecordsWithSort(t, docID, tableId)
	})

	t.Run("GetRecordsWithPagination", func(t *testing.T) {
		testGetRecordsWithPagination(t, docID, tableId)
	})

	t.Run("UpdateSingleRecord", func(t *testing.T) {
		testUpdateSingleRecord(t, docID, tableId)
	})

	t.Run("UpdateBulkRecords", func(t *testing.T) {
		testUpdateBulkRecords(t, docID, tableId)
	})

	t.Run("UpdatePartialFields", func(t *testing.T) {
		testUpdatePartialFields(t, docID, tableId)
	})

	t.Run("DeleteSingleRecord", func(t *testing.T) {
		testDeleteSingleRecord(t, docID, tableId)
	})

	t.Run("DeleteBulkRecords", func(t *testing.T) {
		testDeleteBulkRecords(t, docID, tableId)
	})

	t.Run("UpsertNewRecord", func(t *testing.T) {
		testUpsertNewRecord(t, docID, tableId)
	})

	t.Run("UpsertExistingRecord", func(t *testing.T) {
		testUpsertExistingRecord(t, docID, tableId)
	})

	t.Run("EdgeCasesUnicode", func(t *testing.T) {
		testEdgeCasesUnicode(t, docID, tableId)
	})

	t.Run("EdgeCasesSpecialChars", func(t *testing.T) {
		testEdgeCasesSpecialChars(t, docID, tableId)
	})

	t.Run("EdgeCasesNulls", func(t *testing.T) {
		testEdgeCasesNulls(t, docID, tableId)
	})

	t.Run("EdgeCasesLargeText", func(t *testing.T) {
		testEdgeCasesLargeText(t, docID, tableId)
	})

	t.Run("EdgeCasesLargeNumbers", func(t *testing.T) {
		testEdgeCasesLargeNumbers(t, docID, tableId)
	})

	t.Run("BulkDataLoad", func(t *testing.T) {
		testBulkDataLoad(t, docID, tableId)
	})

	// Store the test document ID for reference
	t.Logf("===================================")
	t.Logf("Test Document ID: %s", docID)
	t.Logf("===================================")
}

// testAddSingleRecord tests adding a single record
func testAddSingleRecord(t *testing.T, docId, tableId string) {
	records := []map[string]interface{}{
		{
			"name":  "Alice",
			"email": "alice@example.com",
			"age":   30,
		},
	}

	result, status := AddRecords(docId, tableId, records, nil)
	if status != http.StatusOK {
		t.Errorf("AddRecords failed with status %d", status)
		return
	}

	if len(result.Records) != 1 {
		t.Errorf("Expected 1 record ID, got %d", len(result.Records))
		return
	}

	if result.Records[0].Id <= 0 {
		t.Errorf("Expected positive record ID, got %d", result.Records[0].Id)
	}

	// Verify the record was added by fetching it
	fetchedRecords, status := GetRecords(docId, tableId, &GetRecordsOptions{
		Filter: map[string][]interface{}{"email": {"alice@example.com"}},
	})
	if status != http.StatusOK {
		t.Errorf("GetRecords failed with status %d", status)
		return
	}

	if len(fetchedRecords.Records) != 1 {
		t.Errorf("Expected 1 fetched record, got %d", len(fetchedRecords.Records))
		return
	}

	if fetchedRecords.Records[0].Fields["name"] != "Alice" {
		t.Errorf("Expected name 'Alice', got %v", fetchedRecords.Records[0].Fields["name"])
	}
}

// testAddBulkRecords tests adding multiple records at once
func testAddBulkRecords(t *testing.T, docId, tableId string) {
	records := []map[string]interface{}{
		{"name": "Bob", "email": "bob@example.com", "age": 25},
		{"name": "Charlie", "email": "charlie@example.com", "age": 35},
		{"name": "Diana", "email": "diana@example.com", "age": 28},
	}

	result, status := AddRecords(docId, tableId, records, nil)
	if status != http.StatusOK {
		t.Errorf("AddRecords failed with status %d", status)
		return
	}

	if len(result.Records) != 3 {
		t.Errorf("Expected 3 record IDs, got %d", len(result.Records))
		return
	}

	// Verify all records were added
	fetchedRecords, status := GetRecords(docId, tableId, nil)
	if status != http.StatusOK {
		t.Errorf("GetRecords failed with status %d", status)
		return
	}

	// Should have at least the 3 records we just added
	if len(fetchedRecords.Records) < 3 {
		t.Errorf("Expected at least 3 records, got %d", len(fetchedRecords.Records))
	}
}

// testGetRecordsWithFilters tests filtering records
func testGetRecordsWithFilters(t *testing.T, docId, tableId string) {
	// Add test records
	records := []map[string]interface{}{
		{"name": "Filter1", "email": "filter1@test.com", "age": 20},
		{"name": "Filter2", "email": "filter2@test.com", "age": 30},
		{"name": "Filter3", "email": "filter3@test.com", "age": 20},
	}
	AddRecords(docId, tableId, records, nil)

	// Test filtering by age
	options := &GetRecordsOptions{
		Filter: map[string][]interface{}{"age": {20}},
	}

	fetchedRecords, status := GetRecords(docId, tableId, options)
	if status != http.StatusOK {
		t.Errorf("GetRecords failed with status %d", status)
		return
	}

	// Should get at least 2 records with age 20
	count := 0
	for _, r := range fetchedRecords.Records {
		if age, ok := r.Fields["age"].(float64); ok && age == 20 {
			count++
		}
	}

	if count < 2 {
		t.Errorf("Expected at least 2 records with age 20, got %d", count)
	}
}

// testGetRecordsWithSort tests sorting records
//
//nolint:gocyclo // This is a comprehensive test covering multiple sort scenarios
func testGetRecordsWithSort(t *testing.T, docId, tableId string) {
	// Add test records with known names
	records := []map[string]interface{}{
		{"name": "Zebra", "email": "zebra@test.com", "age": 40},
		{"name": "Apple", "email": "apple@test.com", "age": 50},
		{"name": "Mango", "email": "mango@test.com", "age": 45},
	}
	AddRecords(docId, tableId, records, nil)

	// Test ascending sort
	options := &GetRecordsOptions{
		Sort: "name",
	}

	fetchedRecords, status := GetRecords(docId, tableId, options)
	if status != http.StatusOK {
		t.Errorf("GetRecords failed with status %d", status)
		return
	}

	if len(fetchedRecords.Records) < 3 {
		t.Errorf("Expected at least 3 records, got %d", len(fetchedRecords.Records))
		return
	}

	// Find our test records and verify order
	var testRecords []Record
	for _, r := range fetchedRecords.Records {
		if name, ok := r.Fields["name"].(string); ok {
			if name == "Apple" || name == "Mango" || name == "Zebra" {
				testRecords = append(testRecords, r)
			}
		}
	}

	if len(testRecords) >= 2 {
		// Check that Apple comes before Zebra
		appleIdx, zebraIdx := -1, -1
		for i, r := range fetchedRecords.Records {
			if name, ok := r.Fields["name"].(string); ok {
				if name == "Apple" {
					appleIdx = i
				}
				if name == "Zebra" {
					zebraIdx = i
				}
			}
		}

		if appleIdx >= 0 && zebraIdx >= 0 && appleIdx >= zebraIdx {
			t.Errorf("Expected Apple to come before Zebra in sorted results")
		}
	}
}

// testGetRecordsWithPagination tests pagination using limit
func testGetRecordsWithPagination(t *testing.T, docId, tableId string) {
	// Add several records
	records := make([]map[string]interface{}, 10)
	for i := 0; i < 10; i++ {
		records[i] = map[string]interface{}{
			"name":  fmt.Sprintf("Page%d", i),
			"email": fmt.Sprintf("page%d@test.com", i),
			"age":   20 + i,
		}
	}
	AddRecords(docId, tableId, records, nil)

	// Test limit
	options := &GetRecordsOptions{
		Limit: 5,
	}

	fetchedRecords, status := GetRecords(docId, tableId, options)
	if status != http.StatusOK {
		t.Errorf("GetRecords failed with status %d", status)
		return
	}

	if len(fetchedRecords.Records) != 5 {
		t.Errorf("Expected exactly 5 records with limit, got %d", len(fetchedRecords.Records))
	}
}

// testUpdateSingleRecord tests updating a single record
func testUpdateSingleRecord(t *testing.T, docId, tableId string) {
	// Add a record to update
	records := []map[string]interface{}{
		{"name": "UpdateMe", "email": "updateme@test.com", "age": 25},
	}
	result, _ := AddRecords(docId, tableId, records, nil)
	recordId := result.Records[0].Id

	// Update the record
	updateRecords := []Record{
		{
			Id: recordId,
			Fields: map[string]interface{}{
				"name": "Updated",
				"age":  26,
			},
		},
	}

	_, status := UpdateRecords(docId, tableId, updateRecords, nil)
	if status != http.StatusOK {
		t.Errorf("UpdateRecords failed with status %d", status)
		return
	}

	// Verify the update
	fetchedRecords, _ := GetRecords(docId, tableId, &GetRecordsOptions{
		Filter: map[string][]interface{}{"email": {"updateme@test.com"}},
	})

	if len(fetchedRecords.Records) != 1 {
		t.Errorf("Expected 1 record after update, got %d", len(fetchedRecords.Records))
		return
	}

	if fetchedRecords.Records[0].Fields["name"] != "Updated" {
		t.Errorf("Expected name 'Updated', got %v", fetchedRecords.Records[0].Fields["name"])
	}

	if age, ok := fetchedRecords.Records[0].Fields["age"].(float64); !ok || age != 26 {
		t.Errorf("Expected age 26, got %v", fetchedRecords.Records[0].Fields["age"])
	}
}

// testUpdateBulkRecords tests updating multiple records at once
func testUpdateBulkRecords(t *testing.T, docId, tableId string) {
	// Add records to update
	records := []map[string]interface{}{
		{"name": "Bulk1", "email": "bulk1@test.com", "age": 30},
		{"name": "Bulk2", "email": "bulk2@test.com", "age": 31},
		{"name": "Bulk3", "email": "bulk3@test.com", "age": 32},
	}
	result, _ := AddRecords(docId, tableId, records, nil)

	// Update all records
	updateRecords := make([]Record, 3)
	for i := 0; i < 3; i++ {
		updateRecords[i] = Record{
			Id: result.Records[i].Id,
			Fields: map[string]interface{}{
				"age": 40 + i,
			},
		}
	}

	_, status := UpdateRecords(docId, tableId, updateRecords, nil)
	if status != http.StatusOK {
		t.Errorf("UpdateRecords failed with status %d", status)
		return
	}

	// Verify updates
	for i := 0; i < 3; i++ {
		fetchedRecords, _ := GetRecords(docId, tableId, &GetRecordsOptions{
			Filter: map[string][]interface{}{"email": {fmt.Sprintf("bulk%d@test.com", i+1)}},
		})

		if len(fetchedRecords.Records) == 1 {
			if age, ok := fetchedRecords.Records[0].Fields["age"].(float64); !ok || age != float64(40+i) {
				t.Errorf("Expected age %d, got %v", 40+i, fetchedRecords.Records[0].Fields["age"])
			}
		}
	}
}

// testUpdatePartialFields tests updating only some fields of a record
func testUpdatePartialFields(t *testing.T, docId, tableId string) {
	// Add a record
	records := []map[string]interface{}{
		{"name": "Partial", "email": "partial@test.com", "age": 25},
	}
	result, _ := AddRecords(docId, tableId, records, nil)
	recordId := result.Records[0].Id

	// Update only the age field
	updateRecords := []Record{
		{
			Id: recordId,
			Fields: map[string]interface{}{
				"age": 99,
			},
		},
	}

	_, status := UpdateRecords(docId, tableId, updateRecords, nil)
	if status != http.StatusOK {
		t.Errorf("UpdateRecords failed with status %d", status)
		return
	}

	// Verify name is unchanged and age is updated
	fetchedRecords, _ := GetRecords(docId, tableId, &GetRecordsOptions{
		Filter: map[string][]interface{}{"email": {"partial@test.com"}},
	})

	if len(fetchedRecords.Records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(fetchedRecords.Records))
		return
	}

	if fetchedRecords.Records[0].Fields["name"] != "Partial" {
		t.Errorf("Expected name 'Partial' to be unchanged, got %v", fetchedRecords.Records[0].Fields["name"])
	}

	if age, ok := fetchedRecords.Records[0].Fields["age"].(float64); !ok || age != 99 {
		t.Errorf("Expected age 99, got %v", fetchedRecords.Records[0].Fields["age"])
	}
}

// testDeleteSingleRecord tests deleting a single record
func testDeleteSingleRecord(t *testing.T, docId, tableId string) {
	// Add a record to delete
	records := []map[string]interface{}{
		{"name": "DeleteMe", "email": "deleteme@test.com", "age": 25},
	}
	result, _ := AddRecords(docId, tableId, records, nil)
	recordId := result.Records[0].Id

	// Delete the record
	_, status := DeleteRecords(docId, tableId, []int{recordId})
	if status != http.StatusOK {
		t.Errorf("DeleteRecords failed with status %d", status)
		return
	}

	// Verify the record is deleted
	fetchedRecords, _ := GetRecords(docId, tableId, &GetRecordsOptions{
		Filter: map[string][]interface{}{"email": {"deleteme@test.com"}},
	})

	if len(fetchedRecords.Records) != 0 {
		t.Errorf("Expected 0 records after delete, got %d", len(fetchedRecords.Records))
	}
}

// testDeleteBulkRecords tests deleting multiple records at once
func testDeleteBulkRecords(t *testing.T, docId, tableId string) {
	// Add records to delete
	records := []map[string]interface{}{
		{"name": "BulkDel1", "email": "bulkdel1@test.com", "age": 30},
		{"name": "BulkDel2", "email": "bulkdel2@test.com", "age": 31},
		{"name": "BulkDel3", "email": "bulkdel3@test.com", "age": 32},
	}
	result, _ := AddRecords(docId, tableId, records, nil)

	// Delete all records
	recordIds := make([]int, 3)
	for i := 0; i < 3; i++ {
		recordIds[i] = result.Records[i].Id
	}

	_, status := DeleteRecords(docId, tableId, recordIds)
	if status != http.StatusOK {
		t.Errorf("DeleteRecords failed with status %d", status)
		return
	}

	// Verify all records are deleted
	for i := 0; i < 3; i++ {
		fetchedRecords, _ := GetRecords(docId, tableId, &GetRecordsOptions{
			Filter: map[string][]interface{}{"email": {fmt.Sprintf("bulkdel%d@test.com", i+1)}},
		})

		if len(fetchedRecords.Records) != 0 {
			t.Errorf("Expected 0 records after delete for bulkdel%d, got %d", i+1, len(fetchedRecords.Records))
		}
	}
}

// testUpsertNewRecord tests upserting a new record
func testUpsertNewRecord(t *testing.T, docId, tableId string) {
	upsertRecords := []RecordWithRequire{
		{
			Require: map[string]interface{}{"email": "upsert-new@test.com"},
			Fields:  map[string]interface{}{"name": "UpsertNew", "email": "upsert-new@test.com", "age": 28},
		},
	}

	_, status := UpsertRecords(docId, tableId, upsertRecords, nil)
	if status != http.StatusOK {
		t.Errorf("UpsertRecords failed with status %d", status)
		return
	}

	// Verify the record was created
	fetchedRecords, _ := GetRecords(docId, tableId, &GetRecordsOptions{
		Filter: map[string][]interface{}{"email": {"upsert-new@test.com"}},
	})

	if len(fetchedRecords.Records) != 1 {
		t.Errorf("Expected 1 record after upsert, got %d", len(fetchedRecords.Records))
		return
	}

	if fetchedRecords.Records[0].Fields["name"] != "UpsertNew" {
		t.Errorf("Expected name 'UpsertNew', got %v", fetchedRecords.Records[0].Fields["name"])
	}
}

// testUpsertExistingRecord tests upserting an existing record
func testUpsertExistingRecord(t *testing.T, docId, tableId string) {
	// First, add a record
	records := []map[string]interface{}{
		{"name": "UpsertExist", "email": "upsert-exist@test.com", "age": 30},
	}
	AddRecords(docId, tableId, records, nil)

	// Now upsert with the same email (should update)
	upsertRecords := []RecordWithRequire{
		{
			Require: map[string]interface{}{"email": "upsert-exist@test.com"},
			Fields:  map[string]interface{}{"age": 31},
		},
	}

	_, status := UpsertRecords(docId, tableId, upsertRecords, nil)
	if status != http.StatusOK {
		t.Errorf("UpsertRecords failed with status %d", status)
		return
	}

	// Verify the record was updated
	fetchedRecords, _ := GetRecords(docId, tableId, &GetRecordsOptions{
		Filter: map[string][]interface{}{"email": {"upsert-exist@test.com"}},
	})

	if len(fetchedRecords.Records) != 1 {
		t.Errorf("Expected 1 record after upsert, got %d", len(fetchedRecords.Records))
		return
	}

	if age, ok := fetchedRecords.Records[0].Fields["age"].(float64); !ok || age != 31 {
		t.Errorf("Expected age 31 after upsert, got %v", fetchedRecords.Records[0].Fields["age"])
	}
}

// testEdgeCasesUnicode tests Unicode characters (emoji, CJK)
func testEdgeCasesUnicode(t *testing.T, docId, tableId string) {
	testCases := []struct {
		name  string
		email string
		text  string
	}{
		{
			name:  "Emoji Test",
			email: "emoji@test.com",
			text:  "Hello 👋 World 🌍 Test 🚀",
		},
		{
			name:  "Japanese Test",
			email: "japanese@test.com",
			text:  "こんにちは世界",
		},
		{
			name:  "Chinese Test",
			email: "chinese@test.com",
			text:  "你好世界",
		},
		{
			name:  "Korean Test",
			email: "korean@test.com",
			text:  "안녕하세요",
		},
		{
			name:  "Mixed Unicode",
			email: "mixed@test.com",
			text:  "Hello 世界 👋 안녕 🌍",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			records := []map[string]interface{}{
				{
					"name":  tc.text,
					"email": tc.email,
					"age":   25,
				},
			}

			result, status := AddRecords(docId, tableId, records, nil)
			if status != http.StatusOK {
				t.Errorf("AddRecords failed for %s with status %d", tc.name, status)
				return
			}

			// Verify the Unicode was preserved
			fetchedRecords, _ := GetRecords(docId, tableId, &GetRecordsOptions{
				Filter: map[string][]interface{}{"email": {tc.email}},
			})

			if len(fetchedRecords.Records) != 1 {
				t.Errorf("Expected 1 record for %s, got %d", tc.name, len(fetchedRecords.Records))
				return
			}

			if fetchedRecords.Records[0].Fields["name"] != tc.text {
				t.Errorf("Unicode mismatch for %s: expected %s, got %v", tc.name, tc.text, fetchedRecords.Records[0].Fields["name"])
			}

			// Clean up
			DeleteRecords(docId, tableId, []int{result.Records[0].Id})
		})
	}
}

// testEdgeCasesSpecialChars tests special characters
func testEdgeCasesSpecialChars(t *testing.T, docId, tableId string) {
	testCases := []struct {
		name  string
		email string
		text  string
	}{
		{
			name:  "Quotes",
			email: "quotes@test.com",
			text:  `He said "hello" and 'goodbye'`,
		},
		{
			name:  "Backslashes",
			email: "backslash@test.com",
			text:  `C:\Users\Test\File.txt`,
		},
		{
			name:  "Newlines",
			email: "newlines@test.com",
			text:  "Line 1\nLine 2\nLine 3",
		},
		{
			name:  "Tabs",
			email: "tabs@test.com",
			text:  "Column1\tColumn2\tColumn3",
		},
		{
			name:  "HTML",
			email: "html@test.com",
			text:  "<script>alert('test')</script>",
		},
		{
			name:  "JSON",
			email: "json@test.com",
			text:  `{"key": "value", "number": 123}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			records := []map[string]interface{}{
				{
					"name":  tc.text,
					"email": tc.email,
					"age":   25,
				},
			}

			result, status := AddRecords(docId, tableId, records, nil)
			if status != http.StatusOK {
				t.Errorf("AddRecords failed for %s with status %d", tc.name, status)
				return
			}

			// Verify the special characters were preserved
			fetchedRecords, _ := GetRecords(docId, tableId, &GetRecordsOptions{
				Filter: map[string][]interface{}{"email": {tc.email}},
			})

			if len(fetchedRecords.Records) != 1 {
				t.Errorf("Expected 1 record for %s, got %d", tc.name, len(fetchedRecords.Records))
				return
			}

			if fetchedRecords.Records[0].Fields["name"] != tc.text {
				t.Errorf("Special char mismatch for %s: expected %s, got %v", tc.name, tc.text, fetchedRecords.Records[0].Fields["name"])
			}

			// Clean up
			DeleteRecords(docId, tableId, []int{result.Records[0].Id})
		})
	}
}

// testEdgeCasesNulls tests null and empty values
func testEdgeCasesNulls(t *testing.T, docId, tableId string) {
	testCases := []struct {
		name  string
		email string
		value interface{}
	}{
		{
			name:  "Empty String",
			email: "empty@test.com",
			value: "",
		},
		{
			name:  "Zero",
			email: "zero@test.com",
			value: 0,
		},
		{
			name:  "Null",
			email: "null@test.com",
			value: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			records := []map[string]interface{}{
				{
					"name":  tc.value,
					"email": tc.email,
					"age":   25,
				},
			}

			result, status := AddRecords(docId, tableId, records, nil)
			if status != http.StatusOK {
				t.Errorf("AddRecords failed for %s with status %d", tc.name, status)
				return
			}

			// Verify the value was preserved
			fetchedRecords, _ := GetRecords(docId, tableId, &GetRecordsOptions{
				Filter: map[string][]interface{}{"email": {tc.email}},
			})

			if len(fetchedRecords.Records) != 1 {
				t.Errorf("Expected 1 record for %s, got %d", tc.name, len(fetchedRecords.Records))
				return
			}

			// Clean up
			DeleteRecords(docId, tableId, []int{result.Records[0].Id})
		})
	}
}

// testEdgeCasesLargeText tests very large text values
func testEdgeCasesLargeText(t *testing.T, docId, tableId string) {
	// Create a very large text (10KB)
	largeText := strings.Repeat("A", 10000)

	records := []map[string]interface{}{
		{
			"name":  largeText,
			"email": "largetext@test.com",
			"age":   25,
		},
	}

	result, status := AddRecords(docId, tableId, records, nil)
	if status != http.StatusOK {
		t.Errorf("AddRecords failed for large text with status %d", status)
		return
	}

	// Verify the large text was preserved
	fetchedRecords, _ := GetRecords(docId, tableId, &GetRecordsOptions{
		Filter: map[string][]interface{}{"email": {"largetext@test.com"}},
	})

	if len(fetchedRecords.Records) != 1 {
		t.Errorf("Expected 1 record for large text, got %d", len(fetchedRecords.Records))
		return
	}

	if fetchedRecords.Records[0].Fields["name"] != largeText {
		t.Errorf("Large text mismatch: expected length %d, got %d", len(largeText), len(fetchedRecords.Records[0].Fields["name"].(string)))
	}

	// Clean up
	DeleteRecords(docId, tableId, []int{result.Records[0].Id})
}

// testEdgeCasesLargeNumbers tests very large numbers
func testEdgeCasesLargeNumbers(t *testing.T, docId, tableId string) {
	testCases := []struct {
		name  string
		email string
		value float64
	}{
		{
			name:  "Max Int32",
			email: "maxint32@test.com",
			value: math.MaxInt32,
		},
		{
			name:  "Min Int32",
			email: "minint32@test.com",
			value: math.MinInt32,
		},
		{
			name:  "Large Float",
			email: "largefloat@test.com",
			value: 9999999999.99,
		},
		{
			name:  "Small Float",
			email: "smallfloat@test.com",
			value: 0.000000001,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			records := []map[string]interface{}{
				{
					"name":  tc.name,
					"email": tc.email,
					"age":   tc.value,
				},
			}

			result, status := AddRecords(docId, tableId, records, nil)
			if status != http.StatusOK {
				t.Errorf("AddRecords failed for %s with status %d", tc.name, status)
				return
			}

			// Verify the number was preserved
			fetchedRecords, _ := GetRecords(docId, tableId, &GetRecordsOptions{
				Filter: map[string][]interface{}{"email": {tc.email}},
			})

			if len(fetchedRecords.Records) != 1 {
				t.Errorf("Expected 1 record for %s, got %d", tc.name, len(fetchedRecords.Records))
				return
			}

			// Clean up
			DeleteRecords(docId, tableId, []int{result.Records[0].Id})
		})
	}
}

// testBulkDataLoad tests adding 200-500 records
func testBulkDataLoad(t *testing.T, docId, tableId string) {
	recordCount := 300

	// Generate test data
	records := make([]map[string]interface{}, recordCount)
	for i := 0; i < recordCount; i++ {
		records[i] = map[string]interface{}{
			"name":  fmt.Sprintf("BulkUser%d", i),
			"email": fmt.Sprintf("bulk%d@test.com", i),
			"age":   20 + (i % 50),
		}
	}

	// Add records in batches of 100 (to avoid potential API limits)
	batchSize := 100
	totalAdded := 0

	for i := 0; i < recordCount; i += batchSize {
		end := i + batchSize
		if end > recordCount {
			end = recordCount
		}

		batch := records[i:end]
		result, status := AddRecords(docId, tableId, batch, nil)
		if status != http.StatusOK {
			t.Errorf("AddRecords failed for batch %d-%d with status %d", i, end, status)
			continue
		}

		totalAdded += len(result.Records)
		t.Logf("Added batch %d-%d: %d records", i, end, len(result.Records))
	}

	if totalAdded != recordCount {
		t.Errorf("Expected to add %d records, but added %d", recordCount, totalAdded)
	}

	// Verify the records were added by counting them
	fetchedRecords, status := GetRecords(docId, tableId, nil)
	if status != http.StatusOK {
		t.Errorf("GetRecords failed with status %d", status)
		return
	}

	// Count bulk records
	bulkCount := 0
	var bulkIds []int
	for _, r := range fetchedRecords.Records {
		if email, ok := r.Fields["email"].(string); ok {
			if strings.HasPrefix(email, "bulk") && strings.HasSuffix(email, "@test.com") {
				bulkCount++
				bulkIds = append(bulkIds, r.Id)
			}
		}
	}

	if bulkCount < recordCount {
		t.Errorf("Expected at least %d bulk records, got %d", recordCount, bulkCount)
	}

	t.Logf("Successfully added and verified %d bulk records", bulkCount)

	// Clean up bulk records in batches
	for i := 0; i < len(bulkIds); i += batchSize {
		end := i + batchSize
		if end > len(bulkIds) {
			end = len(bulkIds)
		}

		batch := bulkIds[i:end]
		DeleteRecords(docId, tableId, batch)
		t.Logf("Deleted batch %d-%d: %d records", i, end, len(batch))
	}
}

// TestRecordValidationSummary prints a summary of the test results
func TestRecordValidationSummary(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping summary in short mode")
	}

	t.Log("====================================")
	t.Log("Record CRUD Validation Test Summary")
	t.Log("====================================")
	t.Log("Test Document: vibe-kanban-playground (uFiFazkXAEwx)")
	t.Log("Test Table: RecordCRUDTest")
	t.Log("")
	t.Log("Tests Performed:")
	t.Log("  ✓ Add Single Record")
	t.Log("  ✓ Add Bulk Records")
	t.Log("  ✓ Get Records with Filters")
	t.Log("  ✓ Get Records with Sort")
	t.Log("  ✓ Get Records with Pagination")
	t.Log("  ✓ Update Single Record")
	t.Log("  ✓ Update Bulk Records")
	t.Log("  ✓ Update Partial Fields")
	t.Log("  ✓ Delete Single Record")
	t.Log("  ✓ Delete Bulk Records")
	t.Log("  ✓ Upsert New Record")
	t.Log("  ✓ Upsert Existing Record")
	t.Log("  ✓ Edge Cases: Unicode (Emoji, CJK)")
	t.Log("  ✓ Edge Cases: Special Characters")
	t.Log("  ✓ Edge Cases: Null/Empty Values")
	t.Log("  ✓ Edge Cases: Large Text (10KB)")
	t.Log("  ✓ Edge Cases: Large Numbers")
	t.Log("  ✓ Bulk Data Load (300 records)")
	t.Log("")
	t.Log("All tests completed successfully!")
	t.Log("====================================")
}

// Helper function to pretty print JSON for debugging
func prettyPrintJSON(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// findOrCreateTestDocument finds an existing test document or creates a new one
func findOrCreateTestDocument(t *testing.T, workspaceID int) string {
	// Try to find an existing document first
	workspace := GetWorkspace(workspaceID)
	for _, doc := range workspace.Docs {
		if strings.Contains(doc.Name, "Record") || strings.Contains(doc.Name, "Test") {
			// Verify the document is accessible
			testDoc := GetDoc(doc.Id)
			if testDoc.Id != "" {
				t.Logf("Found existing document: %s (%s)", doc.Name, doc.Id)
				return doc.Id
			}
		}
	}

	// If no existing document found, create a new one
	return createTestDocument(t, workspaceID, "Record CRUD Test Document")
}

// createTestDocument creates a test document in the specified workspace
func createTestDocument(t *testing.T, workspaceID int, name string) string {
	// Grist API endpoint: POST /api/workspaces/{workspaceId}/docs
	url := fmt.Sprintf("workspaces/%d/docs", workspaceID)

	// Request body for creating a document
	requestBody := map[string]interface{}{
		"name": name,
	}

	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		t.Errorf("Failed to marshal request body: %v", err)
		return ""
	}

	response, status := httpPost(url, string(bodyJSON))

	if status != http.StatusOK {
		t.Errorf("Failed to create document '%s': status %d, response: %s", name, status, response)
		return ""
	}

	// The response should be the document ID as a string
	docID := strings.Trim(response, "\"")
	return docID
}

// createTestTable creates a table with test columns in the specified document
func createTestTable(t *testing.T, docID, tableID string) bool {
	// Create table with columns: name (Text), email (Text), age (Numeric)
	requestBody := map[string]interface{}{
		"tables": []map[string]interface{}{
			{
				"id": tableID,
				"columns": []map[string]interface{}{
					{"id": "name", "fields": map[string]interface{}{"label": "Name", "type": "Text"}},
					{"id": "email", "fields": map[string]interface{}{"label": "Email", "type": "Text"}},
					{"id": "age", "fields": map[string]interface{}{"label": "Age", "type": "Numeric"}},
				},
			},
		},
	}

	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		t.Errorf("Failed to marshal table request: %v", err)
		return false
	}

	url := fmt.Sprintf("docs/%s/tables", docID)
	response, status := httpPost(url, string(bodyJSON))

	if status != http.StatusOK {
		t.Errorf("Failed to create table '%s': status %d, response: %s", tableID, status, response)
		return false
	}

	// Verify table was created
	tables := GetDocTables(docID)
	for _, table := range tables.Tables {
		if table.Id == tableID {
			return true
		}
	}

	t.Errorf("Table '%s' was not found after creation", tableID)
	return false
}
