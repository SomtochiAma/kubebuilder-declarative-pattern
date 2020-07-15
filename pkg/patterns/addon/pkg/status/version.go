package status

import (
	"context"
	"fmt"

	"github.com/blang/semver"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	addonsv1alpha1 "sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/addon/pkg/apis/v1alpha1"

	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/declarative"
	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/declarative/pkg/manifest"
)

// NewVersionCheck provides an implementation of declarative.Reconciled that
// checks the version of the operator if it is up to the version required by the manifest
func NewVersionCheck(client client.Client, operatorVersionString string) (*versionCheck, error) {
	operatorVersion, err := semver.Parse(operatorVersionString)
	if err != nil {
		return nil, fmt.Errorf("unable to parse operator version %q: %v", operatorVersionString, err)
	}
	return &versionCheck{client: client, operatorVersion: operatorVersion}, nil
}

type versionCheck struct {
	client          client.Client
	operatorVersion semver.Version
}

func (p *versionCheck) VersionCheck(
	ctx context.Context,
	src declarative.DeclarativeObject,
	objs *manifest.Objects,
) (bool, error) {
	log := log.Log
	var minOperatorVersion semver.Version

	// Look for annotation from any resource with the max version
	for _, obj := range objs.Items {
		annotations := obj.UnstructuredObject().GetAnnotations()
		if versionNeededStr, ok := annotations["addons.k8s.io/min-operator-version"]; ok {
			log.WithValues("version", versionNeededStr).Info("Got version requirement addons.k8s.io/operator-version=%v")

			versionNeeded, err := semver.Parse(versionNeededStr)
			if err != nil {
				log.WithValues("version", versionNeededStr).Error(err, "Unable to parse version restriction")
				return false, err
			}

			if versionNeeded.GT(minOperatorVersion) {
				minOperatorVersion = versionNeeded
			}
		}
	}

	if p.operatorVersion.GTE(minOperatorVersion) {
		return true, nil
	}

	addonObject, ok := src.(addonsv1alpha1.CommonObject)
	if !ok {
		return false, fmt.Errorf("object %T was not an addonsv1alpha1.CommonObject", src)
	}

	status := addonsv1alpha1.CommonStatus{
		Healthy: false,
		Errors: []string{
			fmt.Sprintf("manifest needs operator version >= %v, this operator is version %v", minOperatorVersion.String(), p.operatorVersion.String()),
		},
	}
	log.WithValues("name", addonObject.GetName()).WithValues("status", status).Info("updating status")
	addonObject.SetCommonStatus(status)

	return false, fmt.Errorf("operator not qualified, manifest needs operator version >= %v", minOperatorVersion.String())
}
