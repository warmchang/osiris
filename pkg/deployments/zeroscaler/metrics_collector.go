package zeroscaler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8s_types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	k8s "github.com/dailymotion-oss/osiris/pkg/kubernetes"
)

type metricsCollectorConfig struct {
	appKind              string
	appName              string
	appNamespace         string
	selector             labels.Selector
	metricsCheckInterval time.Duration
	scraperConfig        metricsScraperConfig
}

type metricsCollector struct {
	config       metricsCollectorConfig
	scraper      metricsScraper
	kubeClient   kubernetes.Interface
	podsInformer cache.SharedIndexInformer
	appPods      map[string]*corev1.Pod
	appPodsLock  sync.Mutex
	cancelFunc   func()
}

func newMetricsCollector(
	kubeClient kubernetes.Interface,
	config metricsCollectorConfig,
) (*metricsCollector, error) {
	s, err := newMetricsScraper(config.scraperConfig)
	if err != nil {
		return nil, err
	}
	m := &metricsCollector{
		config:     config,
		scraper:    s,
		kubeClient: kubeClient,
		podsInformer: k8s.PodsIndexInformer(
			kubeClient,
			config.appNamespace,
			nil,
			config.selector,
		),
		appPods: map[string]*corev1.Pod{},
	}
	m.podsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: m.syncAppPod,
		UpdateFunc: func(_, newObj interface{}) {
			m.syncAppPod(newObj)
		},
		DeleteFunc: m.syncDeletedAppPod,
	})
	return m, nil
}

func (m *metricsCollector) run(ctx context.Context) {
	ctx, m.cancelFunc = context.WithCancel(ctx)
	defer m.cancelFunc()
	go func() {
		<-ctx.Done()
		glog.Infof(
			"Stopping metrics collection for %s %s in namespace %s",
			m.config.appKind,
			m.config.appName,
			m.config.appNamespace,
		)
	}()
	glog.Infof(
		"Starting metrics collection for %s %s in namespace %s",
		m.config.appKind,
		m.config.appName,
		m.config.appNamespace,
	)
	go m.podsInformer.Run(ctx.Done())
	// When this exits, the cancel func will stop the informer
	m.collectMetrics(ctx)
}

func (m *metricsCollector) stop() {
	m.cancelFunc()
}

func (m *metricsCollector) syncAppPod(obj interface{}) {
	m.appPodsLock.Lock()
	defer m.appPodsLock.Unlock()
	pod := obj.(*corev1.Pod)
	m.appPods[pod.Name] = pod
}

func (m *metricsCollector) syncDeletedAppPod(obj interface{}) {
	m.appPodsLock.Lock()
	defer m.appPodsLock.Unlock()
	pod := obj.(*corev1.Pod)
	delete(m.appPods, pod.Name)
}

func (m *metricsCollector) collectMetrics(ctx context.Context) {
	requestCountsByProxy := map[string]uint64{}
	var lastTotalRequestCount uint64
	ticker := time.NewTicker(m.config.metricsCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.appPodsLock.Lock()
			var mustNotDecide bool
			var scrapeWG sync.WaitGroup
			// An aggressively small timeout. We make the decision fast or not at
			// all.
			timer := time.NewTimer(3 * time.Second)
			for _, pod := range m.appPods {
				scrapeWG.Add(1)
				go func(pod *corev1.Pod) {
					defer scrapeWG.Done()
					// Get the results
					prc := m.scraper.Scrap(pod)
					if prc == nil {
						mustNotDecide = true
					} else {
						requestCountsByProxy[prc.ProxyID] = prc.RequestCount
					}
				}(pod)
			}
			m.appPodsLock.Unlock()
			scrapeWG.Wait()
			var totalRequestCount uint64
			for _, requestCount := range requestCountsByProxy {
				totalRequestCount += requestCount
			}
			select {
			case <-timer.C:
				mustNotDecide = true
			case <-ctx.Done():
				return
			default:
			}
			timer.Stop()
			if !mustNotDecide && totalRequestCount == lastTotalRequestCount {
				m.scaleToZero(context.TODO())
			}
			lastTotalRequestCount = totalRequestCount
		case <-ctx.Done():
			return
		}
	}
}

func (m *metricsCollector) scaleToZero(ctx context.Context) {
	glog.Infof(
		"Scale to zero starting for %s %s in namespace %s",
		m.config.appKind,
		m.config.appName,
		m.config.appNamespace,
	)

	patches := []k8s.PatchOperation{{
		Op:    "replace",
		Path:  "/spec/replicas",
		Value: 0,
	}}
	patchesBytes, _ := json.Marshal(patches)
	var err error
	switch m.config.appKind {
	case "Deployment":
		_, err = m.kubeClient.AppsV1().Deployments(m.config.appNamespace).Patch(
			ctx,
			m.config.appName,
			k8s_types.JSONPatchType,
			patchesBytes,
			metav1.PatchOptions{},
		)
	case "StatefulSet":
		_, err = m.kubeClient.AppsV1().StatefulSets(m.config.appNamespace).Patch(
			ctx,
			m.config.appName,
			k8s_types.JSONPatchType,
			patchesBytes,
			metav1.PatchOptions{},
		)
	default:
		err = fmt.Errorf("unknown kind '%s'", m.config.appKind)
	}
	if err != nil {
		glog.Errorf(
			"Error scaling %s %s in namespace %s to zero: %s",
			m.config.appKind,
			m.config.appName,
			m.config.appNamespace,
			err,
		)
		return
	}

	glog.Infof(
		"Scaled %s %s in namespace %s to zero",
		m.config.appKind,
		m.config.appName,
		m.config.appNamespace,
	)
}
