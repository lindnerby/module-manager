package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
)

type Workers interface {
	GetWorkerPoolSize() int
	SetWorkerPoolSize(newSize int)
	StartWorkers(ctx context.Context, jobChan <-chan DeployInfo, handlerFn func(info DeployInfo, logger *logr.Logger) *RequestError)
}

type ManifestWorkers struct {
	Workers
	logger *logr.Logger
	size   int
}

func NewManifestWorkers(logger *logr.Logger) *ManifestWorkers {
	return &ManifestWorkers{
		logger: logger,
		size:   DefaultWorkersCount,
	}
}

func (mw *ManifestWorkers) StartWorkers(ctx context.Context, jobChan <-chan DeployInfo, handlerFn func(info DeployInfo, logger *logr.Logger) *RequestError) {
	for worker := 1; worker <= mw.GetWorkerPoolSize(); worker++ {
		go func(ctx context.Context, id int, deployJob <-chan DeployInfo) {
			mw.logger.Info(fmt.Sprintf("Starting manifest-operator worker with id %d", id))
			for {
				select {
				case deployInfo := <-deployJob:
					mw.logger.Info(fmt.Sprintf("Processing chart with name %s by worker with id %d", deployInfo.ChartName, id))
					deployInfo.RequestErrChan <- handlerFn(deployInfo, mw.logger)
				case <-ctx.Done():
					return
				}
			}
		}(ctx, worker, jobChan)
	}
}

func (mw *ManifestWorkers) GetWorkerPoolSize() int {
	return mw.size
}

func (mw *ManifestWorkers) SetWorkerPoolSize(newSize int) {
	if newSize < 1 {
		mw.size = DefaultWorkersCount
	} else {
		mw.size = newSize
	}
}