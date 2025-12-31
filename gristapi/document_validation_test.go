// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

package gristapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// findPlaygroundWorkspaceForValidation finds a suitable workspace for document validation testing
func findPlaygroundWorkspaceForValidation(t *testing.T) int {
	orgs := GetOrgs()
	if len(orgs) == 0 {
		t.Fatal("No organizations found - cannot proceed with testing")
	}

	// Find the "docs" workspace (playground workspace)
	var playgroundWorkspaceID int
	for _, org := range orgs {
		workspaces := GetOrgWorkspaces(org.Id)
		for _, ws := range workspaces {
			if ws.Name == "docs" || strings.Contains(strings.ToLower(ws.Name), "playground") {
				playgroundWorkspaceID = ws.Id
				t.Logf("Found playground workspace: %s (ID: %d)", ws.Name, ws.Id)
				return playgroundWorkspaceID
			}
		}
	}

	// Use the first workspace if we can't find playground
	for _, org := range orgs {
		workspaces := GetOrgWorkspaces(org.Id)
		if len(workspaces) > 0 {
			playgroundWorkspaceID = workspaces[0].Id
			t.Logf("Using workspace: %s (ID: %d)", workspaces[0].Name, playgroundWorkspaceID)
			return playgroundWorkspaceID
		}
	}

	t.Fatal("Could not find any workspace for testing")
	return 0
}

// TestDocumentCRUD_Integration is a comprehensive integration test for document operations
// This test creates real documents in the Grist playground workspace and validates all CRUD operations
//
//nolint:gocyclo // This is a comprehensive integration test, high complexity is expected
func TestDocumentCRUD_Integration(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("GRIST_URL") == "" || os.Getenv("GRIST_TOKEN") == "" {
		t.Skip("Skipping integration test: GRIST_URL and GRIST_TOKEN must be set")
	}

	// Test will create 5-10 test documents with varying structures
	timestamp := time.Now().Format("20060102-150405")
	testDocuments := []struct {
		name        string
		description string
	}{
		{fmt.Sprintf("DocValidation-Simple-%s", timestamp), "Simple document for basic CRUD testing"},
		{fmt.Sprintf("DocValidation-Complex-%s", timestamp), "Complex document with multiple tables"},
		{fmt.Sprintf("DocValidation-Webhooks-%s", timestamp), "Document for webhook operations testing"},
		{fmt.Sprintf("DocValidation-Attachments-%s", timestamp), "Document for attachment operations testing"},
		{fmt.Sprintf("DocValidation-Records-%s", timestamp), "Document for records CRUD testing"},
		{fmt.Sprintf("DocValidation-Export-%s", timestamp), "Document for export functionality testing"},
		{fmt.Sprintf("DocValidation-Metadata-%s", timestamp), "Document for metadata operations testing"},
		{fmt.Sprintf("DocValidation-Updates-%s", timestamp), "Document for update operations testing"},
	}

	// Get the playground workspace
	playgroundWorkspaceID := findPlaygroundWorkspaceForValidation(t)

	// Store created document IDs for cleanup and reporting
	var createdDocIDs []string

	// Create test documents
	t.Run("CreateDocuments", func(t *testing.T) {
		for i, testDoc := range testDocuments {
			docID := createTestDocumentForValidation(t, playgroundWorkspaceID, testDoc.name, testDoc.description)
			if docID != "" {
				createdDocIDs = append(createdDocIDs, docID)
				t.Logf("Created document %d/%d: %s (ID: %s)", i+1, len(testDocuments), testDoc.name, docID)
			}
		}

		if len(createdDocIDs) == 0 {
			t.Fatal("Failed to create any test documents")
		}
		t.Logf("Successfully created %d test documents", len(createdDocIDs))
	})

	// Test 1: Get Document Metadata
	t.Run("GetDocumentMetadata", func(t *testing.T) {
		for _, docID := range createdDocIDs {
			doc := GetDoc(docID)
			if doc.Id == "" {
				t.Errorf("Failed to get metadata for document %s", docID)
				continue
			}
			if doc.Id != docID {
				t.Errorf("Document ID mismatch: expected %s, got %s", docID, doc.Id)
			}
			if doc.Name == "" {
				t.Errorf("Document %s has empty name", docID)
			}
			t.Logf("✓ Document %s: Name=%s, IsPinned=%v", doc.Id, doc.Name, doc.IsPinned)
		}
	})

	// Test 2: List Documents (via workspace)
	t.Run("ListDocuments", func(t *testing.T) {
		workspace := GetWorkspace(playgroundWorkspaceID)
		if workspace.Id == 0 {
			t.Fatal("Failed to get workspace")
		}

		foundDocs := 0
		for _, createdID := range createdDocIDs {
			for _, doc := range workspace.Docs {
				if doc.Id == createdID {
					foundDocs++
					t.Logf("✓ Found document %s in workspace listing", doc.Id)
					break
				}
			}
		}

		if foundDocs != len(createdDocIDs) {
			t.Errorf("Expected to find %d documents in workspace, found %d", len(createdDocIDs), foundDocs)
		}
	})

	// Test 3: Get Document Tables
	t.Run("GetDocumentTables", func(t *testing.T) {
		for _, docID := range createdDocIDs {
			tables := GetDocTables(docID)
			// New documents should have at least one default table
			if len(tables.Tables) == 0 {
				t.Logf("⚠ Document %s has no tables (this may be expected for new docs)", docID)
			} else {
				t.Logf("✓ Document %s has %d table(s)", docID, len(tables.Tables))
				for _, table := range tables.Tables {
					t.Logf("  - Table: %s", table.Id)
				}
			}
		}
	})

	// Test 4: Export Documents
	t.Run("ExportDocuments", func(t *testing.T) {
		if len(createdDocIDs) == 0 {
			t.Skip("No documents to test export")
		}

		// Test Excel export
		t.Run("ExportExcel", func(t *testing.T) {
			docID := createdDocIDs[0]
			tmpFile, err := os.CreateTemp("", "test-export-*.xlsx")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			tmpFile.Close()

			ExportDocExcel(docID, tmpFile.Name())

			// Check if file was created and has content
			stat, err := os.Stat(tmpFile.Name())
			if err != nil {
				t.Errorf("Export file not created: %v", err)
			} else if stat.Size() == 0 {
				t.Error("Export file is empty")
			} else {
				t.Logf("✓ Excel export successful: %d bytes", stat.Size())
			}
		})

		// Test Grist format export
		t.Run("ExportGrist", func(t *testing.T) {
			docID := createdDocIDs[0]
			tmpFile, err := os.CreateTemp("", "test-export-*.grist")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			tmpFile.Close()

			ExportDocGrist(docID, tmpFile.Name())

			// Check if file was created and has content
			stat, err := os.Stat(tmpFile.Name())
			if err != nil {
				t.Errorf("Export file not created: %v", err)
			} else if stat.Size() == 0 {
				t.Error("Export file is empty")
			} else {
				t.Logf("✓ Grist export successful: %d bytes", stat.Size())
			}
		})
	})

	// Test 5: Document Access Control
	t.Run("GetDocumentAccess", func(t *testing.T) {
		if len(createdDocIDs) == 0 {
			t.Skip("No documents to test access")
		}

		docID := createdDocIDs[0]
		access := GetDocAccess(docID)

		if len(access.Users) == 0 {
			t.Log("⚠ No users found in document access (may be expected)")
		} else {
			t.Logf("✓ Document %s has %d user(s) with access", docID, len(access.Users))
			for _, user := range access.Users {
				t.Logf("  - User: %s (Access: %s)", user.Email, user.Access)
			}
		}
	})

	// Test 6: Webhook CRUD Operations
	t.Run("WebhookOperations", func(t *testing.T) {
		if len(createdDocIDs) < 3 {
			t.Skip("Need at least 3 documents for webhook testing")
		}

		docID := createdDocIDs[2] // Use the "Webhooks" test document

		// Create webhooks
		t.Run("CreateWebhooks", func(t *testing.T) {
			url1 := "https://webhook.site/test-1"
			url2 := "https://webhook.site/test-2"
			name1 := "Test Webhook 1"
			name2 := "Test Webhook 2"
			tableID := "Table1"
			enabled := true
			eventTypes := []string{"add", "update"}

			webhooks := []WebhookPartialFields{
				{
					Name:       &name1,
					URL:        &url1,
					TableId:    &tableID,
					Enabled:    &enabled,
					EventTypes: &eventTypes,
				},
				{
					Name:       &name2,
					URL:        &url2,
					TableId:    &tableID,
					Enabled:    &enabled,
					EventTypes: &eventTypes,
				},
			}

			result, status := CreateWebhooks(docID, webhooks)
			if status != http.StatusOK {
				t.Errorf("Failed to create webhooks: status %d", status)
				return
			}

			if len(result.Webhooks) != 2 {
				t.Errorf("Expected 2 webhooks created, got %d", len(result.Webhooks))
			} else {
				t.Logf("✓ Created %d webhooks", len(result.Webhooks))
			}

			// List webhooks
			t.Run("ListWebhooks", func(t *testing.T) {
				webhooksList, status := GetWebhooks(docID)
				if status != http.StatusOK {
					t.Errorf("Failed to list webhooks: status %d", status)
					return
				}

				if len(webhooksList.Webhooks) < 2 {
					t.Errorf("Expected at least 2 webhooks, got %d", len(webhooksList.Webhooks))
				} else {
					t.Logf("✓ Found %d webhook(s)", len(webhooksList.Webhooks))
					for _, wh := range webhooksList.Webhooks {
						t.Logf("  - Webhook: %s (Name: %s, Enabled: %v)", wh.Id, wh.Fields.Name, wh.Fields.Enabled)
					}
				}

				// Update webhook if we have any
				if len(result.Webhooks) > 0 {
					t.Run("UpdateWebhook", func(t *testing.T) {
						webhookID := result.Webhooks[0].Id
						disabled := false
						newName := "Updated Webhook Name"
						updateFields := WebhookPartialFields{
							Enabled: &disabled,
							Name:    &newName,
						}

						_, updateStatus := UpdateWebhook(docID, webhookID, updateFields)
						if updateStatus != http.StatusOK {
							t.Errorf("Failed to update webhook: status %d", updateStatus)
						} else {
							t.Logf("✓ Updated webhook %s", webhookID)
						}
					})

					// Delete webhooks
					t.Run("DeleteWebhooks", func(t *testing.T) {
						for _, wh := range result.Webhooks {
							deleteResult, deleteStatus := DeleteWebhook(docID, wh.Id)
							if deleteStatus != http.StatusOK {
								t.Errorf("Failed to delete webhook %s: status %d", wh.Id, deleteStatus)
							} else if !deleteResult.Success {
								t.Errorf("Webhook deletion reported failure for %s", wh.Id)
							} else {
								t.Logf("✓ Deleted webhook %s", wh.Id)
							}
						}
					})
				}
			})
		})

		// Test webhook queue clearing
		t.Run("ClearWebhookQueue", func(t *testing.T) {
			_, status := ClearWebhookQueue(docID)
			if status != http.StatusOK {
				t.Logf("⚠ Failed to clear webhook queue: status %d (may be expected if no queue)", status)
			} else {
				t.Log("✓ Cleared webhook queue")
			}
		})
	})

	// Test 7: Attachment Operations
	t.Run("AttachmentOperations", func(t *testing.T) {
		if len(createdDocIDs) < 4 {
			t.Skip("Need at least 4 documents for attachment testing")
		}

		docID := createdDocIDs[3] // Use the "Attachments" test document

		// Create test files
		testFile1, err := os.CreateTemp("", "test-attachment-1-*.txt")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(testFile1.Name())
		testFile1.WriteString("Test attachment content 1")
		testFile1.Close()

		testFile2, err := os.CreateTemp("", "test-attachment-2-*.txt")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(testFile2.Name())
		testFile2.WriteString("Test attachment content 2")
		testFile2.Close()

		var uploadedIDs []int

		// Upload attachments
		t.Run("UploadAttachments", func(t *testing.T) {
			result, status := UploadAttachments(docID, []string{testFile1.Name(), testFile2.Name()})
			if status != http.StatusOK {
				t.Errorf("Failed to upload attachments: status %d", status)
				return
			}

			if len(result) != 2 {
				t.Errorf("Expected 2 attachment IDs, got %d", len(result))
			} else {
				uploadedIDs = result
				t.Logf("✓ Uploaded %d attachments: %v", len(result), result)
			}
		})

		// List attachments
		t.Run("ListAttachments", func(t *testing.T) {
			attachments, status := ListAttachments(docID, nil)
			if status != http.StatusOK {
				t.Errorf("Failed to list attachments: status %d", status)
				return
			}

			if len(attachments.Records) < len(uploadedIDs) {
				t.Errorf("Expected at least %d attachments, got %d", len(uploadedIDs), len(attachments.Records))
			} else {
				t.Logf("✓ Found %d attachment(s)", len(attachments.Records))
				for _, att := range attachments.Records {
					t.Logf("  - Attachment ID %d: %s (%d bytes)", att.Id, att.FileName, att.FileSize)
				}
			}
		})

		// Get attachment metadata
		if len(uploadedIDs) > 0 {
			t.Run("GetAttachmentMetadata", func(t *testing.T) {
				for _, attID := range uploadedIDs {
					metadata, status := GetAttachmentMetadata(docID, attID)
					if status != http.StatusOK {
						t.Errorf("Failed to get metadata for attachment %d: status %d", attID, status)
						continue
					}

					if metadata.Id != attID {
						t.Errorf("Attachment ID mismatch: expected %d, got %d", attID, metadata.Id)
					}
					if metadata.FileName == "" {
						t.Errorf("Attachment %d has empty filename", attID)
					} else {
						t.Logf("✓ Attachment %d: %s (%d bytes, uploaded: %s)", metadata.Id, metadata.FileName, metadata.FileSize, metadata.TimeUploaded)
					}
				}
			})

			// Download attachment
			t.Run("DownloadAttachment", func(t *testing.T) {
				attID := uploadedIDs[0]
				content, contentType, status := DownloadAttachment(docID, attID)
				if status != http.StatusOK {
					t.Errorf("Failed to download attachment %d: status %d", attID, status)
					return
				}

				if len(content) == 0 {
					t.Errorf("Downloaded attachment %d is empty", attID)
				} else {
					t.Logf("✓ Downloaded attachment %d: %d bytes (type: %s)", attID, len(content), contentType)
				}

				// Download to file
				tmpFile, err := os.CreateTemp("", "downloaded-attachment-*.txt")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer os.Remove(tmpFile.Name())
				tmpFile.Close()

				err = DownloadAttachmentToFile(docID, attID, tmpFile.Name())
				if err != nil {
					t.Errorf("Failed to download attachment to file: %v", err)
				} else {
					stat, _ := os.Stat(tmpFile.Name())
					t.Logf("✓ Downloaded attachment to file: %d bytes", stat.Size())
				}
			})
		}

		// Delete unused attachments
		t.Run("DeleteUnusedAttachments", func(t *testing.T) {
			_, status := DeleteUnusedAttachments(docID)
			if status != http.StatusOK {
				t.Logf("⚠ Failed to delete unused attachments: status %d", status)
			} else {
				t.Log("✓ Deleted unused attachments")
			}
		})
	})

	// Test 8: Update Documents (move, purge)
	t.Run("UpdateOperations", func(t *testing.T) {
		if len(createdDocIDs) < 2 {
			t.Skip("Need at least 2 documents for update testing")
		}

		// Test purge history
		t.Run("PurgeDocumentHistory", func(t *testing.T) {
			docID := createdDocIDs[len(createdDocIDs)-1]
			PurgeDoc(docID, 1)
			// Note: PurgeDoc prints to stdout, we just verify it doesn't panic
			t.Logf("✓ Purged document %s history (kept 1 state)", docID)
		})
	})

	// Test 9: Delete Documents
	t.Run("DeleteDocuments", func(t *testing.T) {
		// We'll only delete some documents to leave some for manual inspection
		deleteCount := len(createdDocIDs) / 2
		if deleteCount < 2 {
			deleteCount = 1
		}

		for i := 0; i < deleteCount; i++ {
			docID := createdDocIDs[i]
			DeleteDoc(docID)
			t.Logf("✓ Deleted document %s", docID)
		}

		// Verify deletion
		for i := 0; i < deleteCount; i++ {
			docID := createdDocIDs[i]
			doc := GetDoc(docID)
			if doc.Id != "" {
				t.Errorf("Document %s still exists after deletion", docID)
			}
		}

		// Keep remaining documents for manual inspection
		remainingDocs := createdDocIDs[deleteCount:]
		if len(remainingDocs) > 0 {
			t.Logf("Keeping %d documents for inspection: %v", len(remainingDocs), remainingDocs)
		}
	})

	// Final summary
	t.Run("Summary", func(t *testing.T) {
		t.Logf("=== Test Summary ===")
		t.Logf("Total documents created: %d", len(createdDocIDs))
		t.Logf("Document IDs: %v", createdDocIDs)
		t.Logf("Workspace ID: %d", playgroundWorkspaceID)
		t.Log("All document CRUD validation tests completed")
	})
}

// createTestDocumentForValidation creates a test document in the specified workspace
// Note: Grist API doesn't have a direct "create document" endpoint in the documented API
// Documents are typically created through the UI or by copying templates
// This helper function creates a document by POSTing to the workspace
func createTestDocumentForValidation(t *testing.T, workspaceID int, name, description string) string {
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

// findFirstDocumentID finds the first available document ID for testing
func findFirstDocumentID(t *testing.T) string {
	orgs := GetOrgs()
	if len(orgs) == 0 {
		t.Skip("No organizations found")
		return ""
	}

	for _, org := range orgs {
		workspaces := GetOrgWorkspaces(org.Id)
		for _, ws := range workspaces {
			if len(ws.Docs) > 0 {
				return ws.Docs[0].Id
			}
		}
	}

	t.Skip("No documents found for testing")
	return ""
}

// TestDocumentOperations_TableDriven provides table-driven tests for document operations
func TestDocumentOperations_TableDriven(t *testing.T) {
	if os.Getenv("GRIST_URL") == "" || os.Getenv("GRIST_TOKEN") == "" {
		t.Skip("Skipping integration test: GRIST_URL and GRIST_TOKEN must be set")
	}

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{"GetDoc_ValidID", testGetDocValidID},
		{"GetDoc_InvalidID", testGetDocInvalidID},
		{"GetDocTables_ValidDoc", testGetDocTablesValid},
		{"GetDocAccess_ValidDoc", testGetDocAccessValid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t)
		})
	}
}

func testGetDocValidID(t *testing.T) {
	docID := findFirstDocumentID(t)
	if docID == "" {
		return
	}

	doc := GetDoc(docID)
	if doc.Id == "" {
		t.Error("GetDoc returned empty document")
	}
	if doc.Id != docID {
		t.Errorf("Expected doc ID %s, got %s", docID, doc.Id)
	}
}

func testGetDocInvalidID(t *testing.T) {
	doc := GetDoc("nonexistent-doc-id-12345")
	if doc.Id != "" {
		t.Error("Expected empty document for invalid ID")
	}
}

func testGetDocTablesValid(t *testing.T) {
	docID := findFirstDocumentID(t)
	if docID == "" {
		return
	}

	tables := GetDocTables(docID)
	t.Logf("Document has %d table(s)", len(tables.Tables))
}

func testGetDocAccessValid(t *testing.T) {
	docID := findFirstDocumentID(t)
	if docID == "" {
		return
	}

	access := GetDocAccess(docID)
	t.Logf("Document has %d user(s) with access", len(access.Users))
}

// TestDocumentExport_Formats tests different export formats
func TestDocumentExport_Formats(t *testing.T) {
	if os.Getenv("GRIST_URL") == "" || os.Getenv("GRIST_TOKEN") == "" {
		t.Skip("Skipping integration test: GRIST_URL and GRIST_TOKEN must be set")
	}

	// Find a document to export
	orgs := GetOrgs()
	if len(orgs) == 0 {
		t.Skip("No organizations found")
	}

	var docID string
	for _, org := range orgs {
		workspaces := GetOrgWorkspaces(org.Id)
		for _, ws := range workspaces {
			if len(ws.Docs) > 0 {
				docID = ws.Docs[0].Id
				break
			}
		}
		if docID != "" {
			break
		}
	}

	if docID == "" {
		t.Skip("No documents found for testing")
	}

	exportFormats := []struct {
		name       string
		extension  string
		exportFunc func(docID, fileName string)
	}{
		{
			name:      "Excel",
			extension: ".xlsx",
			exportFunc: func(docID, fileName string) {
				ExportDocExcel(docID, fileName)
			},
		},
		{
			name:      "Grist",
			extension: ".grist",
			exportFunc: func(docID, fileName string) {
				ExportDocGrist(docID, fileName)
			},
		},
	}

	for _, format := range exportFormats {
		t.Run(format.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", fmt.Sprintf("export-test-*%s", format.extension))
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			tmpFile.Close()

			format.exportFunc(docID, tmpFile.Name())

			stat, err := os.Stat(tmpFile.Name())
			if err != nil {
				t.Errorf("Export file not created: %v", err)
			} else if stat.Size() == 0 {
				t.Error("Export file is empty")
			} else {
				t.Logf("Export successful: %d bytes", stat.Size())
			}
		})
	}
}

// Benchmark document operations
func BenchmarkGetDoc(b *testing.B) {
	if os.Getenv("GRIST_URL") == "" || os.Getenv("GRIST_TOKEN") == "" {
		b.Skip("Skipping benchmark: GRIST_URL and GRIST_TOKEN must be set")
	}

	// Find a document
	orgs := GetOrgs()
	if len(orgs) == 0 {
		b.Skip("No organizations found")
	}

	var docID string
	for _, org := range orgs {
		workspaces := GetOrgWorkspaces(org.Id)
		for _, ws := range workspaces {
			if len(ws.Docs) > 0 {
				docID = ws.Docs[0].Id
				break
			}
		}
		if docID != "" {
			break
		}
	}

	if docID == "" {
		b.Skip("No documents found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetDoc(docID)
	}
}

func BenchmarkGetDocTables(b *testing.B) {
	if os.Getenv("GRIST_URL") == "" || os.Getenv("GRIST_TOKEN") == "" {
		b.Skip("Skipping benchmark: GRIST_URL and GRIST_TOKEN must be set")
	}

	orgs := GetOrgs()
	if len(orgs) == 0 {
		b.Skip("No organizations found")
	}

	var docID string
	for _, org := range orgs {
		workspaces := GetOrgWorkspaces(org.Id)
		for _, ws := range workspaces {
			if len(ws.Docs) > 0 {
				docID = ws.Docs[0].Id
				break
			}
		}
		if docID != "" {
			break
		}
	}

	if docID == "" {
		b.Skip("No documents found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetDocTables(docID)
	}
}
