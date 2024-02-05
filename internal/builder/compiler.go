package builder

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/x1unix/go-playground/internal/builder/storage"
	"github.com/x1unix/go-playground/pkg/util/osutil"
	"go.uber.org/zap"
)

// predefinedBuildVars is list of environment vars which contain build values
var predefinedBuildVars = osutil.EnvironmentVariables{
	"CGO_ENABLED": "0",
	"GOOS":        "js",
	"GOARCH":      "wasm",
	"GOCACHE":     "off",
	"HOME":        os.Getenv("HOME"),
}

// Result is WASM build result
type Result struct {
	// FileName is artifact file name
	FileName string
}

// BuildEnvironmentConfig is BuildService environment configuration.
type BuildEnvironmentConfig struct {
	// IncludedEnvironmentVariables is a list included environment variables for build.
	IncludedEnvironmentVariables osutil.EnvironmentVariables

	// KeepGoModCache disables Go modules cache cleanup.
	KeepGoModCache bool
}

// BuildService is WASM build service
type BuildService struct {
	log     *zap.Logger
	config  BuildEnvironmentConfig
	storage storage.StoreProvider
}

// NewBuildService is BuildService constructor
func NewBuildService(log *zap.Logger, cfg BuildEnvironmentConfig, store storage.StoreProvider) BuildService {
	return BuildService{
		log:     log.Named("builder"),
		config:  cfg,
		storage: store,
	}
}

func (s BuildService) getEnvironmentVariables() []string {
	if len(s.config.IncludedEnvironmentVariables) == 0 {
		return predefinedBuildVars.Join()
	}

	return s.config.IncludedEnvironmentVariables.Concat(predefinedBuildVars).Join()
}

// GetArtifact returns artifact by id
func (s BuildService) GetArtifact(id storage.ArtifactID) (storage.ReadCloseSizer, error) {
	return s.storage.GetItem(id)
}

// Build compiles Go source to WASM and returns result
func (s BuildService) Build(ctx context.Context, files map[string][]byte) (*Result, error) {
	aid, err := storage.GetArtifactID(files)
	if err != nil {
		return nil, err
	}

	result := &Result{FileName: aid.Ext(storage.ExtWasm)}
	isCached, err := s.storage.HasItem(aid)
	if err != nil {
		s.log.Error("failed to check cache", zap.Stringer("artifact", aid), zap.Error(err))
		return nil, err
	}

	if isCached {
		// Just return precompiled result if data is cached already
		s.log.Debug("build cached, returning cached file", zap.Stringer("artifact", aid))
		return result, nil
	}

	workspace, err := s.storage.CreateWorkspace(aid, files)
	if err != nil {
		return nil, err
	}

	err = s.buildSource(ctx, workspace)
	return result, err
}

func (s BuildService) buildSource(ctx context.Context, workspace *storage.Workspace) error {
	cmd := newGoToolCommand(ctx, "build", "-o", workspace.BinaryPath, ".")
	cmd.Dir = workspace.WorkDir
	cmd.Env = s.getEnvironmentVariables()
	buff := &bytes.Buffer{}
	cmd.Stderr = buff

	s.log.Debug(
		"starting go build", zap.Strings("command", cmd.Args), zap.Strings("env", cmd.Env),
	)
	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		errMsg := buff.String()
		s.log.Debug("build failed", zap.Error(err), zap.String("stderr", errMsg))
		return newBuildError(errMsg)
	}

	return nil
}

// CleanJobName implements' builder.Cleaner interface.
func (s BuildService) CleanJobName() string {
	return "gocache"
}

// Clean implements' builder.Cleaner interface.
//
// Cleans go build and modules cache.
func (s BuildService) Clean(ctx context.Context) error {
	if s.config.KeepGoModCache {
		s.log.Info("go mod cache cleanup is disabled, skip")
		return nil
	}

	cmd := newGoToolCommand(ctx, "clean", "-modcache", "-cache", "-testcache", "-fuzzcache")
	cmd.Env = s.getEnvironmentVariables()
	buff := &bytes.Buffer{}
	cmd.Stderr = buff

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command %s: %w", cmd.Args, err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("process returned error: %s. Stderr: %s", err, buff.String())
	}

	return nil
}
