package coco_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/tobolabs/coco"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResponseAppend(t *testing.T) {
	app := coco.NewApp()

	// Define a route that appends headers to the response
	app.Get("/test-append", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Append("X-Custom-Header", "Value1")
		res.Append("X-Custom-Header", "Value2")
		res.SendStatus(http.StatusOK)
	})

	// Create a test server using the app's handler
	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	// Make a request to the test server
	resp, err := http.Get(srv.URL + "/test-append")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	// Check if the header is set correctly
	values, ok := resp.Header["X-Custom-Header"]
	if !ok {
		t.Fatalf("Header 'X-Custom-Header' not found")
	}
	if len(values) != 2 {
		t.Fatalf("Expected 2 values for 'X-Custom-Header', got %d", len(values))
	}
	if values[0] != "Value1" {
		t.Errorf("Expected first value of 'X-Custom-Header' to be 'Value1', got '%s'", values[0])
	}
	if values[1] != "Value2" {
		t.Errorf("Expected second value of 'X-Custom-Header' to be 'Value2', got '%s'", values[1])
	}
}

func TestResponseAttachment(t *testing.T) {
	app := coco.NewApp()

	app.Get("/test-attachment", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Attachment("test.pdf")
	})

	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/test-attachment")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition != "attachment; filename=test.pdf" {
		t.Errorf("Expected Content-Disposition to be 'attachment; filename=test.pdf', got '%s'", contentDisposition)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/pdf" {
		t.Errorf("Expected Content-Type to be 'application/pdf', got '%s'", contentType)
	}
}

func TestResponseCookie(t *testing.T) {
	app := coco.NewApp()

	app.Get("/test-cookie", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		cookie := &http.Cookie{Name: "test", Value: "cookie_value"}
		res.Cookie(cookie)
	})

	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/test-cookie")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	cookies := resp.Cookies()
	var cookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "test" {
			cookie = c
			break
		}
	}

	if cookie == nil {
		t.Fatalf("Cookie 'test' not found")
	}

	if cookie.Value != "cookie_value" {
		t.Errorf("Expected cookie 'test' to have value 'cookie_value', got '%s'", cookie.Value)
	}
}

func TestResponseSignedCookie(t *testing.T) {
	app := coco.NewApp()
	secret := "secret_key"

	app.Get("/test-signed-cookie", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		cookie := &http.Cookie{Name: "signed_test", Value: "signed_value"}
		res.SignedCookie(cookie, secret)
	})

	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/test-signed-cookie")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	cookies := resp.Cookies()
	var cookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "signed_test" {
			cookie = c
			break
		}
	}

	if cookie == nil {
		t.Fatalf("Signed cookie 'signed_test' not found")
	}

	// Verify the signature
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		t.Fatalf("Expected signed cookie value to have two parts, got %d", len(parts))
	}

	signature, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("Error decoding signature: %v", err)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(parts[1])) // The original value
	expectedMAC := mac.Sum(nil)

	if !hmac.Equal(signature, expectedMAC) {
		t.Errorf("Signature does not match")
	}
}

func TestResponseClearCookie(t *testing.T) {
	app := coco.NewApp()

	app.Get("/clear-cookie", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		// Assuming ClearCookie is implemented to take the name of the cookie
		res.ClearCookie("test")
	})

	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/clear-cookie")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	// Check if the 'Set-Cookie' header is present and contains 'Max-Age=-1'
	cookies := resp.Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == "test" && cookie.MaxAge == -1 {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected cookie 'test' to be cleared with 'Max-Age=-1'")
	}
}

func TestDownload(t *testing.T) {
	app := coco.NewApp()

	tmpFile, err := os.CreateTemp("", "testfile-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up after the test.

	fileName := filepath.Base(tmpFile.Name())
	filePath := tmpFile.Name()

	// Define test cases as a slice of anonymous structs.
	tests := []struct {
		name       string
		path       string
		filename   string
		options    *coco.DownloadOption
		wantErr    bool
		errChecker func(error) bool
	}{
		{
			name:     "DefaultOptions",
			path:     filePath,
			filename: fileName,
			options:  nil,
			wantErr:  false,
		},
		{
			name:     "DenyDotfiles",
			path:     "/path/to/.dotfile",
			filename: ".dotfile",
			options: &coco.DownloadOption{
				Dotfiles: "deny",
			},
			wantErr:    true,
			errChecker: func(err error) bool { return errors.Is(err, coco.ErrDotfilesDeny) },
		},
		{
			name:     "CustomOptions",
			path:     filePath,
			filename: fileName,
			options: &coco.DownloadOption{
				MaxAge: 3600,
				Headers: map[string]string{
					"X-Custom-Header": "CustomValue",
				},
			},
			wantErr: false,
		},
		{
			name:     "ErrorHandling",
			path:     "/path/to/non-existent/file.txt",
			filename: "file.txt",
			options:  nil,
			wantErr:  true,
		},
	}

	// Iterate over each test case.
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app.Get("/download-"+tc.name, func(res coco.Response, req *coco.Request, next coco.NextFunc) {
				res.Download(tc.path, tc.filename, tc.options, func(err error) {
					if (err != nil) != tc.wantErr {
						t.Errorf("Download() error = %v, wantErr %v", err, tc.wantErr)
					}
					if tc.wantErr && tc.errChecker != nil && !tc.errChecker(err) {
						t.Errorf("Download() error = %v, does not match expected error", err)
					}
				})
			})

			srv := httptest.NewServer(app.GetHandler())
			defer srv.Close()

			resp, err := http.Get(srv.URL + "/download-" + tc.name)
			if err != nil {
				t.Fatalf("Failed to make GET request: %v", err)
			}
			defer resp.Body.Close()

			// Additional checks can be added here to verify headers and response content.
		})
	}
}

func TestSendFile(t *testing.T) {

	type handlerSetupResult struct {
		routePath    string
		tempFilePath string
		cleanup      func()
	}
	// Define a struct for test cases
	type testCase struct {
		name         string
		setupHandler func(*testing.T, *coco.App) handlerSetupResult
		validate     func(*testing.T, *http.Response, handlerSetupResult)
		expectError  bool
	}

	// Define the test cases
	testCases := []testCase{
		{
			name: "NormalOperation",
			setupHandler: func(t *testing.T, app *coco.App) handlerSetupResult {

				tmpFile, err := os.CreateTemp("", "testfile-*.txt")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}

				fileName := filepath.Base(tmpFile.Name())
				filePath := tmpFile.Name()

				app.Get("/send-file", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
					options := &coco.DownloadOption{
						MaxAge:       3600,
						LastModified: true,
						Headers: map[string]string{
							"X-Custom-Header": "TestValue",
						},
						AcceptRanges: true,
						CacheControl: true,
						Immutable:    true,
					}
					res.SendFile(filePath, fileName, options, nil)
				})

				return handlerSetupResult{
					routePath:    "/send-file",
					tempFilePath: fileName,
					cleanup:      func() { os.Remove(tmpFile.Name()) },
				}
			},
			validate: func(t *testing.T, resp *http.Response, setup handlerSetupResult) {
				expectedMimeType := "text/plain; charset=utf-8"
				if resp.Header.Get("Content-Type") != expectedMimeType {
					t.Errorf("Expected Content-Type '%s', got '%s'", expectedMimeType, resp.Header.Get("Content-Type"))
				}

				// Test Cache-Control header
				expectedCacheControl := "public, max-age=3600, immutable"
				if resp.Header.Get("Cache-Control") != expectedCacheControl {
					t.Errorf("Expected Cache-Control '%s', got '%s'", expectedCacheControl, resp.Header.Get("Cache-Control"))
				}
				// Test Last-Modified header
				if resp.Header.Get("Last-Modified") == "" {
					t.Error("Expected Last-Modified header to be set")
				}

				// Test Accept-Ranges header
				if resp.Header.Get("Accept-Ranges") != "bytes" {
					t.Error("Expected Accept-Ranges header to be 'bytes'")
				}

				// Test custom header
				if resp.Header.Get("X-Custom-Header") != "TestValue" {
					t.Errorf("Expected custom header 'X-Custom-Header' to be 'TestValue'")
				}

				// Test Content-Disposition header
				expectedContentDisposition := "attachment; filename*=UTF-8''" + url.PathEscape(setup.tempFilePath)
				if resp.Header.Get("Content-Disposition") != expectedContentDisposition {
					t.Errorf("Expected Content-Disposition '%s', got '%s'", expectedContentDisposition, resp.Header.Get("Content-Disposition"))
				}
			},
			expectError: false,
		},
		{
			name: "NilCallback",
			setupHandler: func(t *testing.T, app *coco.App) handlerSetupResult {
				return handlerSetupResult{
					routePath: "/send-file-nil-callback",
					cleanup:   func() {},
				}
			},
			validate:    func(t *testing.T, resp *http.Response, setup handlerSetupResult) {},
			expectError: false,
		},
		{
			name: "StatError",
			setupHandler: func(t *testing.T, app *coco.App) handlerSetupResult {
				nonExistentFilePath := "/path/to/non-existent/file.txt"
				app.Get("/send-file-stat-error", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
					res.SendFile(nonExistentFilePath, "file.txt", nil, func(err error) {
						if err == nil {
							t.Error("Expected an error when stating a non-existent file path, got nil")
						}
					})
				})
				return handlerSetupResult{
					routePath: "/send-file-stat-error",
					cleanup:   func() {},
				}
			},
			validate:    func(t *testing.T, resp *http.Response, setup handlerSetupResult) {},
			expectError: false,
		},
		{
			name: "DirectoryError",
			setupHandler: func(t *testing.T, app *coco.App) handlerSetupResult {
				dirPath := os.TempDir()
				app.Get("/send-file-dir-error", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
					res.SendFile(dirPath, "dir", nil, func(err error) {
						if err == nil {
							t.Error("Expected an error when sending a directory, got nil")
						}
					})
				})
				return handlerSetupResult{
					routePath: "/send-file-dir-error",
					cleanup:   func() {},
				}
			},
			validate: func(t *testing.T, resp *http.Response, setup handlerSetupResult) {

			},
		},
		{
			name: "OpenError",
			setupHandler: func(t *testing.T, app *coco.App) handlerSetupResult {
				tmpFile, err := os.CreateTemp("", "testfile-*.txt")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				tmpFile.Close()
				os.Remove(tmpFile.Name()) // Delete the file to induce an open error

				fileName := filepath.Base(tmpFile.Name())
				filePath := tmpFile.Name()

				app.Get("/send-file-open-error", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
					res.SendFile(filePath, fileName, nil, func(err error) {
						if err == nil {
							t.Error("Expected an error when opening a non-existent file, got nil")
						}
					})
				})
				return handlerSetupResult{
					routePath:    "/send-file-open-error",
					tempFilePath: filePath,
					cleanup:      func() {},
				}
			},
			validate: func(t *testing.T, resp *http.Response, setup handlerSetupResult) {
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := coco.NewApp()

			setup := tc.setupHandler(t, app)

			defer setup.cleanup()
			srv := httptest.NewServer(app.GetHandler())
			defer srv.Close()
			resp, err := http.Get(srv.URL + setup.routePath)
			if err != nil {
				t.Fatalf("Failed to make GET request: %v", err)
			}
			defer resp.Body.Close()

			if tc.validate != nil {
				tc.validate(t, resp, setup)
			}

			if tc.expectError && err == nil {
				t.Errorf("Expected error, got nil")
			} else if !tc.expectError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestResponseJSON(t *testing.T) {

	app := coco.NewApp()

	app.Get("/test-json", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.JSON(map[string]string{"status": "ok"})
	})

	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/test-json")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	if !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		t.Errorf("Expected Content-Type to be 'application/json', got '%s'", resp.Header.Get("Content-Type"))
	}
}

func TestResponseSend(t *testing.T) {
	app := coco.NewApp()

	app.Get("/test-send", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("Hello World")
	})

	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/test-send")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Errorf("Expected Content-Type to be 'text/plain; charset=utf-8', got '%s'", resp.Header.Get("Content-Type"))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != "Hello World" {
		t.Errorf("Expected response body to be 'Hello World', got '%s'", string(body))
	}
}

func TestResponseSet(t *testing.T) {

	app := coco.NewApp()

	app.Get("/test-set", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Set("X-Custom-Header", "TestValue")
		res.Send("Hello World")
	})

	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/test-set")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Custom-Header") != "TestValue" {
		t.Errorf("Expected X-Custom-Header to be 'TestValue', got '%s'", resp.Header.Get("X-Custom-Header"))
	}
}

func TestResponseSendStatus(t *testing.T) {
	app := coco.NewApp()

	app.Get("/test-send-status", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.SendStatus(http.StatusNotFound)
	})

	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/test-send-status")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status code to be %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestResponseStatus(t *testing.T) {
	app := coco.NewApp()

	app.Get("/test-response-status", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Status(http.StatusNotFound).Send("Not Found")
	})

	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/test-response-status")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status code to be %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestResponseType(t *testing.T) {
	app := coco.NewApp()

	app.Get("/test-response-type", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Type("text/plain").Send("Hello World")
	})

	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/test-response-type")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	expectedContentType := "text/plain; charset=utf-8"
	if resp.Header.Get("Content-Type") != expectedContentType {
		t.Errorf("Expected Content-Type to be '%s', got '%s'", expectedContentType, resp.Header.Get("Content-Type"))
	}
}

func TestResponseVary(t *testing.T) {
	app := coco.NewApp()

	app.Get("/test-response-vary", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Vary("Origin")
		res.Send("Hello World")
	})

	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/test-response-vary")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Vary") != "Origin" {
		t.Errorf("Expected Vary to be 'Origin', got '%s'", resp.Header.Get("Vary"))
	}
}

func TestResponseGet(t *testing.T) {
	app := coco.NewApp()

	app.Get("/test-response-get", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Set("X-Custom-Header", "TestValue")

		if res.Get("X-Custom-Header") != "TestValue" {
			t.Errorf("Expected X-Custom-Header to be 'TestValue', got '%s'", res.Get("X-Custom-Header"))
		}
		res.SendStatus(http.StatusOK)
	})

	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	_, err := http.Get(srv.URL + "/test-response-get")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}

}

func TestResponseRedirect(t *testing.T) {
	app := coco.NewApp()

	app.Get("/test-redirect", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Status(http.StatusCreated).Send("Redirect successful")
	})

	app.Get("/test-response-redirect", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Redirect("/test-redirect")
	})

	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/test-response-redirect")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status code to be %d, got %d", http.StatusCreated, resp.StatusCode)
	}

}

func TestResponseLocation(t *testing.T) {
	app := coco.NewApp()

	app.Get("/test-response-location", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Location("/test-location")
		res.SendStatus(http.StatusOK)
	})

	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/test-response-location")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	location, err := resp.Location()
	if err != nil {
		t.Fatalf("Failed to get Location header: %v", err)
	}

	if location.Path != "/test-location" {
		t.Errorf("Expected Location header to be '/test-location', got '%s'", location.Path)
	}
}

func TestResponseRender(t *testing.T) {
	appFS := afero.NewMemMapFs()

	htmlTemplate := `<h1>Hello World</h1>`
	afero.WriteFile(appFS, "templates/test.html", []byte(htmlTemplate), 0644)
	afero.WriteFile(appFS, "templates/layout.html", []byte(htmlTemplate), 0644)

	app := coco.NewApp()

	err := app.LoadTemplates(afero.NewIOFS(appFS), &coco.TemplateConfig{
		Ext: ".html",
	})
	assert.NoError(t, err)

	app.Get("/test-response-render", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Render("test", nil)
	})

	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/test-response-render")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	expectedBody := "<h1>Hello World</h1>"
	assert.Equal(t, expectedBody, string(body))
}

func TestResponseStatusCode(t *testing.T) {
	app := coco.NewApp()

	app.Use(func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		next(res, req)
		log.Printf("Status code: %d", res.StatusCode())
		if res.StatusCode() != 401 {
			t.Errorf("Expected status code to be 401, got %d", res.StatusCode())
		}
	})
	
	app.Get("/test-response-status-code", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Status(401)
		res.Send("Unauthorized")
	})

	srv := httptest.NewServer(app.GetHandler())
	defer srv.Close()

	_, err := http.Get(srv.URL + "/test-response-status-code")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
}
