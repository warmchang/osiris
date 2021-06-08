package activator

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/dailymotion-oss/osiris/pkg/healthz"
	k8s "github.com/dailymotion-oss/osiris/pkg/kubernetes"
)

type Activator interface {
	Run(ctx context.Context)
}

type activator struct {
	kubeClient           kubernetes.Interface
	servicesInformer     cache.SharedIndexInformer
	nodeInformer         cache.SharedIndexInformer
	deploymentsInformer  cache.SharedIndexInformer
	statefulSetsInformer cache.SharedIndexInformer
	services             map[string]*corev1.Service
	deployments          map[string]*appsv1.Deployment
	statefulSets         map[string]*appsv1.StatefulSet
	nodeAddresses        map[string]struct{}
	appsByHost           map[string]*app
	indicesLock          sync.RWMutex
	appActivations       map[string]*appActivation
	appActivationsLock   sync.Mutex
	appActivationTimeout time.Duration
	srv                  *http.Server
	internalSrv          *http.Server
}

func NewActivator(cfg Config, kubeClient kubernetes.Interface) Activator {
	const (
		port         = 5000
		internalPort = 5002
	)
	var (
		mux         = http.NewServeMux()
		internalMux = http.NewServeMux()
	)
	a := &activator{
		kubeClient: kubeClient,
		servicesInformer: k8s.ServicesIndexInformer(
			kubeClient,
			metav1.NamespaceAll,
			nil,
			nil,
			cfg.ResyncInterval,
		),
		nodeInformer: k8s.NodesIndexInformer(
			kubeClient,
			metav1.NamespaceAll,
			nil,
			nil,
			cfg.ResyncInterval,
		),
		deploymentsInformer: k8s.DeploymentsIndexInformer(
			kubeClient,
			metav1.NamespaceAll,
			nil,
			nil,
			cfg.ResyncInterval,
		),
		statefulSetsInformer: k8s.StatefulSetsIndexInformer(
			kubeClient,
			metav1.NamespaceAll,
			nil,
			nil,
			cfg.ResyncInterval,
		),
		services:      map[string]*corev1.Service{},
		deployments:   map[string]*appsv1.Deployment{},
		statefulSets:  map[string]*appsv1.StatefulSet{},
		nodeAddresses: map[string]struct{}{},
		srv: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
		internalSrv: &http.Server{
			Addr:    fmt.Sprintf(":%d", internalPort),
			Handler: internalMux,
		},
		appsByHost:           map[string]*app{},
		appActivations:       map[string]*appActivation{},
		appActivationTimeout: 5 * time.Minute,
	}
	a.servicesInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: a.syncService,
		UpdateFunc: func(_, newObj interface{}) {
			a.syncService(newObj)
		},
		DeleteFunc: a.syncDeletedService,
	})
	a.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: a.syncNode,
		UpdateFunc: func(_, newObj interface{}) {
			a.syncNode(newObj)
		},
		DeleteFunc: a.syncDeletedNode,
	})
	a.deploymentsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: a.syncDeployment,
		UpdateFunc: func(_, newObj interface{}) {
			a.syncDeployment(newObj)
		},
		DeleteFunc: a.syncDeletedDeployment,
	})
	a.statefulSetsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: a.syncStatefulSet,
		UpdateFunc: func(_, newObj interface{}) {
			a.syncStatefulSet(newObj)
		},
		DeleteFunc: a.syncDeletedStatefulSet,
	})
	mux.HandleFunc("/", a.handleRequest)
	internalMux.HandleFunc("/", a.printInternalIndicesState)
	internalMux.HandleFunc("/services", a.printInternalServicesState)
	return a
}

func (a *activator) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		<-ctx.Done()
		glog.Infof("Activator is shutting down")
	}()
	glog.Infof("Activator is started")
	go func() {
		a.servicesInformer.Run(ctx.Done())
		cancel()
	}()
	go func() {
		a.nodeInformer.Run(ctx.Done())
		cancel()
	}()
	go func() {
		a.deploymentsInformer.Run(ctx.Done())
		cancel()
	}()
	go func() {
		a.statefulSetsInformer.Run(ctx.Done())
		cancel()
	}()
	go func() {
		glog.Infof(
			"Activator server is listening on %s, proxying all deactivated, Osiris-enabled applications",
			a.srv.Addr,
		)
		if err := a.runServer(ctx, a.srv); err != nil {
			glog.Errorf("Server error: %s", err)
			cancel()
		}
	}()
	go func() {
		glog.Infof("Activator internal server is listening on %s", a.internalSrv.Addr)
		if err := a.runServer(ctx, a.internalSrv); err != nil {
			glog.Errorf("Server error: %s", err)
			cancel()
		}
	}()
	healthz.RunServer(ctx, 5001)
	cancel()
}

func (a *activator) syncService(obj interface{}) {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		return
	}
	svcKey := getKey(svc.Namespace, "Service", svc.Name)
	a.indicesLock.Lock()
	defer a.indicesLock.Unlock()
	if k8s.ServiceIsEligibleForEndpointsManagement(svc.Annotations) {
		a.services[svcKey] = svc
	} else {
		delete(a.services, svcKey)
	}
	a.updateIndex()
}

func (a *activator) syncDeletedService(obj interface{}) {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		return
	}
	svcKey := getKey(svc.Namespace, "Service", svc.Name)
	a.indicesLock.Lock()
	defer a.indicesLock.Unlock()
	delete(a.services, svcKey)
	a.updateIndex()
}

func (a *activator) syncNode(obj interface{}) {
	node, ok := obj.(*corev1.Node)
	if !ok {
		return
	}
	a.indicesLock.Lock()
	defer a.indicesLock.Unlock()
	for _, nodeAddress := range node.Status.Addresses {
		a.nodeAddresses[nodeAddress.Address] = struct{}{}
	}
	a.updateIndex()
}

func (a *activator) syncDeletedNode(obj interface{}) {
	node, ok := obj.(*corev1.Node)
	if !ok {
		return
	}
	a.indicesLock.Lock()
	defer a.indicesLock.Unlock()
	for _, nodeAddress := range node.Status.Addresses {
		delete(a.nodeAddresses, nodeAddress.Address)
	}
	a.updateIndex()
}

func (a *activator) syncDeployment(obj interface{}) {
	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		return
	}
	deploymentKey := getKey(deployment.Namespace, appKindDeployment, deployment.Name)
	a.indicesLock.Lock()
	defer a.indicesLock.Unlock()
	a.deployments[deploymentKey] = deployment
	a.updateIndex()
}

func (a *activator) syncDeletedDeployment(obj interface{}) {
	a.indicesLock.Lock()
	defer a.indicesLock.Unlock()
	deployment := obj.(*appsv1.Deployment)
	deploymentKey := getKey(deployment.Namespace, appKindDeployment, deployment.Name)
	delete(a.deployments, deploymentKey)
	a.updateIndex()
}

func (a *activator) syncStatefulSet(obj interface{}) {
	statefulSet, ok := obj.(*appsv1.StatefulSet)
	if !ok {
		return
	}
	statefulSetKey := getKey(statefulSet.Namespace, appKindStatefulSet, statefulSet.Name)
	a.indicesLock.Lock()
	defer a.indicesLock.Unlock()
	a.statefulSets[statefulSetKey] = statefulSet
	a.updateIndex()
}

func (a *activator) syncDeletedStatefulSet(obj interface{}) {
	statefulSet, ok := obj.(*appsv1.StatefulSet)
	if !ok {
		return
	}
	statefulSetKey := getKey(statefulSet.Namespace, appKindStatefulSet, statefulSet.Name)
	a.indicesLock.Lock()
	defer a.indicesLock.Unlock()
	delete(a.statefulSets, statefulSetKey)
	a.updateIndex()
}
