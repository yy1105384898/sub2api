package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/alitto/pond/v2"
)

// relayRunnerSvc 抽出 runner 实际依赖的两个方法，便于单测注入 stub。
type relayRunnerSvc interface {
	ListEnabledMonitors(ctx context.Context) ([]*RelayMonitor, error)
	RunProbe(ctx context.Context, id int64) error
}

// RelayMonitorRunner 中转站监控调度器。
//
// 设计与 ChannelMonitorRunner 一致：每个 enabled 监控一个独立 goroutine + ticker，
// 实际探测交给 pond 池（容量 relayWorkerConcurrency）；Service CRUD 后通过
// RelayScheduler 接口即时重建/取消任务。
type RelayMonitorRunner struct {
	svc relayRunnerSvc

	pool         pond.Pool
	parentCtx    context.Context
	parentCancel context.CancelFunc

	mu      sync.Mutex
	tasks   map[int64]*scheduledRelay
	wg      sync.WaitGroup
	started bool
	stopped bool

	inFlight   map[int64]struct{}
	inFlightMu sync.Mutex
}

type scheduledRelay struct {
	id       int64
	name     string
	interval time.Duration
	cancel   context.CancelFunc
}

// NewRelayMonitorRunner 构造调度器。Start 在 wire 中调用一次。
func NewRelayMonitorRunner(svc *RelayMonitorService) *RelayMonitorRunner {
	return newRelayMonitorRunner(svc)
}

func newRelayMonitorRunner(svc relayRunnerSvc) *RelayMonitorRunner {
	ctx, cancel := context.WithCancel(context.Background())
	return &RelayMonitorRunner{
		svc:          svc,
		pool:         pond.NewPool(relayWorkerConcurrency),
		parentCtx:    ctx,
		parentCancel: cancel,
		tasks:        make(map[int64]*scheduledRelay),
		inFlight:     make(map[int64]struct{}),
	}
}

// Start 加载所有 enabled 监控并为每个建立独立定时任务。调用方保证只调一次。
func (r *RelayMonitorRunner) Start() {
	if r == nil || r.svc == nil {
		return
	}
	r.mu.Lock()
	if r.started || r.stopped {
		r.mu.Unlock()
		return
	}
	r.started = true
	r.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), relayStartupLoadTimeout)
	defer cancel()
	enabled, err := r.svc.ListEnabledMonitors(ctx)
	if err != nil {
		slog.Error("relay_monitor: load enabled monitors failed at startup", "error", err)
		return
	}
	for _, m := range enabled {
		r.Schedule(m)
	}
	slog.Info("relay_monitor: runner started", "scheduled_tasks", len(enabled))
}

// Schedule 为指定监控创建（或重置）定时任务；m.Enabled=false 等同 Unschedule。
func (r *RelayMonitorRunner) Schedule(m *RelayMonitor) {
	if r == nil || m == nil {
		return
	}
	if !m.Enabled {
		r.Unschedule(m.ID)
		return
	}
	interval := time.Duration(m.IntervalSeconds) * time.Second
	if interval <= 0 {
		slog.Error("relay_monitor: skip schedule for invalid interval", "monitor_id", m.ID, "interval_seconds", m.IntervalSeconds)
		return
	}

	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return
	}
	if !r.started {
		r.mu.Unlock()
		slog.Warn("relay_monitor: schedule before runner started, skip", "monitor_id", m.ID, "name", m.Name)
		return
	}
	if existing, ok := r.tasks[m.ID]; ok {
		existing.cancel()
	}
	ctx, cancel := context.WithCancel(r.parentCtx)
	task := &scheduledRelay{id: m.ID, name: m.Name, interval: interval, cancel: cancel}
	r.tasks[m.ID] = task
	r.wg.Add(1)
	r.mu.Unlock()

	go r.runScheduled(ctx, task)
}

// Unschedule 取消指定监控的定时任务（若存在）。
func (r *RelayMonitorRunner) Unschedule(id int64) {
	if r == nil {
		return
	}
	r.mu.Lock()
	task, ok := r.tasks[id]
	if ok {
		delete(r.tasks, id)
	}
	r.mu.Unlock()
	if ok {
		task.cancel()
	}
}

// Stop 优雅停止：取消所有任务、关闭池。
func (r *RelayMonitorRunner) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return
	}
	r.stopped = true
	r.parentCancel()
	r.tasks = nil
	r.mu.Unlock()

	r.wg.Wait()
	r.pool.StopAndWait()
}

// runScheduled 单监控循环：立即首探一次，之后按 interval 周期触发。
func (r *RelayMonitorRunner) runScheduled(ctx context.Context, task *scheduledRelay) {
	defer r.wg.Done()

	r.fire(ctx, task)

	ticker := time.NewTicker(task.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.fire(ctx, task)
		}
	}
}

// fire 提交一次探测到 worker 池；重复在飞或池满时跳过。
func (r *RelayMonitorRunner) fire(_ context.Context, task *scheduledRelay) {
	if !r.tryAcquireInFlight(task.id) {
		slog.Debug("relay_monitor: skip already in-flight", "monitor_id", task.id, "name", task.name)
		return
	}
	if _, ok := r.pool.TrySubmit(func() {
		r.runOne(task.id, task.name)
	}); !ok {
		r.releaseInFlight(task.id)
		slog.Warn("relay_monitor: worker pool full, skip submission", "monitor_id", task.id, "name", task.name)
	}
}

func (r *RelayMonitorRunner) tryAcquireInFlight(id int64) bool {
	r.inFlightMu.Lock()
	defer r.inFlightMu.Unlock()
	if _, exists := r.inFlight[id]; exists {
		return false
	}
	r.inFlight[id] = struct{}{}
	return true
}

func (r *RelayMonitorRunner) releaseInFlight(id int64) {
	r.inFlightMu.Lock()
	delete(r.inFlight, id)
	r.inFlightMu.Unlock()
}

// runOne 执行单监控探测。错误只记日志，不熔断。
func (r *RelayMonitorRunner) runOne(id int64, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), relayProbeTimeout+relayRunOneBuffer)
	defer cancel()

	defer r.releaseInFlight(id)
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("relay_monitor: runner panic", "monitor_id", id, "name", name, "panic", rec)
		}
	}()

	if err := r.svc.RunProbe(ctx, id); err != nil {
		slog.Warn("relay_monitor: probe failed", "monitor_id", id, "name", name, "error", err)
	}
}
