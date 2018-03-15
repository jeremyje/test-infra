/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clone

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/test-infra/prow/kube"
)

// ParseRefs parses a human-provided string into the repo
// that should be cloned and the refs that need to be
// checked out once it is. The format is:
//   org,repo=base-ref[:base-sha][,pull-id[:pull-sha]]...
// For the base ref and pull IDs, a SHA may optionally be
// provided or may be omitted for the latest available SHA.
// Examples:
//   kubernetes,test-infra=master
//   kubernetes,test-infra=master:abcde12
//   kubernetes,test-infra=master:abcde12,34
//   kubernetes,test-infra=master:abcde12,34:fghij56
//   kubernetes,test-infra=master,34:fghij56
//   kubernetes,test-infra=master:abcde12,34:fghij56,78
func ParseRefs(value string) (kube.Refs, error) {
	gitRef := kube.Refs{}
	values := strings.SplitN(value, "=", 2)
	if len(values) != 2 {
		return gitRef, fmt.Errorf("refspec %s invalid: does not contain '='", value)
	}
	info := values[0]
	allRefs := values[1]

	infoValues := strings.SplitN(info, ",", 2)
	if len(infoValues) != 2 {
		return gitRef, fmt.Errorf("refspec %s invalid: does not contain 'org,repo' as prefix", value)
	}
	gitRef.Org = infoValues[0]
	gitRef.Repo = infoValues[1]

	refValues := strings.Split(allRefs, ",")
	if len(refValues) == 1 && refValues[0] == "" {
		return gitRef, fmt.Errorf("refspec %s invalid: does not contain any refs", value)
	}
	baseRefParts := strings.Split(refValues[0], ":")
	if len(baseRefParts) != 1 && len(baseRefParts) != 2 {
		return gitRef, fmt.Errorf("refspec %s invalid: malformed base ref", refValues[0])
	}
	gitRef.BaseRef = baseRefParts[0]
	if len(baseRefParts) == 2 {
		gitRef.BaseSHA = baseRefParts[1]
	}
	for _, refValue := range refValues[1:] {
		refParts := strings.Split(refValue, ":")
		if len(refParts) != 1 && len(refParts) != 2 {
			return gitRef, fmt.Errorf("refspec %s invalid: malformed pull ref", refValue)
		}
		pullNumber, err := strconv.Atoi(refParts[0])
		if err != nil {
			return gitRef, fmt.Errorf("refspec %s invalid: pull request identifier not a number: %v", refValue, err)
		}
		pullRef := kube.Pull{
			Number: pullNumber,
		}
		if len(refParts) == 2 {
			pullRef.SHA = refParts[1]
		}
		gitRef.Pulls = append(gitRef.Pulls, pullRef)
	}

	return gitRef, nil
}

// PathResolver provides path overrides for a given set
// of repos
type PathResolver struct {
	org  string
	repo string

	path string
}

// Resolve returns an override clone path if the org and
// repo match the settings in in the resolver
func (r *PathResolver) Resolve(org, repo string) string {
	if r.org == org && (r.repo == "" || r.repo == repo) {
		return r.path
	}

	return ""
}

func (r *PathResolver) String() string {
	var prefix string
	if r.repo == "" {
		prefix = r.org
	} else {
		prefix = fmt.Sprintf("%s,%s", r.org, r.repo)
	}
	return fmt.Sprintf("%s=%s", prefix, r.path)
}

// ParseAliases parses a human-provided string into a
// PathResolver that resolves the path under the
// $GOPATH/src directory where the repository should
// be cloned. The format for the human-provided string
// is:
//   org[,repo]=path
// The repository is optional and if not set, all repos
// for the org will be captured. Exmaples:
//   kubernetes=k8s.io
//   myorg,non-go-project=somewhere/else
func ParseAliases(value string) (PathResolver, error) {
	var resolver PathResolver
	values := strings.SplitN(value, "=", 2)
	if len(values) != 2 {
		return resolver, fmt.Errorf("path override %s invalid: does not contain '='", value)
	}
	info := values[0]
	resolver.path = values[1]

	infoValues := strings.SplitN(info, ",", 2)
	switch len(infoValues) {
	case 1:
		resolver.org = infoValues[0]
		return resolver, nil
	case 2:
		resolver.org = infoValues[0]
		resolver.repo = infoValues[1]
		return resolver, nil
	default:
		return resolver, fmt.Errorf("path override %s invalid: does not contain 'org[,repo]' as prefix", value)
	}
}
