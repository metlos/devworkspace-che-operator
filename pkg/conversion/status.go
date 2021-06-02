package conversion

import (
	"encoding/json"
	"fmt"
	"reflect"

	v1 "github.com/eclipse-che/che-operator/pkg/apis/org/v1"
	"github.com/eclipse-che/che-operator/pkg/apis/org/v2alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ client.Patch = (*v1StatusPatch)(nil)

type v1StatusPatch struct {
	status v2alpha1.CheClusterStatus
}

// StatusPatch provides a patch object that updates the devworkspace portion of the CheCluster v1 status object.
// This is so that we can operate solely on CheCluster v2alpha1 objects in memory yet still persist correct data
// to the cluster that only stores v1. Generally, we don't want to update the v1 portions of the status and only
// deal with the portions relevant to v2. There might be occassions though where we do want to update both. This
// interface is generic enough to support both of the situations.
func StatusPatch(obj *v2alpha1.CheCluster) client.Patch {
	return &v1StatusPatch{status: obj.Status}
}

func (s *v1StatusPatch) Type() types.PatchType {
	return types.JSONPatchType
}

func (s *v1StatusPatch) Data(obj runtime.Object) ([]byte, error) {
	_, ok := obj.(*v1.CheCluster)
	if !ok {
		return nil, fmt.Errorf("only CheCluster v1 objects are supported but `%v` was provided", reflect.TypeOf(obj).String())
	}

	status, err := json.Marshal(s.status)
	if err != nil {
		return nil, err
	}

	data := `[{"op": "replace", "path": "/status/devworkspaceStatus", "value": ` + string(status) + "}]"

	return []byte(data), nil
}
