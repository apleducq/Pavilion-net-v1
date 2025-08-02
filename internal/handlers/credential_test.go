package handlers

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/services"
)

func TestCredentialHandler(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Issuer: "https://test-issuer.com",
	}

	// Generate test keys
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key: %v", err)
	}

	// Create signing service
	signingService := services.NewCredentialSigningService(rsaKey, ecdsaKey, "test-key-1", "test-issuer")

	// Create credential handler
	handler := NewCredentialHandler(cfg, signingService)

	t.Run("CreateCredential", func(t *testing.T) {
		// Create test request
		reqBody := CreateCredentialRequest{
			Type:          "StudentCredential",
			Subject:       "student-123",
			Claims:        map[string]interface{}{"program": "Computer Science", "status": "enrolled"},
			SigningMethod: "JWT",
			Metadata:      map[string]interface{}{"source": "university"},
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/credentials", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.HandleCreateCredential(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		var response CreateCredentialResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "success" {
			t.Errorf("Expected status success, got %s", response.Status)
		}

		if response.Credential == nil {
			t.Error("Expected credential in response")
		}

		if response.Signature == nil {
			t.Error("Expected signature in response")
		}

		// Store credential ID for later tests
		credentialID := response.Credential.ID
		t.Logf("Created credential with ID: %s", credentialID)

		t.Run("GetCredential", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/credentials/"+credentialID, nil)
			w := httptest.NewRecorder()

			// Create router for URL parameters
			router := mux.NewRouter()
			router.HandleFunc("/credentials/{id}", handler.HandleGetCredential).Methods("GET")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response GetCredentialResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Status != "success" {
				t.Errorf("Expected status success, got %s", response.Status)
			}

			if response.Credential.ID != credentialID {
				t.Errorf("Expected credential ID %s, got %s", credentialID, response.Credential.ID)
			}
		})

		t.Run("GetCredentialStatus", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/credentials/"+credentialID+"/status", nil)
			w := httptest.NewRecorder()

			router := mux.NewRouter()
			router.HandleFunc("/credentials/{id}/status", handler.HandleGetCredentialStatus).Methods("GET")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response CredentialStatusResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Status != "valid" {
				t.Errorf("Expected status valid, got %s", response.Status)
			}

			if response.CredentialID != credentialID {
				t.Errorf("Expected credential ID %s, got %s", credentialID, response.CredentialID)
			}
		})

		t.Run("VerifyCredential", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/credentials/"+credentialID+"/verify", nil)
			w := httptest.NewRecorder()

			router := mux.NewRouter()
			router.HandleFunc("/credentials/{id}/verify", handler.HandleVerifyCredential).Methods("POST")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["valid"] != true {
				t.Errorf("Expected valid true, got %v", response["valid"])
			}
		})

		t.Run("RevokeCredential", func(t *testing.T) {
			reqBody := RevokeCredentialRequest{
				Reason: "Student graduated",
			}

			reqBodyBytes, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/credentials/"+credentialID+"/revoke", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := mux.NewRouter()
			router.HandleFunc("/credentials/{id}/revoke", handler.HandleRevokeCredential).Methods("POST")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response RevokeCredentialResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Status != "success" {
				t.Errorf("Expected status success, got %s", response.Status)
			}

			if response.CredentialID != credentialID {
				t.Errorf("Expected credential ID %s, got %s", credentialID, response.CredentialID)
			}

			if response.Reason != "Student graduated" {
				t.Errorf("Expected reason 'Student graduated', got %s", response.Reason)
			}

			// Verify credential is now revoked
			req = httptest.NewRequest("GET", "/credentials/"+credentialID+"/status", nil)
			w = httptest.NewRecorder()

			router = mux.NewRouter()
			router.HandleFunc("/credentials/{id}/status", handler.HandleGetCredentialStatus).Methods("GET")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var statusResponse CredentialStatusResponse
			if err := json.NewDecoder(w.Body).Decode(&statusResponse); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if statusResponse.Status != "revoked" {
				t.Errorf("Expected status revoked, got %s", statusResponse.Status)
			}
		})
	})

	t.Run("ListCredentials", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/credentials", nil)
		w := httptest.NewRecorder()

		handler.HandleListCredentials(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["status"] != "success" {
			t.Errorf("Expected status success, got %s", response["status"])
		}

		// Should have at least one credential from previous test
		count := response["count"].(float64)
		if count < 1 {
			t.Errorf("Expected at least 1 credential, got %v", count)
		}
	})

	t.Run("GetNonExistentCredential", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/credentials/non-existent", nil)
		w := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/credentials/{id}", handler.HandleGetCredential).Methods("GET")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("CreateCredentialWithInvalidRequest", func(t *testing.T) {
		// Missing required fields
		reqBody := CreateCredentialRequest{
			Type: "StudentCredential",
			// Missing Subject and SigningMethod
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/credentials", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.HandleCreateCredential(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("CreateCredentialWithDifferentSigningMethods", func(t *testing.T) {
		signingMethods := []string{"JWT", "LD-PROOF", "ECDSA", "RSA"}

		for _, method := range signingMethods {
			t.Run("SigningMethod_"+method, func(t *testing.T) {
				reqBody := CreateCredentialRequest{
					Type:          "TestCredential",
					Subject:       "test-subject",
					Claims:        map[string]interface{}{"test": "value"},
					SigningMethod: method,
				}

				reqBodyBytes, _ := json.Marshal(reqBody)
				req := httptest.NewRequest("POST", "/credentials", bytes.NewBuffer(reqBodyBytes))
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				handler.HandleCreateCredential(w, req)

				if w.Code != http.StatusCreated {
					t.Errorf("Expected status 201 for method %s, got %d", method, w.Code)
				}

				var response CreateCredentialResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response for method %s: %v", method, err)
				}

				if response.Signature.Method != services.SigningMethod(method) {
					t.Errorf("Expected signing method %s, got %s", method, response.Signature.Method)
				}
			})
		}
	})
} 