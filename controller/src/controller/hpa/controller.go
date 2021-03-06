package hpa

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"minik8s/apiObject"
	"minik8s/apiObject/types"
	"minik8s/controller/src/cache"
	"minik8s/entity"
	"minik8s/listwatch"
	"minik8s/util/logger"
	"minik8s/util/topicutil"
)

const DefaultScaleInterval = 15

var bgCtx = context.Background()
var logManager = logger.Log("HPA manager")

type controller struct {
	cacheManager cache.Manager
	workers      map[types.UID]Worker
	cancelFuncs  map[types.UID]context.CancelFunc
}

func (c *controller) AddHpa(hpa *apiObject.HorizontalPodAutoscaler) {
	UID := hpa.UID()
	_, cancelExists := c.cancelFuncs[UID]
	_, workerExists := c.workers[UID]
	if cancelExists && workerExists {
		return
	}
	logManager("Add hpa: %s_%s", hpa.Name(), UID)
	ctx, cancelFunc := context.WithCancel(bgCtx)
	worker := NewWorker(ctx, hpa, c.cacheManager)
	c.cancelFuncs[UID] = cancelFunc
	c.workers[UID] = worker
	go worker.Run()
}

func (c *controller) DeleteHpa(hpa *apiObject.HorizontalPodAutoscaler) {
	logManager("Delete hpa: %s_%s", hpa.Name(), hpa.UID())
	UID := hpa.UID()
	if cancel, exists := c.cancelFuncs[UID]; exists {
		delete(c.cancelFuncs, UID)
		delete(c.workers, UID)
		cancel()
	}
}

func (c *controller) UpdateHpa(hpa *apiObject.HorizontalPodAutoscaler) {
	logManager("Update hpa: %s_%s", hpa.Name(), hpa.UID())
	UID := hpa.UID()
	if worker, exists := c.workers[UID]; exists {
		worker.SetTarget(hpa)
	}
}

func (c *controller) parseHPAUpdate(msg *redis.Message) {
	hpaUpdate := &entity.HPAUpdate{}
	err := json.Unmarshal([]byte(msg.Payload), hpaUpdate)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	hpa := &hpaUpdate.Target

	switch hpaUpdate.Action {
	case entity.CreateAction:
		c.AddHpa(hpa)
	case entity.UpdateAction:
		c.UpdateHpa(hpa)
	case entity.DeleteAction:
		c.DeleteHpa(hpa)
	}
}

func (c *controller) Run() {
	topic := topicutil.HPAUpdateTopic()
	listwatch.Watch(topic, c.parseHPAUpdate)
}

type Controller interface {
	Run()
	AddHpa(hpa *apiObject.HorizontalPodAutoscaler)
	UpdateHpa(hpa *apiObject.HorizontalPodAutoscaler)
	DeleteHpa(hpa *apiObject.HorizontalPodAutoscaler)
}

func NewController(cacheManager cache.Manager) Controller {
	return &controller{
		cacheManager: cacheManager,
		workers:      make(map[types.UID]Worker),
		cancelFuncs:  make(map[types.UID]context.CancelFunc),
	}
}
