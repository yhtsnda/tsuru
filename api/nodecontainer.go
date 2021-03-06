// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ajg/form"
	"github.com/pkg/errors"
	"github.com/tsuru/tsuru/auth"
	tsuruErrors "github.com/tsuru/tsuru/errors"
	"github.com/tsuru/tsuru/event"
	tsuruIo "github.com/tsuru/tsuru/io"
	"github.com/tsuru/tsuru/permission"
	"github.com/tsuru/tsuru/provision"
	"github.com/tsuru/tsuru/provision/nodecontainer"
)

// title: remove node container list
// path: /docker/nodecontainers
// method: GET
// produce: application/json
// responses:
//   200: Ok
//   401: Unauthorized
func nodeContainerList(w http.ResponseWriter, r *http.Request, t auth.Token) error {
	pools, err := permission.ListContextValues(t, permission.PermNodecontainerRead, true)
	if err != nil {
		return err
	}
	lst, err := nodecontainer.AllNodeContainers()
	if err != nil {
		return err
	}
	if pools != nil {
		poolMap := map[string]struct{}{}
		for _, p := range pools {
			poolMap[p] = struct{}{}
		}
		for i, entry := range lst {
			for poolName := range entry.ConfigPools {
				if poolName == "" {
					continue
				}
				if _, ok := poolMap[poolName]; !ok {
					delete(entry.ConfigPools, poolName)
				}
			}
			lst[i] = entry
		}
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(lst)
}

// title: node container create
// path: /docker/nodecontainers
// method: POST
// consume: application/x-www-form-urlencoded
// responses:
//   200: Ok
//   400: Invald data
//   401: Unauthorized
func nodeContainerCreate(w http.ResponseWriter, r *http.Request, t auth.Token) (err error) {
	err = r.ParseForm()
	if err != nil {
		return err
	}
	dec := form.NewDecoder(nil)
	dec.IgnoreUnknownKeys(true)
	dec.IgnoreCase(true)
	var config nodecontainer.NodeContainerConfig
	err = dec.DecodeValues(&config, r.Form)
	if err != nil {
		return err
	}
	poolName := r.FormValue("pool")
	var ctxs []permission.PermissionContext
	if poolName != "" {
		ctxs = append(ctxs, permission.Context(permission.CtxPool, poolName))
	}
	if !permission.Check(t, permission.PermNodecontainerCreate, ctxs...) {
		return permission.ErrUnauthorized
	}
	evt, err := event.New(&event.Opts{
		Target:     event.Target{Type: event.TargetTypeNodeContainer, Value: config.Name},
		Kind:       permission.PermNodecontainerCreate,
		Owner:      t,
		CustomData: event.FormToCustomData(r.Form),
		Allowed:    event.Allowed(permission.PermPoolReadEvents, ctxs...),
	})
	if err != nil {
		return err
	}
	defer func() { evt.Done(err) }()
	err = nodecontainer.AddNewContainer(poolName, &config)
	if err != nil {
		if _, ok := err.(nodecontainer.ValidationErr); ok {
			return &tsuruErrors.HTTP{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			}
		}
		return err
	}
	return nil
}

// title: node container info
// path: /docker/nodecontainers/{name}
// method: GET
// produce: application/json
// responses:
//   200: Ok
//   401: Unauthorized
//   404: Not found
func nodeContainerInfo(w http.ResponseWriter, r *http.Request, t auth.Token) error {
	pools, err := permission.ListContextValues(t, permission.PermNodecontainerRead, true)
	if err != nil {
		return err
	}
	name := r.URL.Query().Get(":name")
	configMap, err := nodecontainer.LoadNodeContainersForPools(name)
	if err != nil {
		if err == nodecontainer.ErrNodeContainerNotFound {
			return &tsuruErrors.HTTP{
				Code:    http.StatusNotFound,
				Message: err.Error(),
			}
		}
		return err
	}
	if pools != nil {
		poolMap := map[string]struct{}{}
		for _, p := range pools {
			poolMap[p] = struct{}{}
		}
		for poolName := range configMap {
			if poolName == "" {
				continue
			}
			if _, ok := poolMap[poolName]; !ok {
				delete(configMap, poolName)
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(configMap)
}

// title: node container update
// path: /docker/nodecontainers/{name}
// method: POST
// consume: application/x-www-form-urlencoded
// responses:
//   200: Ok
//   400: Invald data
//   401: Unauthorized
//   404: Not found
func nodeContainerUpdate(w http.ResponseWriter, r *http.Request, t auth.Token) (err error) {
	err = r.ParseForm()
	if err != nil {
		return err
	}
	dec := form.NewDecoder(nil)
	dec.IgnoreUnknownKeys(true)
	dec.IgnoreCase(true)
	var config nodecontainer.NodeContainerConfig
	err = dec.DecodeValues(&config, r.Form)
	if err != nil {
		return err
	}
	poolName := r.FormValue("pool")
	var ctxs []permission.PermissionContext
	if poolName != "" {
		ctxs = append(ctxs, permission.Context(permission.CtxPool, poolName))
	}
	if !permission.Check(t, permission.PermNodecontainerUpdate, ctxs...) {
		return permission.ErrUnauthorized
	}
	evt, err := event.New(&event.Opts{
		Target:     event.Target{Type: event.TargetTypeNodeContainer, Value: config.Name},
		Kind:       permission.PermNodecontainerUpdate,
		Owner:      t,
		CustomData: event.FormToCustomData(r.Form),
		Allowed:    event.Allowed(permission.PermPoolReadEvents, ctxs...),
	})
	if err != nil {
		return err
	}
	defer func() { evt.Done(err) }()
	config.Name = r.URL.Query().Get(":name")
	err = nodecontainer.UpdateContainer(poolName, &config)
	if err != nil {
		if err == nodecontainer.ErrNodeContainerNotFound {
			return &tsuruErrors.HTTP{
				Code:    http.StatusNotFound,
				Message: err.Error(),
			}
		}
		if _, ok := err.(nodecontainer.ValidationErr); ok {
			return &tsuruErrors.HTTP{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			}
		}
		return err
	}
	return nil
}

// title: remove node container
// path: /docker/nodecontainers/{name}
// method: DELETE
// responses:
//   200: Ok
//   401: Unauthorized
//   404: Not found
func nodeContainerDelete(w http.ResponseWriter, r *http.Request, t auth.Token) (err error) {
	r.ParseForm()
	name := r.URL.Query().Get(":name")
	poolName := r.URL.Query().Get("pool")
	var ctxs []permission.PermissionContext
	if poolName != "" {
		ctxs = append(ctxs, permission.Context(permission.CtxPool, poolName))
	}
	if !permission.Check(t, permission.PermNodecontainerDelete, ctxs...) {
		return permission.ErrUnauthorized
	}
	evt, err := event.New(&event.Opts{
		Target:     event.Target{Type: event.TargetTypeNodeContainer, Value: name},
		Kind:       permission.PermNodecontainerDelete,
		Owner:      t,
		CustomData: event.FormToCustomData(r.Form),
		Allowed:    event.Allowed(permission.PermPoolReadEvents, ctxs...),
	})
	if err != nil {
		return err
	}
	defer func() { evt.Done(err) }()
	err = nodecontainer.RemoveContainer(poolName, name)
	if err == nodecontainer.ErrNodeContainerNotFound {
		return &tsuruErrors.HTTP{
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("node container %q not found for pool %q", name, poolName),
		}
	}
	return err
}

// title: node container upgrade
// path: /docker/nodecontainers/{name}/upgrade
// method: POST
// consume: application/x-www-form-urlencoded
// produce: application/x-json-stream
// responses:
//   200: Ok
//   400: Invald data
//   401: Unauthorized
//   404: Not found
func nodeContainerUpgrade(w http.ResponseWriter, r *http.Request, t auth.Token) (err error) {
	name := r.URL.Query().Get(":name")
	poolName := r.FormValue("pool")
	var ctxs []permission.PermissionContext
	if poolName != "" {
		ctxs = append(ctxs, permission.Context(permission.CtxPool, poolName))
	}
	if !permission.Check(t, permission.PermNodecontainerUpdateUpgrade, ctxs...) {
		return permission.ErrUnauthorized
	}
	evt, err := event.New(&event.Opts{
		Target:     event.Target{Type: event.TargetTypeNodeContainer, Value: name},
		Kind:       permission.PermNodecontainerUpdateUpgrade,
		Owner:      t,
		CustomData: event.FormToCustomData(r.Form),
		Allowed:    event.Allowed(permission.PermPoolReadEvents, ctxs...),
	})
	if err != nil {
		return err
	}
	defer func() { evt.Done(err) }()
	err = nodecontainer.ResetImage(poolName, name)
	if err != nil {
		if err == nodecontainer.ErrNodeContainerNotFound {
			return &tsuruErrors.HTTP{
				Code:    http.StatusNotFound,
				Message: err.Error(),
			}
		}
		return err
	}
	provs, err := provision.Registry()
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/x-json-stream")
	keepAliveWriter := tsuruIo.NewKeepAliveWriter(w, 15*time.Second, "")
	defer keepAliveWriter.Stop()
	writer := &tsuruIo.SimpleJsonMessageEncoderWriter{Encoder: json.NewEncoder(keepAliveWriter)}
	var allErrors []string
	for _, prov := range provs {
		ncProv, ok := prov.(provision.NodeContainerProvisioner)
		if !ok {
			continue
		}
		err = ncProv.UpgradeNodeContainer(name, poolName, writer)
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}
	if len(allErrors) > 0 {
		return errors.Errorf("multiple errors upgrading nodes: %s", strings.Join(allErrors, "; "))
	}
	return nil
}
