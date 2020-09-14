package activator

import (
	"net/http"

	"github.com/golang/glog"
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
		app.kind,
		app.name,
		app.namespace,
	)

	// Are we already activating the deployment/statefulset in question?
	var err error
	appKey := getKey(app.namespace, app.kind, app.name)
	appActivation, ok := a.appActivations[appKey]
	if ok {
		glog.Infof(
			"Found activation in-progress for %s %s in namespace %s",
			app.kind,
			app.name,
			app.namespace,
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
					app.kind,
					app.name,
					app.namespace,
				)
				return
			}
			glog.Infof(
				"Found NO activation in-progress for %s %s in namespace %s",
				app.kind,
				app.name,
				app.namespace,
			)
			// Initiate activation (or discover that it may already have been started
			// by another activator process)
			switch app.kind {
			case appKindDeployment:
				appActivation, err = a.activateDeployment(r.Context(), app)
			case appKindStatefulSet:
				appActivation, err = a.activateStatefulSet(r.Context(), app)
			default:
				glog.Errorf("unvalid app kind %s", app.kind)
				return
			}
			if err != nil {
				glog.Errorf(
					"%s activation for %s in namespace %s failed: %s",
					app.kind,
					app.name,
					app.namespace,
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
				app.kind,
				app.name,
				app.namespace,
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
		glog.Infof("Passing request on to: %s", app.targetURL)
		app.proxyRequestHandler.ServeHTTP(w, r)
	case <-appActivation.timeoutCh:
		a.returnError(w, http.StatusServiceUnavailable)
	}
}

func (a *activator) returnError(w http.ResponseWriter, statusCode int) {
	w.WriteHeader(statusCode)
	if _, err := w.Write([]byte{}); err != nil {
		glog.Errorf("Error writing response body: %s", err)
	}
}
