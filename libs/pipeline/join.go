package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pzlauncher/libs/contracts"
	"pzlauncher/libs/download"
	"pzlauncher/libs/fixtures"
	"pzlauncher/libs/game"
	"pzlauncher/libs/manifestv1"
	"pzlauncher/libs/modplan"
	"pzlauncher/libs/profile"
	"pzlauncher/libs/providers"
	"pzlauncher/libs/resolver"
	"pzlauncher/libs/session"
)

type JoinResult struct {
	SessionID   string
	ProfilePath string
	Manifest    *manifestv1.ServerManifest
	Plan        *modplan.ResolvedModPlan
	Ready       bool
}

type Service struct {
	cfg Config
}

func NewService(cfg Config) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) RunJoin(ctx context.Context, serverID string, emit Emitter) (*JoinResult, error) {
	sessionID := fmt.Sprintf("session-%d", time.Now().Unix())
	emit(Event{Type: "session.start", SessionID: sessionID, Metadata: map[string]interface{}{"serverId": serverID}})

	reg, err := manifestv1.LoadRegistry(s.cfg.RegistryPath)
	if err != nil {
		return nil, s.fail(emit, sessionID, "PIPELINE_MANIFEST", err)
	}
	desc, err := manifestv1.FindServer(reg, serverID)
	if err != nil {
		return nil, s.fail(emit, sessionID, "PIPELINE_MANIFEST", err)
	}

	manifestPath := filepath.Join(s.cfg.Root, desc.ManifestPath)
	emit(Event{Type: "manifest.loaded", SessionID: sessionID, Metadata: map[string]interface{}{
		"serverId": serverID, "path": manifestPath,
	}})

	manifest, err := manifestv1.LoadFile(manifestPath)
	if err != nil {
		emit(Event{Type: "manifest.failed", SessionID: sessionID, Error: err.Error()})
		return nil, s.fail(emit, sessionID, "PIPELINE_MANIFEST", err)
	}

	emit(Event{Type: "mod.resolve.start", SessionID: sessionID})
	plan, err := modplan.FromManifest(manifest)
	if err != nil {
		return nil, s.fail(emit, sessionID, "PIPELINE_RESOLVER", err)
	}
	emit(Event{Type: "mod.resolve.complete", SessionID: sessionID, Metadata: map[string]interface{}{
		"modCount": len(plan.OrderedMods),
	}})

	queue := download.NewQueueFromPlan(sessionID, serverID, plan)
	_ = s.writeJoinTrace(serverID, sessionID, "ResolveMods", "ok", map[string]interface{}{"modCount": len(plan.OrderedMods)})

	legacy := manifestv1.ToLegacyManifest(manifest)
	if len(s.cfg.DemoSeedMods) > 0 {
		_ = fixtures.SeedCache(legacy, s.cfg.CacheDir, s.cfg.DemoSeedMods)
	}

	r := resolver.NewDefaultResolver()
	resolved, err := r.Resolve(legacy)
	if err != nil {
		return nil, s.fail(emit, sessionID, "PIPELINE_RESOLVER", err)
	}
	resolved = enrichResolved(resolved, plan)

	local := providers.NewLocalCacheProvider(s.cfg.CacheDir)
	steam := providers.NewSteamProvider()
	resolved, decisions, err := providers.ApplyFallback(ctx, resolved, []providers.Provider{local, steam})
	if err != nil {
		return nil, s.fail(emit, sessionID, "PIPELINE_PLAN", err)
	}

	profileID := manifest.ProfileID()
	pb := profile.NewProfileBuilder(s.cfg.ProfilesDir)
	profilePath, err := pb.Prepare(profileID, legacy.ID, resolved, s.cfg.CacheDir)
	if err != nil {
		return nil, s.fail(emit, sessionID, "PIPELINE_PROFILE", err)
	}
	_ = manifestv1.SaveSnapshot(profilePath, manifest)

	for _, m := range plan.OrderedMods {
		item := queue.Find(m.ID)
		if item != nil {
			item.State = download.StatePending
		}
		emit(Event{Type: "download.start", SessionID: sessionID, PackageID: m.ID})
	}

	sessionMgr := session.NewSimpleManager(s.cfg.SessionsDir)
	executor := session.DefaultExecutor(s.cfg.CacheDir)
	executor = wrapProgress(executor, emit, sessionID, queue)

	sess, err := sessionMgr.CreateSession(serverID, profilePath, decisions)
	if err != nil {
		return nil, s.fail(emit, sessionID, "PIPELINE_DOWNLOAD", err)
	}

	if !sess.IsComplete {
		if err := sessionMgr.Execute(ctx, sess, executor); err != nil {
			return nil, s.fail(emit, sessionID, "PIPELINE_DOWNLOAD", err)
		}
	}

	for _, m := range plan.OrderedMods {
		item := queue.Find(m.ID)
		if item != nil {
			item.State = download.StateCompleted
			item.BytesDone = item.BytesTotal
		}
		emit(Event{Type: "download.complete", SessionID: sessionID, PackageID: m.ID})
	}

	_ = s.writeJoinTrace(serverID, sessionID, "Ready", "ok", nil)
	emit(Event{Type: "install.complete", SessionID: sessionID})
	emit(Event{Type: "session.complete", SessionID: sessionID, Metadata: map[string]interface{}{"ready": true, "profilePath": profilePath}})

	return &JoinResult{
		SessionID:   sessionID,
		ProfilePath: profilePath,
		Manifest:    manifest,
		Plan:        plan,
		Ready:       true,
	}, nil
}

func (s *Service) Launch(ctx context.Context, serverID, profilePath string, emit Emitter) error {
	reg, err := manifestv1.LoadRegistry(s.cfg.RegistryPath)
	if err != nil {
		return err
	}
	desc, err := manifestv1.FindServer(reg, serverID)
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(s.cfg.Root, desc.ManifestPath)
	manifest, err := manifestv1.LoadFile(manifestPath)
	if err != nil {
		return err
	}

	emit(Event{Type: "launch.started", SessionID: serverID, Metadata: map[string]interface{}{"serverId": serverID}})

	finder := game.NewSimpleFinder()
	inst, err := finder.FindInstallation()
	if err != nil {
		emit(Event{Type: "launch.failed", SessionID: serverID, Error: err.Error()})
		return err
	}
	launcher := game.NewSimpleLauncher()
	req := contracts.LaunchRequest{
		ServerID:   serverID,
		ProfileID:  profilePath,
		ManifestID: manifest.ServerID + "-v" + manifest.Version,
		LaunchArgs: strings.Join(manifest.LaunchArgs, " "),
	}
	res, err := launcher.Launch(inst, req)
	if err != nil || !res.Success {
		msg := res.Error
		if err != nil {
			msg = err.Error()
		}
		emit(Event{Type: "launch.failed", SessionID: serverID, Error: msg})
		return fmt.Errorf("launch: %s", msg)
	}
	emit(Event{Type: "launch.exited", SessionID: serverID, Metadata: map[string]interface{}{"success": true}})
	return nil
}

func (s *Service) fail(emit Emitter, sessionID, code string, err error) error {
	emit(Event{Type: "error", SessionID: sessionID, Error: fmt.Sprintf("%s: %v", code, err)})
	return fmt.Errorf("%s: %w", code, err)
}

func enrichResolved(pkgs []contracts.ResolvedPackage, plan *modplan.ResolvedModPlan) []contracts.ResolvedPackage {
	byID := make(map[string]modplan.ResolvedMod, len(plan.OrderedMods))
	for _, m := range plan.OrderedMods {
		byID[m.ID] = m
	}
	for i := range pkgs {
		if m, ok := byID[pkgs[i].ID]; ok {
			pkgs[i].Size = m.SizeBytes
			pkgs[i].WorkshopID = m.WorkshopID
			if pkgs[i].DownloadURL == "" {
				pkgs[i].DownloadURL = m.DownloadURL
			}
		}
	}
	return pkgs
}

type progressExecutor struct {
	inner session.Executor
	emit  Emitter
	sid   string
	queue *download.Queue
}

func wrapProgress(inner session.Executor, emit Emitter, sid string, q *download.Queue) session.Executor {
	return &progressExecutor{inner: inner, emit: emit, sid: sid, queue: q}
}

func (p *progressExecutor) Execute(ctx context.Context, exec *contracts.PackageExecution) (*contracts.PackageExecution, error) {
	item := p.queue.Find(exec.PackageID)
	if item != nil {
		item.State = download.StateDownloading
	}
	out, err := p.inner.Execute(ctx, exec)
	if item != nil {
		if err != nil {
			item.State = download.StateFailed
			item.LastError = err.Error()
		} else if out.State == contracts.PackageStateComplete || out.State == contracts.PackageStateSkipped {
			item.State = download.StateCompleted
			item.BytesDone = item.BytesTotal
		}
		p.emit(Event{
			Type:      "download.progress",
			SessionID: p.sid,
			PackageID: exec.PackageID,
			Progress:  &Progress{Current: item.BytesDone, Total: item.BytesTotal, Percent: p.queue.OverallPercent()},
		})
	}
	return out, err
}

func (s *Service) writeJoinTrace(serverID, sessionID, stage, status string, extra map[string]interface{}) error {
	dir := filepath.Join(s.cfg.ProfilesDir, serverID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "join-trace-"+sessionID+".json")
	var doc map[string]interface{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &doc)
	}
	if doc == nil {
		doc = map[string]interface{}{"serverId": serverID, "sessionId": sessionID, "stages": []interface{}{}}
	}
	stages, _ := doc["stages"].([]interface{})
	entry := map[string]interface{}{"name": stage, "status": status, "at": time.Now().Format(time.RFC3339)}
	for k, v := range extra {
		entry[k] = v
	}
	doc["stages"] = append(stages, entry)
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
