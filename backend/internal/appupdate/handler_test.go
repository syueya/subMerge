package appupdate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/updater"
)

func TestHandlerStatusAndDisabledCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old := updater.PublicKeyBase64
	updater.PublicKeyBase64 = ""
	t.Cleanup(func() { updater.PublicKeyBase64 = old })
	runtimeOptions := testRuntime(t)
	service := NewServiceWithOptions(ServiceOptions{
		CurrentVersion: "1.1.9", Runtime: &runtimeOptions, Installer: &recordingInstaller{},
	})
	handler := NewHandler(service)
	router := gin.New()
	router.GET("/api/update/status", handler.Status)
	router.POST("/api/update/check", handler.Check)

	statusResponse := httptest.NewRecorder()
	router.ServeHTTP(statusResponse, httptest.NewRequest(http.MethodGet, "/api/update/status", nil))
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("status code = %d, body=%s", statusResponse.Code, statusResponse.Body.String())
	}
	var statusBody common.ApiResponse[Status]
	if err := json.Unmarshal(statusResponse.Body.Bytes(), &statusBody); err != nil {
		t.Fatal(err)
	}
	if !statusBody.OK || statusBody.Data == nil || statusBody.Data.Enabled {
		t.Fatalf("status body = %+v", statusBody)
	}

	checkResponse := httptest.NewRecorder()
	router.ServeHTTP(checkResponse, httptest.NewRequest(http.MethodPost, "/api/update/check", nil))
	if checkResponse.Code != http.StatusServiceUnavailable {
		t.Fatalf("check code = %d, body=%s", checkResponse.Code, checkResponse.Body.String())
	}
}

func TestHandlerInstallReturnsAcceptedBeforeDispatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	publicKey := make([]byte, 32)
	runtimeOptions := testRuntime(t)
	dispatched := make(chan ShutdownRequest, 1)
	service := NewServiceWithOptions(ServiceOptions{
		CurrentVersion: "1.1.9", PublicKey: publicKey, Runtime: &runtimeOptions, Installer: &recordingInstaller{},
		Shutdown: func(request ShutdownRequest) { dispatched <- request },
	})
	service.mu.Lock()
	service.status.Phase = PhaseReady
	service.status.Available = true
	service.status.Staged = true
	service.check = &updater.CheckResult{LatestVersion: "1.2.0", Available: true, Asset: updater.Asset{SHA256: string(make([]byte, 64))}}
	service.download = &updater.DownloadResult{Path: "staged"}
	service.mu.Unlock()

	handler := NewHandler(service)
	router := gin.New()
	router.POST("/api/update/install", handler.Install)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/update/install", nil))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("install code = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	select {
	case request := <-dispatched:
		if request.Action != ShutdownInstall {
			t.Fatalf("request = %+v", request)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown callback was not dispatched")
	}
}
