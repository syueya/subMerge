package appupdate

import (
	"context"
	"crypto/ed25519"
	"errors"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/submerge/submerge/backend/internal/updater"
)

const DefaultManifestURL = "https://github.com/syueya/subMerge/releases/latest/download/update-manifest.json"

type Phase string

const (
	PhaseIdle        Phase = "idle"
	PhaseChecking    Phase = "checking"
	PhaseAvailable   Phase = "available"
	PhaseDownloading Phase = "downloading"
	PhaseReady       Phase = "ready"
	PhaseInstalling  Phase = "installing"
	PhaseRestarting  Phase = "restarting"
	PhaseFailed      Phase = "failed"
)

var (
	ErrDisabled     = errors.New("online updates are disabled")
	ErrBusy         = errors.New("another update operation is already running")
	ErrNotAvailable = errors.New("no newer update is available")
	ErrNotReady     = errors.New("no verified update is ready to install")
	ErrNoShutdown   = errors.New("update shutdown callback is not configured")
	ErrStaleRequest = errors.New("shutdown request is no longer pending")
)

type Status struct {
	Enabled           bool           `json:"enabled"`
	DisabledReason    string         `json:"disabledReason,omitempty"`
	Phase             Phase          `json:"phase"`
	CurrentVersion    string         `json:"currentVersion"`
	LatestVersion     string         `json:"latestVersion,omitempty"`
	Available         bool           `json:"available"`
	ReleaseURL        string         `json:"releaseUrl,omitempty"`
	Notes             string         `json:"notes,omitempty"`
	PublishedAt       *time.Time     `json:"publishedAt,omitempty"`
	Asset             *updater.Asset `json:"asset,omitempty"`
	Downloaded        int64          `json:"downloaded,omitempty"`
	Total             int64          `json:"total,omitempty"`
	Error             string         `json:"error,omitempty"`
	CheckedAt         *time.Time     `json:"checkedAt,omitempty"`
	Staged            bool           `json:"staged"`
	RollbackAvailable bool           `json:"rollbackAvailable"`
	RollbackVersion   string         `json:"rollbackVersion,omitempty"`
	RestartMode       string         `json:"restartMode,omitempty"`
}

type ShutdownAction string

const (
	ShutdownInstall  ShutdownAction = "install"
	ShutdownRollback ShutdownAction = "rollback"
)

// ShutdownRequest is emitted only after the handler has written 202 Accepted.
// Execute must be called after HTTP shutdown and SQLite close. On success the
// caller exits the process and lets systemd or Docker restart policy restart it.
type ShutdownRequest struct {
	Action      ShutdownAction
	RestartMode string
	id          uint64
	service     *Service
}

func (r ShutdownRequest) Execute(ctx context.Context) error {
	if r.service == nil {
		return ErrStaleRequest
	}
	return r.service.execute(ctx, r.id, r.Action)
}

type ShutdownFunc func(ShutdownRequest)

type ServiceOptions struct {
	CurrentVersion  string
	ManifestURL     string
	DBPath          string
	Shutdown        ShutdownFunc
	HTTPClient      *http.Client
	PublicKey       ed25519.PublicKey
	Runtime         *updater.RuntimeOptions
	Installer       updater.Installer
	Now             func() time.Time
	DownloadTimeout time.Duration
}

type Service struct {
	mu              sync.Mutex
	status          Status
	source          updater.ReleaseSource
	downloader      updater.Downloader
	installer       updater.Installer
	runtime         updater.RuntimeOptions
	dbPath          string
	shutdown        ShutdownFunc
	now             func() time.Time
	downloadTimeout time.Duration
	check           *updater.CheckResult
	download        *updater.DownloadResult
	pending         *pendingOperation
	operationID     uint64
}

type pendingOperation struct {
	id      uint64
	action  ShutdownAction
	install updater.InstallRequest
}

func NewService(currentVersion, manifestURL, dbPath string, shutdown ShutdownFunc) *Service {
	return NewServiceWithOptions(ServiceOptions{
		CurrentVersion: currentVersion,
		ManifestURL:    manifestURL,
		DBPath:         dbPath,
		Shutdown:       shutdown,
	})
}

func NewServiceWithOptions(options ServiceOptions) *Service {
	if strings.TrimSpace(options.ManifestURL) == "" {
		options.ManifestURL = DefaultManifestURL
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.DownloadTimeout <= 0 {
		options.DownloadTimeout = 30 * time.Minute
	}
	s := &Service{
		status: Status{
			Enabled:        true,
			Phase:          PhaseIdle,
			CurrentVersion: strings.TrimSpace(options.CurrentVersion),
		},
		dbPath:          filepath.Clean(options.DBPath),
		shutdown:        options.Shutdown,
		now:             options.Now,
		downloadTimeout: options.DownloadTimeout,
	}

	publicKey := options.PublicKey
	if len(publicKey) == 0 {
		var err error
		publicKey, err = updater.EmbeddedPublicKey()
		if err != nil {
			s.disable(err)
		}
	}
	runtimeOptions := options.Runtime
	if runtimeOptions == nil {
		resolved, err := updater.RuntimeOptionsFromEnv()
		if err != nil {
			s.disable(err)
		} else {
			runtimeOptions = &resolved
		}
	}
	if runtimeOptions != nil {
		s.runtime = *runtimeOptions
		s.status.RestartMode = runtimeOptions.RestartMode
	}
	if options.Installer != nil {
		s.installer = options.Installer
	} else if runtimeOptions != nil {
		s.installer = updater.NewFileInstaller(runtimeOptions.TargetPath, runtimeOptions.RollbackDir)
	}
	if s.installer == nil {
		s.disable(errors.New("update installer is not configured"))
	}
	s.source = updater.ReleaseSource{
		ManifestURL: options.ManifestURL,
		PublicKey:   publicKey,
		HTTPClient:  options.HTTPClient,
	}
	s.downloader = updater.Downloader{HTTPClient: options.HTTPClient}
	if _, err := updater.ParseVersion(s.status.CurrentVersion); err != nil {
		s.disable(err)
	}
	return s
}

func (s *Service) disable(err error) {
	s.status.Enabled = false
	if err != nil && s.status.DisabledReason == "" {
		s.status.DisabledReason = err.Error()
	}
}

func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statusLocked()
}

func (s *Service) statusLocked() Status {
	status := s.status
	if status.Asset != nil {
		asset := *status.Asset
		status.Asset = &asset
	}
	if status.CheckedAt != nil {
		checkedAt := *status.CheckedAt
		status.CheckedAt = &checkedAt
	}
	if s.installer != nil {
		state, err := s.installer.State()
		status.RollbackAvailable = err == nil && state.RolledBackAt == nil
		if status.RollbackAvailable {
			status.RollbackVersion = state.PreviousVersion
		}
	}
	return status
}

// Check is the only operation intended for both HTTP and background startup
// use. It is synchronous so callers know exactly which signed release won.
func (s *Service) Check(ctx context.Context) (Status, error) {
	s.mu.Lock()
	if err := s.canStartLocked(); err != nil {
		status := s.statusLocked()
		s.mu.Unlock()
		return status, err
	}
	s.status.Phase = PhaseChecking
	s.status.Error = ""
	s.status.Downloaded = 0
	s.status.Total = 0
	currentVersion := s.status.CurrentVersion
	s.mu.Unlock()

	result, err := s.source.Check(ctx, currentVersion)
	now := s.now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.CheckedAt = &now
	if err != nil {
		s.status.Phase = PhaseFailed
		s.status.Error = err.Error()
		s.status.Available = false
		s.status.LatestVersion = ""
		s.status.ReleaseURL = ""
		s.status.Notes = ""
		s.status.PublishedAt = nil
		s.status.Asset = nil
		s.check = nil
		s.download = nil
		s.status.Staged = false
		return s.statusLocked(), err
	}
	s.check = &result
	s.download = nil
	s.status.LatestVersion = result.LatestVersion
	s.status.Available = result.Available
	s.status.ReleaseURL = result.Manifest.ReleaseURL
	s.status.Notes = result.Manifest.Notes
	publishedAt := result.Manifest.PublishedAt
	s.status.PublishedAt = &publishedAt
	s.status.Asset = &result.Asset
	s.status.Staged = false
	if result.Available {
		s.status.Phase = PhaseAvailable
	} else {
		s.status.Phase = PhaseIdle
	}
	return s.statusLocked(), nil
}

func (s *Service) StartDownload() (Status, error) {
	s.mu.Lock()
	if !s.status.Enabled {
		status := s.statusLocked()
		s.mu.Unlock()
		return status, ErrDisabled
	}
	if s.status.Phase != PhaseAvailable || s.check == nil || !s.check.Available {
		status := s.statusLocked()
		s.mu.Unlock()
		return status, ErrNotAvailable
	}
	s.operationID++
	id := s.operationID
	asset := s.check.Asset
	s.status.Phase = PhaseDownloading
	s.status.Error = ""
	s.status.Downloaded = 0
	s.status.Total = asset.Size
	s.status.Staged = false
	status := s.statusLocked()
	s.mu.Unlock()

	go s.downloadAsset(id, asset)
	return status, nil
}

func (s *Service) downloadAsset(id uint64, asset updater.Asset) {
	ctx, cancel := context.WithTimeout(context.Background(), s.downloadTimeout)
	defer cancel()
	result, err := s.downloader.Download(ctx, asset, filepath.Join(s.runtime.RollbackDir, "staging"), func(progress updater.Progress) {
		s.mu.Lock()
		if s.operationID == id && s.status.Phase == PhaseDownloading {
			s.status.Downloaded = progress.Downloaded
			s.status.Total = progress.Total
		}
		s.mu.Unlock()
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.operationID != id || s.status.Phase != PhaseDownloading {
		return
	}
	if err != nil {
		s.status.Phase = PhaseFailed
		s.status.Error = err.Error()
		s.status.Staged = false
		return
	}
	s.download = &result
	s.status.Phase = PhaseReady
	s.status.Downloaded = result.Size
	s.status.Total = result.Size
	s.status.Staged = true
}

func (s *Service) RequestInstall() (Status, ShutdownRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.status.Enabled {
		return s.statusLocked(), ShutdownRequest{}, ErrDisabled
	}
	if s.shutdown == nil {
		return s.statusLocked(), ShutdownRequest{}, ErrNoShutdown
	}
	if s.status.Phase != PhaseReady || s.check == nil || s.download == nil {
		return s.statusLocked(), ShutdownRequest{}, ErrNotReady
	}
	s.operationID++
	pending := &pendingOperation{
		id:     s.operationID,
		action: ShutdownInstall,
		install: updater.InstallRequest{
			StagedPath:     s.download.Path,
			CurrentVersion: s.status.CurrentVersion,
			NewVersion:     s.check.LatestVersion,
			ExpectedSHA256: s.check.Asset.SHA256,
			DataFiles:      updater.SQLiteDataFiles(s.dbPath),
		},
	}
	s.pending = pending
	s.status.Phase = PhaseInstalling
	s.status.Error = ""
	request := s.shutdownRequestLocked(pending)
	return s.statusLocked(), request, nil
}

func (s *Service) RequestRollback() (Status, ShutdownRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.status.Enabled {
		return s.statusLocked(), ShutdownRequest{}, ErrDisabled
	}
	if s.shutdown == nil {
		return s.statusLocked(), ShutdownRequest{}, ErrNoShutdown
	}
	state, err := s.installer.State()
	if err != nil || state.RolledBackAt != nil {
		return s.statusLocked(), ShutdownRequest{}, updater.ErrNoRollback
	}
	if s.status.Phase == PhaseChecking || s.status.Phase == PhaseDownloading || s.status.Phase == PhaseInstalling || s.status.Phase == PhaseRestarting {
		return s.statusLocked(), ShutdownRequest{}, ErrBusy
	}
	s.operationID++
	pending := &pendingOperation{id: s.operationID, action: ShutdownRollback}
	s.pending = pending
	s.status.Phase = PhaseInstalling
	s.status.Error = ""
	request := s.shutdownRequestLocked(pending)
	return s.statusLocked(), request, nil
}

func (s *Service) shutdownRequestLocked(pending *pendingOperation) ShutdownRequest {
	return ShutdownRequest{Action: pending.action, RestartMode: s.runtime.RestartMode, id: pending.id, service: s}
}

// DispatchShutdown invokes the configured lifecycle callback. Handlers call it
// only after writing their accepted response.
func (s *Service) DispatchShutdown(request ShutdownRequest) {
	go s.shutdown(request)
}

func (s *Service) execute(ctx context.Context, id uint64, action ShutdownAction) error {
	s.mu.Lock()
	if s.pending == nil || s.pending.id != id || s.pending.action != action {
		s.mu.Unlock()
		return ErrStaleRequest
	}
	pending := *s.pending
	s.mu.Unlock()

	var err error
	if action == ShutdownInstall {
		_, err = s.installer.Install(ctx, pending.install)
	} else {
		_, err = s.installer.Rollback(ctx)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending == nil || s.pending.id != id {
		return ErrStaleRequest
	}
	s.pending = nil
	if err != nil {
		s.status.Phase = PhaseFailed
		s.status.Error = err.Error()
		return err
	}
	s.status.Phase = PhaseRestarting
	s.status.Error = ""
	return nil
}

func (s *Service) canStartLocked() error {
	if !s.status.Enabled {
		return ErrDisabled
	}
	switch s.status.Phase {
	case PhaseChecking, PhaseDownloading, PhaseReady, PhaseInstalling, PhaseRestarting:
		return ErrBusy
	default:
		return nil
	}
}

// Platform documents the platform chosen by ReleaseSource.Check and is useful
// to HTTP clients without duplicating runtime logic.
func Platform() string { return runtime.GOOS + "/" + runtime.GOARCH }
