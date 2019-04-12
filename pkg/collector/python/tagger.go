// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// +build python

package python

import (
	"unsafe"

	"github.com/DataDog/datadog-agent/pkg/tagger"
	"github.com/DataDog/datadog-agent/pkg/tagger/collectors"
)

/*
#include <datadog_agent_six.h>
#cgo !windows LDFLAGS: -ldatadog-agent-six -ldl
#cgo windows LDFLAGS: -ldatadog-agent-six -lstdc++ -static
*/
import "C"

// GetTags queries the agent6 tagger and returns a string array containing
// tags for the entity. If entity not found, or tagging error, the returned
// array is empty but valid.
// FIXME: replace highCard with a TagCardinality
//export GetTags
func GetTags(id *C.char, highCard C.int) **C.char {
	goID := C.GoString(id)
	var highCardBool bool
	var tags []string
	if highCard > 0 {
		highCardBool = true
	}

	if highCardBool == true {
		tags, _ = tagger.Tag(goID, collectors.HighCardinality)
	} else {
		tags, _ = tagger.Tag(goID, collectors.LowCardinality)
	}

	length := len(tags)
	if length == 0 {
		return nil
	}

	cTags := C.malloc(C.size_t(length+1) * C.size_t(unsafe.Sizeof(uintptr(0))))

	// convert the C array to a Go Array so we can index it
	indexTag := (*[1<<29 - 1]*C.char)(cTags)[: length+1 : length+1]
	indexTag[length] = nil
	for idx, tag := range tags {
		indexTag[idx] = C.CString(tag)
	}

	return (**C.char)(cTags)
}
