package appupdate

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/submerge/submerge/backend/internal/updater"
)

func TestServiceCheckDownloadAndInstallAfterShutdown(t *testing.T) {
	binary := []byte("new-release-binary")
	sum := sha256.Sum256(binary)
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	manifest := testManifest(t, "1.2.0", hex.EncodeToString(sum[:]), int64(len(binary)))
	keyID, _ := updater.PublicKeyID(publicKey)
	envelope := []byte(fmt.Sprintf(`{"algorithm":"ed25519","keyId":%q,"signature":%q}`,
		keyID, base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, manifest))))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest":
			_, _ = w.Write(manifest)
		case "/manifest.sig":
			_, _ = w.Write(envelope)
		case "/binary":
			_, _ = w.Write(binary)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	installer := &recordingInstaller{}
	shutdownRequests := make(chan ShutdownRequest, 1)
	runtimeOptions := testRuntime(t)
	service := NewServiceWithOptions(ServiceOptions{
		CurrentVersion: "1.1.9",
		ManifestURL:    "https://updates.test/manifest",
		DBPath:         filepath.Join(t.TempDir(), "submerge.db"),
		Shutdown:       func(request ShutdownRequest) { shutdownRequests <- request },
		HTTPClient:     rewriteClient(t, server.URL),
		PublicKey:      publicKey,
		Runtime:        &runtimeOptions,
		Installer:      installer,
	})

	status, err := service.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.Phase != PhaseAvailable || !status.Available || status.LatestVersion != "1.2.0" || status.Notes != "release notes" || status.PublishedAt == nil {
		t.Fatalf("checked status = %+v", status)
	}
	status, err = service.StartDownload()
	if err != nil || status.Phase != PhaseDownloading {
		t.Fatalf("download start status=%+v err=%v", status, err)
	}
	status = waitForPhase(t, service, PhaseReady)
	if !status.Staged || status.Downloaded != int64(len(binary)) || status.Total != int64(len(binary)) {
		t.Fatalf("ready status = %+v", status)
	}

	status, request, err := service.RequestInstall()
	if err != nil || status.Phase != PhaseInstalling || request.Action != ShutdownInstall {
		t.Fatalf("install request status=%+v request=%+v err=%v", status, request, err)
	}
	service.DispatchShutdown(request)
	select {
	case dispatched := <-shutdownRequests:
		if dispatched.Action != ShutdownInstall || dispatched.RestartMode != updater.RestartModeExit {
			t.Fatalf("dispatched request = %+v", dispatched)
		}
		if err := dispatched.Execute(context.Background()); err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown callback was not called")
	}
	if status := service.Status(); status.Phase != PhaseRestarting {
		t.Fatalf("post-install status = %+v", status)
	}
	requestSeen := installer.installRequest()
	if requestSeen.NewVersion != "1.2.0" || requestSeen.CurrentVersion != "1.1.9" || len(requestSeen.DataFiles) != 3 {
		t.Fatalf("installer request = %+v", requestSeen)
	}
	if err := request.Execute(context.Background()); !errors.Is(err, ErrStaleRequest) {
		t.Fatalf("reused shutdown request error = %v", err)
	}
}

func TestServiceRejectsBusyAndNotReadyOperations(t *testing.T) {
	publicKey, _, _ := ed25519.GenerateKey(rand.Reader)
	installer := &recordingInstaller{}
	runtimeOptions := testRuntime(t)
	service := NewServiceWithOptions(ServiceOptions{
		CurrentVersion: "1.1.9", ManifestURL: "https://updates.test/manifest",
		Shutdown: func(ShutdownRequest) {}, PublicKey: publicKey, Runtime: &runtimeOptions, Installer: installer,
	})
	if _, err := service.StartDownload(); !errors.Is(err, ErrNotAvailable) {
		t.Fatalf("download without check error = %v", err)
	}
	if _, _, err := service.RequestInstall(); !errors.Is(err, ErrNotReady) {
		t.Fatalf("install without download error = %v", err)
	}
	service.mu.Lock()
	service.status.Phase = PhaseDownloading
	service.mu.Unlock()
	if _, err := service.Check(context.Background()); !errors.Is(err, ErrBusy) {
		t.Fatalf("concurrent check error = %v", err)
	}
	service.mu.Lock()
	service.status.Phase = PhaseReady
	service.mu.Unlock()
	if _, err := service.Check(context.Background()); !errors.Is(err, ErrBusy) {
		t.Fatalf("check while ready error = %v", err)
	}
}

func TestServiceRollbackRunsOnlyAfterShutdown(t *testing.T) {
	publicKey, _, _ := ed25519.GenerateKey(rand.Reader)
	installer := &recordingInstaller{state: updater.InstallState{PreviousVersion: "1.1.9", InstalledVersion: "1.2.0"}, hasState: true}
	requests := make(chan ShutdownRequest, 1)
	runtimeOptions := testRuntime(t)
	service := NewServiceWithOptions(ServiceOptions{
		CurrentVersion: "1.2.0", ManifestURL: "https://updates.test/manifest",
		Shutdown: func(request ShutdownRequest) { requests <- request }, PublicKey: publicKey,
		Runtime: &runtimeOptions, Installer: installer,
	})
	status := service.Status()
	if !status.RollbackAvailable || status.RollbackVersion != "1.1.9" {
		t.Fatalf("rollback status = %+v", status)
	}
	status, request, err := service.RequestRollback()
	if err != nil || status.Phase != PhaseInstalling || request.Action != ShutdownRollback {
		t.Fatalf("rollback request status=%+v request=%+v err=%v", status, request, err)
	}
	if installer.rollbackCount() != 0 {
		t.Fatal("rollback executed before shutdown")
	}
	if err := request.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
	if installer.rollbackCount() != 1 || service.Status().Phase != PhaseRestarting {
		t.Fatalf("rollback count=%d status=%+v", installer.rollbackCount(), service.Status())
	}
}

func TestNewServiceWithoutEmbeddedKeyIsDisabled(t *testing.T) {
	old := updater.PublicKeyBase64
	updater.PublicKeyBase64 = ""
	t.Cleanup(func() { updater.PublicKeyBase64 = old })
	runtimeOptions := testRuntime(t)
	service := NewServiceWithOptions(ServiceOptions{
		CurrentVersion: "1.1.9", Runtime: &runtimeOptions, Installer: &recordingInstaller{},
	})
	status := service.Status()
	if status.Enabled || status.DisabledReason == "" {
		t.Fatalf("disabled status = %+v", status)
	}
	if _, err := service.Check(context.Background()); !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled check error = %v", err)
	}
}

func testManifest(t *testing.T, version, hash string, size int64) []byte {
	t.Helper()
	manifest := updater.Manifest{
		SchemaVersion: updater.ManifestSchemaVersion,
		Version:       version,
		PublishedAt:   time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		ReleaseURL:    "https://github.com/syueya/subMerge/releases/tag/v" + version,
		Notes:         "release notes",
		Assets: []updater.Asset{{
			OS: runtime.GOOS, Arch: runtime.GOARCH, Name: "submerge-" + runtime.GOOS + "-" + runtime.GOARCH,
			URL: "https://updates.test/binary", SHA256: hash, Size: size,
		}},
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func rewriteClient(t *testing.T, serverURL string) *http.Client {
	t.Helper()
	target, err := url.Parse(serverURL)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		clone := req.Clone(req.Context())
		clone.URL.Scheme = target.Scheme
		clone.URL.Host = target.Host
		return http.DefaultTransport.RoundTrip(clone)
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func testRuntime(t *testing.T) updater.RuntimeOptions {
	t.Helper()
	dir := t.TempDir()
	return updater.RuntimeOptions{
		TargetPath: filepath.Join(dir, "submerge"), RollbackDir: filepath.Join(dir, ".submerge-update"), RestartMode: updater.RestartModeExit,
	}
}

func waitForPhase(t *testing.T, service *Service, phase Phase) Status {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status := service.Status()
		if status.Phase == phase {
			return status
		}
		if status.Phase == PhaseFailed {
			t.Fatalf("operation failed: %+v", status)
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for phase %s", phase)
	return Status{}
}

type recordingInstaller struct {
	mu          sync.Mutex
	request     updater.InstallRequest
	state       updater.InstallState
	hasState    bool
	rollbacks   int
	installErr  error
	rollbackErr error
}

func (i *recordingInstaller) Install(_ context.Context, request updater.InstallRequest) (updater.InstallState, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.request = request
	if i.installErr != nil {
		return updater.InstallState{}, i.installErr
	}
	i.state = updater.InstallState{PreviousVersion: request.CurrentVersion, InstalledVersion: request.NewVersion}
	i.hasState = true
	return i.state, nil
}

func (i *recordingInstaller) Rollback(context.Context) (updater.InstallState, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.rollbackErr != nil {
		return updater.InstallState{}, i.rollbackErr
	}
	i.rollbacks++
	now := time.Now()
	i.state.RolledBackAt = &now
	return i.state, nil
}

func (i *recordingInstaller) State() (updater.InstallState, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if !i.hasState {
		return updater.InstallState{}, updater.ErrNoRollback
	}
	return i.state, nil
}

func (i *recordingInstaller) installRequest() updater.InstallRequest {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.request
}

func (i *recordingInstaller) rollbackCount() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.rollbacks
}
