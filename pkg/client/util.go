package client

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

func getResourceMapping(obj runtime.Object, mapper meta.RESTMapper, retryOnNoMatch bool) (*meta.RESTMapping, error) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if gvk.Empty() {
		return mapping, nil
	}

	if retryOnNoMatch && meta.IsNoMatchError(err) {
		// reset mapper if a NoMatchError is reported on the first call
		meta.MaybeResetRESTMapper(mapper)
		// return second call after reset
		mapping, err = mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return nil, fmt.Errorf("rest mapping for %s could not be resolved after reset: %w",
				obj.GetObjectKind().GroupVersionKind().String(), err)
		}
	}

	return mapping, err
}