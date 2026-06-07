package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/alitto/pond/v2"
)

type cardRunnerSvc interface {
	ListEnabledMonitors(ctx context.Context) ([]*CardPlatformMonitor, error)
	RunProbe(ctx context.Context, id int64) error
}

type CardPlatformMonitorRunner struct {
	svc          cardRunnerSvc
	pool         pond.Pool
	parentCtx    context.Context
	parentCancel context.CancelFunc
	mu           sync.Mutex
	tasks        map[int64]*scheduledCardPlatform
	wg           sync.WaitGroup
	started      bool
	stopped      bool
	inFlight     map[int64]struct{}
	inFlightMu   sync.Mutex
}

type scheduledCardPlatform struct {
	id       int64
	name     string
	interval time.Duration
	cancel   context.CancelFunc
}

func NewCardPlatformMonitorRunner(svc *CardPlatformMonitorService) *CardPlatformMonitorRunner {
	ctx, cancel := context.WithCancel(context.Background())
	return &CardPlatformMonitorRunner{
		svc:          svc,
		pool:         pond.NewPool(cardWorkerConcurrency),
		parentCtx:    ctx,
		parentCancel: cancel,
		tasks:        make(map[int64]*scheduledCardPlatform),
		inFlight:     make(map[int64]struct{}),
	}
}

func (r *CardPlatformMonitorRunner) Start() {
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
	ctx, cancel := context.WithTimeout(context.Background(), cardStartupLoadTimeout)
	defer cancel()
	items, err := r.svc.ListEnabledMonitors(ctx)
	if err != nil {
		slog.Error("card_monitor: load enabled monitors failed", "error", err)
		return
	}
	for _, m := range items {
		r.Schedule(m)
	}
	slog.Info("card_monitor: runner started", "scheduled_tasks", len(items))
}

func (r *CardPlatformMonitorRunner) Schedule(m *CardPlatformMonitor) {
	if r == nil || m == nil {
		return
	}
	if !m.Enabled {
		r.Unschedule(m.ID)
		return
	}
	interval := time.Duration(m.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = time.Duration(cardDefaultIntervalSeconds) * time.Second
	}
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return
	}
	if !r.started {
		r.mu.Unlock()
		return
	}
	if existing, ok := r.tasks[m.ID]; ok {
		existing.cancel()
	}
	ctx, cancel := context.WithCancel(r.parentCtx)
	task := &scheduledCardPlatform{id: m.ID, name: m.Name, interval: interval, cancel: cancel}
	r.tasks[m.ID] = task
	r.wg.Add(1)
	r.mu.Unlock()
	go r.runScheduled(ctx, task)
}

func (r *CardPlatformMonitorRunner) Unschedule(id int64) {
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

func (r *CardPlatformMonitorRunner) Stop() {
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

func (r *CardPlatformMonitorRunner) runScheduled(ctx context.Context, task *scheduledCardPlatform) {
	defer r.wg.Done()
	r.fire(task)
	ticker := time.NewTicker(task.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.fire(task)
		}
	}
}

func (r *CardPlatformMonitorRunner) fire(task *scheduledCardPlatform) {
	if !r.tryAcquireInFlight(task.id) {
		return
	}
	if _, ok := r.pool.TrySubmit(func() { r.runOne(task.id, task.name) }); !ok {
		r.releaseInFlight(task.id)
		slog.Warn("card_monitor: worker pool full", "monitor_id", task.id)
	}
}

func (r *CardPlatformMonitorRunner) runOne(id int64, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), cardProbeTimeout+cardRunOneBuffer)
	defer cancel()
	defer r.releaseInFlight(id)
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("card_monitor: runner panic", "monitor_id", id, "name", name, "panic", rec)
		}
	}()
	if err := r.svc.RunProbe(ctx, id); err != nil {
		slog.Warn("card_monitor: probe failed", "monitor_id", id, "name", name, "error", err)
	}
}

func (r *CardPlatformMonitorRunner) tryAcquireInFlight(id int64) bool {
	r.inFlightMu.Lock()
	defer r.inFlightMu.Unlock()
	if _, ok := r.inFlight[id]; ok {
		return false
	}
	r.inFlight[id] = struct{}{}
	return true
}

func (r *CardPlatformMonitorRunner) releaseInFlight(id int64) {
	r.inFlightMu.Lock()
	delete(r.inFlight, id)
	r.inFlightMu.Unlock()
}
