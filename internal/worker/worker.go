package worker

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/zaffka/jigsaw/internal/store"
	"github.com/zaffka/jigsaw/pkg/s3"
	"go.uber.org/zap"
)

const (
	pollInterval = 2 * time.Second
	maxAttempts  = 3
)

// Worker polls the tasks table and processes background jobs.
type Worker struct {
	store *store.Store
	s3    *s3.BucketCli
	log   *zap.Logger
}

// New creates a Worker.
func New(st *store.Store, s3cli *s3.BucketCli, log *zap.Logger) *Worker {
	return &Worker{store: st, s3: s3cli, log: log}
}

// Run starts the polling loop. Blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	w.log.Info("worker started")
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("worker stopped")
			return
		case <-ticker.C:
			// Drain all available tasks before waiting for next tick.
			for {
				claimed := w.poll(ctx)
				if !claimed {
					break
				}
			}
		}
	}
}

// poll claims and processes one task. Returns true if a task was claimed.
func (w *Worker) poll(ctx context.Context) bool {
	task, err := w.store.ClaimTask(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return false
	}
	if err != nil {
		w.log.Error("claim task", zap.Error(err))
		return false
	}

	w.log.Info("processing task",
		zap.String("id", task.ID),
		zap.String("type", task.Type),
		zap.Int("attempt", task.Attempts),
	)

	err = w.dispatch(ctx, task)
	if err == nil {
		if completeErr := w.store.CompleteTask(ctx, task.ID); completeErr != nil {
			w.log.Error("complete task", zap.String("id", task.ID), zap.Error(completeErr))
		}
		return true
	}

	w.log.Error("task failed",
		zap.String("id", task.ID),
		zap.String("type", task.Type),
		zap.Int("attempt", task.Attempts),
		zap.Int("max", maxAttempts),
		zap.Error(err),
	)

	if retryErr := w.store.RetryOrFailTask(ctx, task.ID, err.Error(), maxAttempts); retryErr != nil {
		w.log.Error("retry/fail task", zap.String("id", task.ID), zap.Error(retryErr))
	}

	return true
}

func (w *Worker) dispatch(ctx context.Context, task *store.Task) error {
	switch task.Type {
	case "process_image":
		return w.processImage(ctx, task)
	case "generate_tts":
		return w.generateTTS(ctx, task)
	case "process_video":
		return w.processVideo(ctx, task)
	default:
		w.log.Warn("unknown task type", zap.String("type", task.Type))
		return nil // mark completed, don't retry unknown types
	}
}
