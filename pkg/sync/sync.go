//
// Copyright (c) 2019-2020 Red Hat, Inc.
// This program and the accompanying materials are made
// available under the terms of the Eclipse Public License 2.0
// which is available at https://www.eclipse.org/legal/epl-2.0/
//
// SPDX-License-Identifier: EPL-2.0
//
// Contributors:
//   Red Hat, Inc. - initial API and implementation
//

package sync

import (
	"fmt"
	"context"
	
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	log = ctrl.Log.WithName("sync")
)

type Syncer struct {
	client.Client
	Scheme *runtime.Scheme
}

type SyncContext struct {
	Client client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Owner  metav1.Object
}

// Sync syncs the blueprint to the cluster in a generic (as much as Go allows) manner.
// Returns true if the object was created or updated, false if there was no change detected.
func (s *Syncer) Sync(ctx context.Context, owner metav1.Object, blueprint metav1.Object, diffOpts cmp.Option) (bool, error) {
	blueprintObject, ok := blueprint.(runtime.Object)
	if !ok {
		return false, fmt.Errorf("object %T is not a runtime.Object. Cannot sync it", blueprint)
	}

	key := client.ObjectKey{Name: blueprint.GetName(), Namespace: blueprint.GetNamespace()}

	actual := blueprintObject.DeepCopyObject()

	if getErr := s.Get(context.TODO(), key, actual); getErr != nil {
		if statusErr, ok := getErr.(*errors.StatusError); !ok || statusErr.Status().Reason != metav1.StatusReasonNotFound {
			return false, getErr
		}
		actual = nil
	}

	if actual == nil {
		_, err := s.create(ctx, owner, key, blueprint)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return s.update(ctx, owner, actual, blueprint, diffOpts)
}

func (s *Syncer) create(ctx context.Context, owner metav1.Object, key client.ObjectKey, blueprint metav1.Object) (runtime.Object, error) {
	blueprintObject, ok := blueprint.(runtime.Object)
	kind := blueprintObject.GetObjectKind().GroupVersionKind().Kind
	if !ok {
		return nil, fmt.Errorf("object %T is not a runtime.Object. Cannot sync it", blueprint)
	}

	actual := blueprintObject.DeepCopyObject()

	log.Info("Creating a new object", "kind", kind, "name", blueprint.GetName(), "namespace", blueprint.GetNamespace())
	obj, err := s.setOwnerReferenceAndConvertToRuntime(owner, blueprint)
	if err != nil {
		return nil, err
	}

	err = s.Create(ctx, obj)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, err
		}

		// ok, we got an already-exists error. So let's try to load the object into "actual".
		// if we fail this retry for whatever reason, just give up rather than retrying this in a loop...
		// the reconciliation loop will lead us here again in the next round.
		if getErr := s.Get(ctx, key, actual); getErr != nil {
			return nil, getErr
		}
	}

	return actual, nil
}

func (s *Syncer) update(ctx context.Context, owner metav1.Object, actual runtime.Object, blueprint metav1.Object, diffOpts cmp.Option) (bool, error) {
	actualMeta := actual.(metav1.Object)

	diff := cmp.Diff(actual, blueprint, diffOpts)
	if len(diff) > 0 {
		kind := actual.GetObjectKind().GroupVersionKind().Kind
		log.Info("Updating existing object", "kind", kind, "name", actualMeta.GetName(), "namespace", actualMeta.GetNamespace())

		if isUpdateUsingDeleteCreate(actual.GetObjectKind().GroupVersionKind().Kind) {
			err := s.Delete(ctx, actual)
			if err != nil {
				return false, err
			}

			obj, err := s.setOwnerReferenceAndConvertToRuntime(owner, blueprint)
			if err != nil {
				return false, err
			}

			err = s.Create(ctx, obj)
			if err != nil {
				return false, err
			}
		} else {
			obj, err := s.setOwnerReferenceAndConvertToRuntime(owner, blueprint)
			if err != nil {
				return false, err
			}

			// to be able to update, we need to set the resource version of the object that we know of
			obj.(metav1.Object).SetResourceVersion(actualMeta.GetResourceVersion())

			err = s.Update(ctx, obj)
			if err != nil {
				return false, err
			}
		}

		return true, nil
	}
	return false, nil
}

func isUpdateUsingDeleteCreate(kind string) bool {
	// not sure why the old che operator needed this, but keeping the original code around just in case we find the reason :)
	// return "Service" == kind || "Ingress" == kind || "Route" == kind
	return false
}

func (s *Syncer) setOwnerReferenceAndConvertToRuntime(owner metav1.Object, obj metav1.Object) (runtime.Object, error) {
	robj, ok := obj.(runtime.Object)
	if !ok {
		return nil, fmt.Errorf("object %T is not a runtime.Object. Cannot sync it", obj)
	}

	if owner == nil {
		return robj, nil
	}

	err := controllerutil.SetControllerReference(owner, obj, s.Scheme)
	if err != nil {
		return nil, err
	}

	return robj, nil
}