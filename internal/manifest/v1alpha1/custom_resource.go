package v1alpha1

import (
	"context"

	manifestv1alpha1 "github.com/kyma-project/module-manager/api/v1alpha1"
	declarative "github.com/kyma-project/module-manager/pkg/declarative/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const CustomResourceManager = "resource.kyma-project.io/finalizer"

// PostRunCreateCR is a hook for creating the manifest default custom resource if not available in the cluster
// It is used to provide the controller with default data in the Runtime.
func PostRunCreateCR(
	ctx context.Context, skr declarative.Client, kcp client.Client, obj declarative.Object,
) error {
	manifest := obj.(*manifestv1alpha1.Manifest)
	resource := manifest.Spec.Resource.DeepCopy()
	if resource.Object == nil {
		return nil
	}

	if err := skr.Create(
		ctx, resource, client.FieldOwner(CustomResourceManager),
	); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	oMeta := &v1.PartialObjectMetadata{}
	oMeta.SetName(obj.GetName())
	oMeta.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	oMeta.SetNamespace(obj.GetNamespace())
	oMeta.SetFinalizers(obj.GetFinalizers())
	if added := controllerutil.AddFinalizer(oMeta, CustomResourceManager); added {
		if err := kcp.Patch(
			ctx, oMeta, client.Apply, client.ForceOwnership, client.FieldOwner(CustomResourceManager),
		); err != nil {
			return err
		}
	}
	return nil
}

// PreDeleteDeleteCR is a hook for deleting the manifest default custom resource if available in the cluster
// It is used to clean up the controller default data.
func PreDeleteDeleteCR(
	ctx context.Context, skr declarative.Client, kcp client.Client, obj declarative.Object,
) error {
	manifest := obj.(*manifestv1alpha1.Manifest)
	resource := manifest.Spec.Resource.DeepCopy()
	if resource.Object == nil {
		return nil
	}

	if err := skr.Delete(ctx, resource); err != nil && !errors.IsNotFound(err) {
		return err
	}

	onCluster := manifest.DeepCopy()
	err := kcp.Get(ctx, client.ObjectKeyFromObject(obj), onCluster)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if removed := controllerutil.RemoveFinalizer(onCluster, CustomResourceManager); removed {
		if err := kcp.Update(
			ctx, onCluster, client.FieldOwner(CustomResourceManager),
		); err != nil {
			return err
		}
	}
	return nil
}
