package controllers_test

import (
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/kyma-project/module-manager/operator/api/v1alpha1"
	"github.com/kyma-project/module-manager/operator/pkg/labels"
	manifestTypes "github.com/kyma-project/module-manager/operator/pkg/types"
	"github.com/kyma-project/module-manager/operator/pkg/util"
	_ "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockLayer struct {
	filePath string
}

func (m mockLayer) Digest() (v1.Hash, error) {
	r, err := m.Compressed()
	if err != nil {
		return v1.Hash{}, err
	}
	defer r.Close()
	hash, _, err := v1.SHA256(r)
	return hash, err
}

func (m mockLayer) MediaType() (types.MediaType, error) {
	return types.OCILayer, nil
}

func (m mockLayer) Size() (int64, error) { return 137438691328, nil }
func (m mockLayer) Compressed() (io.ReadCloser, error) {
	f, err := os.Open(m.filePath)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(f), nil
}

func (m mockLayer) Uncompressed() (io.ReadCloser, error) {
	f, err := os.Open("../pkg/test_samples/oci/config.yaml")
	if err != nil {
		return nil, err
	}
	return io.NopCloser(f), nil
}

func (m mockLayer) DiffID() (v1.Hash, error) {
	return v1.Hash{Algorithm: "fake", Hex: "diff id"}, nil
}

func CreateImageSpecLayer(ociLayerType OCILayerType) v1.Layer {
	var layer v1.Layer
	var err error
	if ociLayerType == layerCRDs {
		layer, err = partial.CompressedToLayer(mockLayer{filePath: "../pkg/test_samples/oci/crd.tgz"})
	} else {
		layer, err = partial.CompressedToLayer(mockLayer{filePath: "../pkg/test_samples/oci/compressed.tgz"})
	}
	Expect(err).ToNot(HaveOccurred())
	return layer
}

type OCILayerType string

// Valid Helm States.
const (
	layerCRDs OCILayerType = "crds"

	layerInstalls OCILayerType = "Installs"
)

func PushToRemoteOCIRegistry(layerName string, ociLayerType OCILayerType) {
	layer := CreateImageSpecLayer(ociLayerType)
	digest, err := layer.Digest()
	Expect(err).ToNot(HaveOccurred())

	// Set up a fake registry and write what we pulled to it.
	u, err := url.Parse(server.URL)
	Expect(err).NotTo(HaveOccurred())

	dst := fmt.Sprintf("%s/%s@%s", u.Host, layerName, digest)
	ref, err := name.NewDigest(dst)
	Expect(err).ToNot(HaveOccurred())

	err = remote.WriteLayer(ref.Context(), layer)
	Expect(err).ToNot(HaveOccurred())

	got, err := remote.Layer(ref)
	Expect(err).ToNot(HaveOccurred())
	gotHash, err := got.Digest()
	Expect(err).ToNot(HaveOccurred())
	Expect(gotHash).To(Equal(digest))
}

func createImageSpec(name, repo string, ociLayerType OCILayerType) manifestTypes.ImageSpec {
	imageSpec := manifestTypes.ImageSpec{
		Name: name,
		Repo: repo,
		Type: "oci-ref",
	}
	layer := CreateImageSpecLayer(ociLayerType)
	digest, err := layer.Digest()
	Expect(err).ToNot(HaveOccurred())
	imageSpec.Ref = digest.String()
	return imageSpec
}

func NewTestManifest(name string, componentOwner string) *v1alpha1.Manifest {
	return &v1alpha1.Manifest{
		TypeMeta: metav1.TypeMeta{
			Kind: "Manifest",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + RandString(8),
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				labels.ComponentOwner: componentOwner,
				labels.CacheKey:       RandString(8),
			},
		},
	}
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))] //nolint:gosec
	}
	return string(b)
}

func deleteHelmChartResources(imageSpec manifestTypes.ImageSpec) {
	chartYamlPath := filepath.Join(util.GetFsChartPath(imageSpec), "Chart.yaml")
	Expect(os.RemoveAll(chartYamlPath)).Should(Succeed())
	valuesYamlPath := filepath.Join(util.GetFsChartPath(imageSpec), "values.yaml")
	Expect(os.RemoveAll(valuesYamlPath)).Should(Succeed())
	templatesPath := filepath.Join(util.GetFsChartPath(imageSpec), "templates")
	Expect(os.RemoveAll(templatesPath)).Should(Succeed())
}

func verifyHelmResourcesDeletion(imageSpec manifestTypes.ImageSpec) {
	_, err := os.Stat(filepath.Join(util.GetFsChartPath(imageSpec), "Chart.yaml"))
	Expect(os.IsNotExist(err)).To(BeTrue())
	_, err = os.Stat(filepath.Join(util.GetFsChartPath(imageSpec), "values.yaml"))
	Expect(os.IsNotExist(err)).To(BeTrue())
	_, err = os.Stat(filepath.Join(util.GetFsChartPath(imageSpec), "templates"))
	Expect(os.IsNotExist(err)).To(BeTrue())
}
