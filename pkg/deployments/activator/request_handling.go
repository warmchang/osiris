package activator

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
)

func (a *activator) handleRequest(
	w http.ResponseWriter,
	r *http.Request,
) {
	defer r.Body.Close()

	glog.Infof(
		"Request received for host %s with URI %s",
		r.Host,
		r.RequestURI,
	)

	a.indicesLock.RLock()
	app, ok := a.appsByHost[r.Host]
	a.indicesLock.RUnlock()
	if !ok {
		glog.Infof("No deployment/statefulset found for host %s", r.Host)
		a.returnError(w, http.StatusNotFound)
		return
	}

	glog.Infof(
		"%s %s in namespace %s may require activation",
		app.Kind,
		app.Name,
		app.Namespace,
	)

	// Are we already activating the deployment/statefulset in question?
	var err error
	appKey := getKey(app.Namespace, app.Kind, app.Name)
	appActivation, ok := a.appActivations[appKey]
	if ok {
		glog.Infof(
			"Found activation in-progress for %s %s in namespace %s",
			app.Kind,
			app.Name,
			app.Namespace,
		)
	} else {
		func() {
			a.appActivationsLock.Lock()
			defer a.appActivationsLock.Unlock()
			// Some other goroutine could have initiated activation of this deployment/statefulset
			// while we were waiting for the lock. Now that we have the lock, do we
			// still need to do this?
			appActivation, ok = a.appActivations[appKey]
			if ok {
				glog.Infof(
					"Found activation in-progress for %s %s in namespace %s",
					app.Kind,
					app.Name,
					app.Namespace,
				)
				return
			}
			glog.Infof(
				"Found NO activation in-progress for %s %s in namespace %s",
				app.Kind,
				app.Name,
				app.Namespace,
			)
			// Initiate activation (or discover that it may already have been started
			// by another activator process)
			ctx, cancelFunc := context.WithTimeout(context.Background(), a.appActivationTimeout)
			defer cancelFunc()
			appActivation, err = a.activate(ctx, app)
			if err != nil {
				glog.Errorf(
					"%s activation for %s in namespace %s failed: %s",
					app.Kind,
					app.Name,
					app.Namespace,
					err,
				)
				return
			}
			// Add it to the index of in-flight activation
			a.appActivations[appKey] = appActivation
			// But remove it from that index when it's complete
			go func() {
				deleteActivation := func() {
					a.appActivationsLock.Lock()
					defer a.appActivationsLock.Unlock()
					delete(a.appActivations, appKey)
				}
				select {
				case <-appActivation.successCh:
					deleteActivation()
				case <-appActivation.timeoutCh:
					deleteActivation()
				}
			}()
		}()
		if err != nil {
			glog.Errorf(
				"Error activating %s %s in namespace %s: %s",
				app.Kind,
				app.Name,
				app.Namespace,
				err,
			)
			a.returnError(w, http.StatusServiceUnavailable)
			return
		}
	}

	// Regardless of whether we just started an activation or found one already in
	// progress, we need to wait for that activation to be completed... or fail...
	// or time out.
	select {
	case <-appActivation.successCh:
		glog.Infof("Passing request on to: %s", app.TargetURL)
		app.proxyRequestHandler.ServeHTTP(w, r)
	case <-appActivation.timeoutCh:
		a.returnError(w, http.StatusServiceUnavailable)
	}
}

func (a *activator) printInternalIndicesState(
	w http.ResponseWriter,
	r *http.Request,
) {
	a.indicesLock.RLock()
	appsByHost := make(map[string]*app, len(a.appsByHost))
	for host, app := range a.appsByHost {
		appsByHost[host] = app
	}
	a.indicesLock.RUnlock()

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	err := enc.Encode(appsByHost)
	if err != nil {
		glog.Errorf("Error encoding appsByHost in json: %s", err)
	}
}

func (a *activator) printInternalServicesState(
	w http.ResponseWriter,
	r *http.Request,
) {
	a.indicesLock.RLock()
	services := make(map[string]*corev1.Service, len(a.services))
	for name, svc := range a.services {
		services[name] = svc
	}
	a.indicesLock.RUnlock()

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	err := enc.Encode(services)
	if err != nil {
		glog.Errorf("Error encoding services in json: %s", err)
	}
}

func (a *activator) returnError(w http.ResponseWriter, statusCode int) {
	w.WriteHeader(statusCode)
	if _, err := w.Write([]byte{}); err != nil {
		glog.Errorf("Error writing response body: %s", err)
	}
}
