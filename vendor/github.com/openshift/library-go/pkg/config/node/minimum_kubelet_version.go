package node

import (
	"context"
	"fmt"
	"strings"

	"github.com/blang/semver/v4"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

func ValidateMinimumKubeletVersion(nodesGetter corev1client.NodesGetter, minimumKubeletVersion string) *field.Error {
	// unset, no error
	if minimumKubeletVersion == "" {
		return nil
	}

	fieldPath := field.NewPath("spec", "minimumKubeletVersion")
	nodes, err := nodesGetter.Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return field.Forbidden(fieldPath, fmt.Sprintf("Getting nodes to compare minimum version %v", err.Error()))
	}

	version, err := semver.Parse(minimumKubeletVersion)
	if err != nil {
		return field.Invalid(fieldPath, minimumKubeletVersion, fmt.Sprintf("Failed to parse submitted version %s %v", minimumKubeletVersion, err.Error()))
	}

	for _, node := range nodes.Items {
		_, errStr := IsKubeletVersionTooOld(&node, &version)
		if errStr != "" {
			return field.Invalid(fieldPath, minimumKubeletVersion, errStr)
		}
	}
	return nil
}

func IsKubeletVersionTooOld(node *corev1.Node, minVersion *semver.Version) (bool, string) {
	version, err := semver.Parse(strings.TrimPrefix(node.Status.NodeInfo.KubeletVersion, "v"))
	if err != nil {
		return false, fmt.Sprintf("failed to parse node version %s: %v", node.Status.NodeInfo.KubeletVersion, err)
	}

	version.Pre = nil
	version.Build = nil

	name := node.ObjectMeta.Name
	if minVersion.GT(version) {
		return true, fmt.Sprintf("kubelet version of node %s is %v, which is lower than minimumKubeletVersion of %v", name, version, *minVersion)
	}
	return false, ""
}
