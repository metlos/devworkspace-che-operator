package conversion

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/devfile/devworkspace-operator/pkg/infrastructure"
	"github.com/eclipse-che/che-operator/pkg/apis"
	v1 "github.com/eclipse-che/che-operator/pkg/apis/org/v1"
	"github.com/eclipse-che/che-operator/pkg/apis/org/v2alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestStatusPatchFormat(t *testing.T) {
	v2Obj := v2alpha1.CheCluster{}
	v2Obj.Status = v2alpha1.CheClusterStatus{
		GatewayPhase:        v2alpha1.GatewayPhaseInitializing,
		Phase:               v2alpha1.ClusterPhasePendingDeletion,
		GatewayHost:         "over.the.rainbow",
		Reason:              "grave",
		Message:             "serious",
		WorkspaceBaseDomain: "down.on.earth",
	}

	patch := StatusPatch(&v2Obj)

	if patch.Type() != types.JSONPatchType {
		t.Errorf("Unexpected patch type: %v", patch.Type())
	}

	v1Obj := v1.CheCluster{}

	patchData, err := patch.Data(&v1Obj)
	if err != nil {
		t.Error(err)
	}

	unmarshalled := []map[string]interface{}{}

	if err = json.Unmarshal(patchData, &unmarshalled); err != nil {
		t.Error(err)
	}

	if unmarshalled[0]["op"] != "replace" {
		t.Errorf("Unexpected patch operation: %v", unmarshalled[0]["op"])
	}

	if unmarshalled[0]["path"] != "/status/devworkspaceStatus" {
		t.Errorf("Unexpected patch path: %v", unmarshalled[0]["path"])
	}

	origData, err := json.Marshal(v2Obj.Status)
	if err != nil {
		t.Error(err)
	}
	origStatus := map[string]interface{}{}
	if err = json.Unmarshal(origData, &origStatus); err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(origStatus, unmarshalled[0]["value"]) {
		t.Errorf("Unexpected patch value: %v", unmarshalled[0]["value"])
	}
}

func TestStatusPatch(t *testing.T) {
	infrastructure.InitializeForTesting(infrastructure.Kubernetes)

	scheme := runtime.NewScheme()
	utilruntime.Must(apis.AddToScheme(scheme))

	v1Obj := v1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "che",
			Namespace: "che",
		},
		Status: v1.CheClusterStatus{
			CheURL: "over.the.rainbow",
		},
	}

	ctx := context.TODO()
	cl := fake.NewFakeClientWithScheme(scheme, &v1Obj)

	v2Obj := v2alpha1.CheCluster{}
	v2Obj.Status = v2alpha1.CheClusterStatus{
		GatewayPhase:        v2alpha1.GatewayPhaseInitializing,
		Phase:               v2alpha1.ClusterPhasePendingDeletion,
		GatewayHost:         "over.the.mountaintops",
		Reason:              "grave",
		Message:             "serious",
		WorkspaceBaseDomain: "down.on.earth",
	}

	patch := StatusPatch(&v2Obj)

	if err := cl.Status().Patch(ctx, &v1Obj, patch); err != nil {
		t.Error(err)
	}

	fromCluster := v1.CheCluster{}
	if err := cl.Get(ctx, client.ObjectKey{Name: v1Obj.Name, Namespace: v1Obj.Namespace}, &fromCluster); err != nil {
		t.Error(err)
	}

	if fromCluster.Status.CheURL != "over.the.rainbow" {
		t.Errorf("The CheURL in status shouldn't have changed but was: %v", fromCluster.Status.CheURL)
	}

	if fromCluster.Status.DevworkspaceStatus.GatewayPhase != v2alpha1.GatewayPhaseInitializing {
		t.Errorf("The `GatewayPhase` of the devworkspace status should have been set to `v2alpha1.GatewayPhaseInitializing` but was: %v", fromCluster.Status.DevworkspaceStatus.GatewayPhase)
	}

	if fromCluster.Status.DevworkspaceStatus.Phase != v2alpha1.ClusterPhasePendingDeletion {
		t.Errorf("The `Phase` of the devworkspace status should have been set to `v2alpha1.ClusterPhasePendingDeletion` but was: %v", fromCluster.Status.DevworkspaceStatus.Phase)
	}

	if fromCluster.Status.DevworkspaceStatus.GatewayHost != "over.the.mountaintops" {
		t.Errorf("The `GatewayHost` of the devworkspace status should have been set to `over.the.mountaintops` but was: %v", fromCluster.Status.DevworkspaceStatus.GatewayHost)
	}

	if fromCluster.Status.DevworkspaceStatus.Reason != "grave" {
		t.Errorf("The `Reason` of the devworkspace status should have been set to `grave` but was: %v", fromCluster.Status.DevworkspaceStatus.Reason)
	}

	if fromCluster.Status.DevworkspaceStatus.Message != "serious" {
		t.Errorf("The `Message` of the devworkspace status should have been set to `serious` but was: %v", fromCluster.Status.DevworkspaceStatus.Message)
	}

	if fromCluster.Status.DevworkspaceStatus.WorkspaceBaseDomain != "down.on.earth" {
		t.Errorf("The `WorkspaceBaseDomain` of the devworkspace status should have been set to `down.on.earth` but was: %v", fromCluster.Status.DevworkspaceStatus.WorkspaceBaseDomain)
	}

}
