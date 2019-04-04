// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// +build kubelet

package kubelet

import (
	"hash/fnv"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// PodWatcher regularly pools the kubelet for new/changed/removed containers.
// It keeps an internal state to only send the updated pods.
type PodWatcher struct {
	sync.Mutex
	kubeUtil       *KubeUtil
	expiryDuration time.Duration
	lastSeen       map[string]time.Time
	tagsDigests    map[string]digests
}

type digests struct {
	labels      string
	annotations string
}

// NewPodWatcher creates a new watcher. User call must then trigger PullChanges
// and ExpireContainers when needed.
func NewPodWatcher(expiryDuration time.Duration) (*PodWatcher, error) {
	kubeutil, err := GetKubeUtil()
	if err != nil {
		return nil, err
	}
	watcher := &PodWatcher{
		kubeUtil:       kubeutil,
		lastSeen:       make(map[string]time.Time),
		tagsDigests:    make(map[string]digests),
		expiryDuration: expiryDuration,
	}
	return watcher, nil
}

// PullChanges pulls a new podList from the kubelet and returns Pod objects for
// new / updated pods. Updated pods will be sent entirely, user must replace
// previous info for these pods.
func (w *PodWatcher) PullChanges() ([]*Pod, error) {
	var podList []*Pod
	podList, err := w.kubeUtil.GetLocalPodList()
	if err != nil {
		return podList, err
	}
	return w.computeChanges(podList)
}

// computeChanges is used by PullChanges, split for testing
func (w *PodWatcher) computeChanges(podList []*Pod) ([]*Pod, error) {
	now := time.Now()
	var updatedPods []*Pod

	w.Lock()
	defer w.Unlock()
	for _, pod := range podList {
		// Only process ready pods
		if IsPodReady(pod) == false {
			continue
		}

		podEntity := PodUIDToEntityName(pod.Metadata.UID)
		newStaticPod := false
		_, foundPod := w.lastSeen[podEntity]

		if !foundPod {
			w.tagsDigests[podEntity] = digests{labels: digestMap(pod.Metadata.Labels), annotations: digestMap(pod.Metadata.Annotations)}
		}

		// static pods are included specifically because they won't have any container
		// as they're not updated in the pod list after creation
		if isPodStatic(pod) == true && foundPod == false {
			newStaticPod = true
		}

		// Refresh last pod seen time
		w.lastSeen[podEntity] = now

		// Detect new containers
		newContainer := false
		for _, container := range pod.Status.Containers {
			if _, foundContainer := w.lastSeen[container.ID]; foundContainer == false {
				newContainer = true
			}
			w.lastSeen[container.ID] = now
		}

		// Detect changes in labels and annotations values
		newLabelsOrAnnotations := false
		labelsDigest := digestMap(pod.Metadata.Labels)
		annotationsDigest := digestMap(pod.Metadata.Annotations)
		if foundPod && (labelsDigest != w.tagsDigests[podEntity].labels || annotationsDigest != w.tagsDigests[podEntity].annotations) {
			// update digests
			w.tagsDigests[podEntity] = digests{labels: labelsDigest, annotations: annotationsDigest}
			newLabelsOrAnnotations = true
		}

		if newStaticPod || newContainer || newLabelsOrAnnotations {
			updatedPods = append(updatedPods, pod)
		}
	}
	log.Debugf("Found %d changed pods out of %d", len(updatedPods), len(podList))
	return updatedPods, nil
}

// Expire returns a list of entities (containers and pods)
// that are not listed in the podlist anymore. It must be called
// immediately after a PullChanges.
// For containers, string is kubernetes container ID (with runtime name)
// For pods, string is "kubernetes_pod://uid" format
func (w *PodWatcher) Expire() ([]string, error) {
	now := time.Now()
	w.Lock()
	defer w.Unlock()
	var expiredContainers []string

	for id, lastSeen := range w.lastSeen {
		if now.Sub(lastSeen) > w.expiryDuration {
			expiredContainers = append(expiredContainers, id)
		}
	}
	if len(expiredContainers) > 0 {
		for _, id := range expiredContainers {
			delete(w.lastSeen, id)
			delete(w.tagsDigests, id)
		}
	}
	return expiredContainers, nil
}

// GetPodForEntityID finds the pod corresponding to an entity.
// EntityIDs can be Docker container IDs or pod UIDs (prefixed).
// Returns a nil pointer if not found.
func (w *PodWatcher) GetPodForEntityID(entityID string) (*Pod, error) {
	return w.kubeUtil.GetPodForEntityID(entityID)
}

// digestMap return a unique hash for a map
// used to track changes in labels and annotations values
func digestMap(m map[string]string) string {
	if m == nil {
		return ""
	}

	// to store the keys in slice in sorted order
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := fnv.New64()
	for _, k := range keys {
		h.Write([]byte(m[k]))
	}
	return strconv.FormatUint(h.Sum64(), 16)
}
