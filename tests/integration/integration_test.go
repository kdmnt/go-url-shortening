//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"go-url-shortening/config"
	"go-url-shortening/handlers"
	"go-url-shortening/services"
	"go-url-shortening/storage"
	"go-url-shortening/types"
)

func sendRequest(t *testing.T, server *httptest.Server, method, path string, body interface{}) (*http.Response, []byte) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err, "Failed to marshal request body")
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, server.URL+path, reqBody)
	require.NoError(t, err, "Failed to create request")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "Failed to send request")

	respBody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")
	resp.Body.Close()

	return resp, respBody
}

func setupTestEnvironment(t *testing.T, storageCapacity ...int) (*httptest.Server, func(), *logrus.Logger, *gin.Engine, *config.Config) {
	cfg := config.DefaultConfig()
	capacity := 1000000
	if len(storageCapacity) > 0 {
		capacity = storageCapacity[0]
	}
	store := storage.NewInMemoryStorage(capacity, logrus.New())
	urlService := services.NewURLService(store)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	logger := logrus.New()
	limiter := rate.NewLimiter(rate.Every(time.Second/time.Duration(cfg.RateLimit)), cfg.RateLimit)
	urlHandler, err := handlers.NewURLHandler(ctx, urlService, cfg, logger, limiter)
	require.NoError(t, err, "Failed to create URLHandler")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(handlers.CORSMiddleware())
	handlers.RegisterRoutes(router, urlHandler, cfg)

	server := httptest.NewServer(router)

	return server, func() {
		server.Close()
		cancel()
	}, logger, router, cfg
}

func TestIntegration(t *testing.T) {
	server, cleanup, logger, router, cfg := setupTestEnvironment(t, 1000000)
	defer cleanup()

	t.Run("BasicOperations", func(t *testing.T) {
		var shortURL string

		t.Run("CreateShortURL", func(t *testing.T) {
			urlReq := types.URLRequest{URL: "https://example.com"}
			resp, body := sendRequest(t, server, "POST", "/api/v1/short", urlReq)
			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var response types.URLResponse
			err := json.Unmarshal(body, &response)
			require.NoError(t, err, "Failed to unmarshal response")
			assert.NotEmpty(t, response.ShortURL, "Handler failed to return a short URL")
			shortURL = response.ShortURL
		})

		t.Run("GetOriginalURL", func(t *testing.T) {
			resp, body := sendRequest(t, server, "GET", "/api/v1/short/"+shortURL, nil)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var response types.URLResponse
			err := json.Unmarshal(body, &response)
			require.NoError(t, err, "Failed to unmarshal response")
			assert.Equal(t, "https://example.com", response.OriginalURL)
		})

		t.Run("UpdateURL", func(t *testing.T) {
			urlReq := types.URLRequest{URL: "https://updated-example.com"}
			resp, body := sendRequest(t, server, "PUT", "/api/v1/short/"+shortURL, urlReq)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var response types.URLResponse
			err := json.Unmarshal(body, &response)
			require.NoError(t, err, "Failed to unmarshal response")
			assert.Equal(t, "https://updated-example.com", response.OriginalURL)
		})

		t.Run("DeleteURL", func(t *testing.T) {
			resp, _ := sendRequest(t, server, "DELETE", "/api/v1/short/"+shortURL, nil)
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})
	})

	t.Run("HealthCheck", func(t *testing.T) {
		resp, body := sendRequest(t, server, "GET", "/health", nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "OK", string(body))
	})

	t.Run("Rate Limiting", func(t *testing.T) {
		t.Parallel()
		testServer, cleanup, _, _, testCfg := setupTestEnvironment(t)
		defer cleanup()

		client := &http.Client{}

		testIP := func(ip string) {
			// Make testCfg.RateLimit requests, all should succeed
			for i := 0; i < testCfg.RateLimit; i++ {
				req, _ := http.NewRequest("GET", testServer.URL+"/health", nil)
				req.Header.Set("X-Forwarded-For", ip)
				resp, err := client.Do(req)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				resp.Body.Close()
			}

			// The next request should be rate limited
			req, _ := http.NewRequest("GET", testServer.URL+"/health", nil)
			req.Header.Set("X-Forwarded-For", ip)
			resp, err := client.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
			resp.Body.Close()

			// Wait for the rate limit period to pass
			time.Sleep(time.Second)

			// Now we should be able to make a request again
			req, _ = http.NewRequest("GET", testServer.URL+"/health", nil)
			req.Header.Set("X-Forwarded-For", ip)
			resp, err = client.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		}

		// Test rate limiting for multiple IPs
		testIP("192.0.2.1")
		testIP("192.0.2.2")
		testIP("192.0.2.3")
	})

	t.Run("CORS Headers", func(t *testing.T) {
		t.Parallel()
		corsServer := httptest.NewServer(router)
		defer corsServer.Close()

		req, _ := http.NewRequest("OPTIONS", corsServer.URL+"/api/v1/short", nil)
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "POST, GET, OPTIONS, PUT, DELETE", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization", resp.Header.Get("Access-Control-Allow-Headers"))
	})

	t.Run("Error Handling", func(t *testing.T) {
		t.Parallel()
		testServer, cleanup, _, _, _ := setupTestEnvironment(t)
		defer cleanup()

		// Test invalid URL
		invalidURLReq := types.URLRequest{URL: "not-a-valid-url"}
		jsonBody, _ := json.Marshal(invalidURLReq)
		req, _ := http.NewRequest("POST", testServer.URL+"/api/v1/short", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// Test non-existent short URL
		req, _ = http.NewRequest("GET", testServer.URL+"/api/v1/short/nonexistent", nil)
		resp, err = http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Concurrent Access", func(t *testing.T) {
		t.Parallel()
		testServer := httptest.NewServer(router)
		defer testServer.Close()

		const numRequests = 50
		results := make(chan string, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				urlReq := types.URLRequest{URL: fmt.Sprintf("https://example.com/concurrent%d", i)}
				jsonBody, _ := json.Marshal(urlReq)
				req, _ := http.NewRequest("POST", testServer.URL+"/api/v1/short", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					results <- fmt.Sprintf("Error: %v", err)
					return
				}
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusTooManyRequests {
					results <- "RateLimited"
					return
				}
				var response types.URLResponse
				json.NewDecoder(resp.Body).Decode(&response)
				results <- response.ShortURL
			}()
			time.Sleep(5 * time.Millisecond) // Add delay between requests
		}

		shortURLs := make(map[string]bool)
		rateLimited := 0
		errors := 0

		for i := 0; i < numRequests; i++ {
			result := <-results
			switch {
			case result == "RateLimited":
				rateLimited++
			case strings.HasPrefix(result, "Error:"):
				errors++
				t.Logf("Request error: %s", result)
			default:
				shortURLs[result] = true
			}
		}

		t.Logf("Successful requests: %d, Rate limited: %d, Errors: %d", len(shortURLs), rateLimited, errors)
		assert.True(t, len(shortURLs) > 0, "Expected some successful requests")
		assert.True(t, rateLimited > 0, "Expected some requests to be rate limited")
		assert.Equal(t, 0, errors, "Expected no request errors")
		assert.Equal(t, numRequests, len(shortURLs)+rateLimited+errors, "Total results should match number of requests")
	})

	t.Run("Extensive Modifications", func(t *testing.T) {
		t.Parallel()
		// Create a new storage, service, and handler for this test
		testStore := storage.NewInMemoryStorage(1000000, logrus.New())
		testService := services.NewURLService(testStore)
		testLimiter := rate.NewLimiter(rate.Every(time.Second/time.Duration(cfg.RateLimit)), cfg.RateLimit)
		testHandler, err := handlers.NewURLHandler(context.Background(), testService, cfg, logger, testLimiter)
		assert.NoError(t, err)

		testRouter := gin.New()
		testRouter.Use(handlers.CORSMiddleware())
		handlers.RegisterRoutes(testRouter, testHandler, cfg)

		testServer := httptest.NewServer(testRouter)
		defer testServer.Close()

		// Create initial URL
		initialURL := "https://example.com/initial"
		createReq := types.URLRequest{URL: initialURL}
		jsonBody, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", testServer.URL+"/api/v1/short", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		var createResp types.URLResponse
		json.NewDecoder(resp.Body).Decode(&createResp)
		shortURL := createResp.ShortURL

		// Update URL multiple times
		for i := 1; i <= 5; i++ {
			updateURL := fmt.Sprintf("https://example.com/update%d", i)
			updateReq := types.URLRequest{URL: updateURL}
			jsonBody, _ := json.Marshal(updateReq)
			req, _ = http.NewRequest("PUT", testServer.URL+"/api/v1/short/"+shortURL, bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			resp, err = http.DefaultClient.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			time.Sleep(50 * time.Millisecond) // Add a small delay between updates
		}

		// Verify final update
		time.Sleep(100 * time.Millisecond) // Add a delay before verification
		req, _ = http.NewRequest("GET", testServer.URL+"/api/v1/short/"+shortURL, nil)
		resp, err = http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var getResp types.URLResponse
		json.NewDecoder(resp.Body).Decode(&getResp)
		assert.Equal(t, "https://example.com/update5", getResp.OriginalURL)

		// Delete URL
		req, _ = http.NewRequest("DELETE", testServer.URL+"/api/v1/short/"+shortURL, nil)
		resp, err = http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify deletion
		time.Sleep(100 * time.Millisecond) // Add a delay before verification
		req, _ = http.NewRequest("GET", testServer.URL+"/api/v1/short/"+shortURL, nil)
		resp, err = http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Invalid Input", func(t *testing.T) {
		t.Parallel()
		testServer, cleanup, _, _, _ := setupTestEnvironment(t)
		defer cleanup()

		// Test empty URL
		emptyURLReq := types.URLRequest{URL: ""}
		jsonBody, _ := json.Marshal(emptyURLReq)
		req, _ := http.NewRequest("POST", testServer.URL+"/api/v1/short", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// Test malformed URL
		malformedURLReq := types.URLRequest{URL: "not-a-url"}
		jsonBody, _ = json.Marshal(malformedURLReq)
		req, _ = http.NewRequest("POST", testServer.URL+"/api/v1/short", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err = http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Duplicate URL", func(t *testing.T) {
		urlReq := types.URLRequest{URL: "https://example.com/duplicate"}
		jsonBody, _ := json.Marshal(urlReq)

		// First request
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/short", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		var firstResp types.URLResponse
		json.NewDecoder(resp.Body).Decode(&firstResp)

		// Second request with the same URL
		req, _ = http.NewRequest("POST", server.URL+"/api/v1/short", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err = http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		var secondResp types.URLResponse
		json.NewDecoder(resp.Body).Decode(&secondResp)

		assert.Equal(t, firstResp.ShortURL, secondResp.ShortURL)
	})

	t.Run("Update Non-existent Short URL", func(t *testing.T) {
		t.Parallel()
		testServer, cleanup, _, _, _ := setupTestEnvironment(t)
		defer cleanup()

		// Test updating a non-existent short URL
		updateReq := types.URLRequest{URL: "https://example.com/updated"}
		jsonBody, _ := json.Marshal(updateReq)
		req, _ := http.NewRequest("PUT", testServer.URL+"/api/v1/short/nonexistent", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		// Verify the response body
		var errorResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errorResp)
		assert.Equal(t, "Short URL not found", errorResp["error"])
	})

	t.Run("Delete Non-existent Short URL", func(t *testing.T) {
		t.Parallel()
		testServer, cleanup, _, _, _ := setupTestEnvironment(t)
		defer cleanup()

		// Test deleting a non-existent short URL
		req, _ := http.NewRequest("DELETE", testServer.URL+"/api/v1/short/nonexistent", nil)
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		// Verify the response body
		var errorResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errorResp)
		assert.Equal(t, "Short URL not found", errorResp["error"])
	})

	t.Run("Storage Full", func(t *testing.T) {
		t.Parallel()
		// Create a new test environment with a small capacity
		testServer, cleanup, _, _, _ := setupTestEnvironment(t, 2) // Set capacity to 2
		defer cleanup()
		defer testServer.Close()

		// Create 2 URLs to fill the storage
		for i := 0; i < 2; i++ {
			urlReq := types.URLRequest{URL: fmt.Sprintf("https://example.com/full%d", i)}
			jsonBody, _ := json.Marshal(urlReq)
			req, _ := http.NewRequest("POST", testServer.URL+"/api/v1/short", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusCreated, resp.StatusCode)
		}

		// Try to create one more URL
		urlReq := types.URLRequest{URL: "https://example.com/overflow"}
		jsonBody, _ := json.Marshal(urlReq)
		req, _ := http.NewRequest("POST", testServer.URL+"/api/v1/short", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInsufficientStorage, resp.StatusCode)

		// Verify the response body
		var errorResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errorResp)
		assert.Equal(t, "Storage capacity reached", errorResp["error"])
	})

	t.Run("Redirect Short URL", func(t *testing.T) {
		t.Parallel()
		// Create a new test environment
		testCfg := config.DefaultConfig()
		testStore := storage.NewInMemoryStorage(1000000, logrus.New())
		testService := services.NewURLService(testStore)
		testLimiter := rate.NewLimiter(rate.Every(time.Second/time.Duration(testCfg.RateLimit)), testCfg.RateLimit)
		testLogger := logrus.New()
		testHandler, err := handlers.NewURLHandler(context.Background(), testService, testCfg, testLogger, testLimiter)
		assert.NoError(t, err)

		testRouter := gin.New()
		testRouter.Use(handlers.CORSMiddleware())
		handlers.RegisterRoutes(testRouter, testHandler, cfg)

		testServer := httptest.NewServer(testRouter)
		defer testServer.Close()

		// Create a short URL first
		createReq := types.URLRequest{URL: "https://example.com/redirect"}
		jsonBody, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", testServer.URL+"/api/v1/short", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		var createResp types.URLResponse
		json.NewDecoder(resp.Body).Decode(&createResp)

		// Test redirection
		req, _ = http.NewRequest("GET", testServer.URL+"/"+createResp.ShortURL, nil)
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		resp, err = client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusMovedPermanently, resp.StatusCode)
		assert.Equal(t, "https://example.com/redirect", resp.Header.Get("Location"))

		// Test redirection for non-existent short URL
		req, _ = http.NewRequest("GET", testServer.URL+"/nonexistent", nil)
		resp, err = client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Concurrent Modifications", func(t *testing.T) {
		t.Parallel()
		testServer, cleanup, _, _, _ := setupTestEnvironment(t)
		defer cleanup()

		// Create initial URL
		createReq := types.URLRequest{URL: "https://example.com/concurrent"}
		jsonBody, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", testServer.URL+"/api/v1/short", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		var createResp types.URLResponse
		json.NewDecoder(resp.Body).Decode(&createResp)
		shortURL := createResp.ShortURL

		// Concurrent updates
		var wg sync.WaitGroup
		updateCount := 10
		successCount := int32(0)

		for i := 0; i < updateCount; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				updateReq := types.URLRequest{URL: fmt.Sprintf("https://example.com/concurrent/update%d", i)}
				jsonBody, _ := json.Marshal(updateReq)
				req, _ := http.NewRequest("PUT", testServer.URL+"/api/v1/short/"+shortURL, bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				resp, err := http.DefaultClient.Do(req)
				if err == nil && resp.StatusCode == http.StatusOK {
					atomic.AddInt32(&successCount, 1)
				}
			}(i)
			time.Sleep(50 * time.Millisecond) // Increase delay between requests
		}

		wg.Wait()

		// Verify that at least one update was successful
		assert.True(t, atomic.LoadInt32(&successCount) > 0, "At least one update should succeed")

		// Verify final state
		req, _ = http.NewRequest("GET", testServer.URL+"/api/v1/short/"+shortURL, nil)
		resp, err = http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var getResp types.URLResponse
		json.NewDecoder(resp.Body).Decode(&getResp)
		assert.Contains(t, getResp.OriginalURL, "https://example.com/concurrent/update", "Final URL should be one of the updates")
	})
}
