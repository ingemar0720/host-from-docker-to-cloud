package deployplan

import (
	"fmt"
	"sort"

	"github.com/compose-spec/compose-go/v2/types"
)

// Order returns a deterministic topological order for service deployment.
// Ties are broken lexically by service name for stable output.
func Order(project *types.Project) ([]string, error) {
	names := project.ServiceNames()
	sort.Strings(names)

	indegree := make(map[string]int, len(names))
	dependents := make(map[string][]string, len(names))
	for _, name := range names {
		indegree[name] = 0
	}

	for _, name := range names {
		svc := project.Services[name]
		deps := make([]string, 0, len(svc.DependsOn))
		for dep := range svc.DependsOn {
			deps = append(deps, dep)
		}
		sort.Strings(deps)
		for _, dep := range deps {
			if _, ok := indegree[dep]; !ok {
				continue
			}
			indegree[name]++
			dependents[dep] = append(dependents[dep], name)
		}
	}

	ready := make([]string, 0, len(names))
	for _, name := range names {
		if indegree[name] == 0 {
			ready = append(ready, name)
		}
	}
	sort.Strings(ready)

	order := make([]string, 0, len(names))
	for len(ready) > 0 {
		name := ready[0]
		ready = ready[1:]
		order = append(order, name)

		ds := dependents[name]
		sort.Strings(ds)
		for _, d := range ds {
			indegree[d]--
			if indegree[d] == 0 {
				i := sort.SearchStrings(ready, d)
				ready = append(ready, "")
				copy(ready[i+1:], ready[i:])
				ready[i] = d
			}
		}
	}

	if len(order) != len(names) {
		return nil, fmt.Errorf("dependency planner: cycle detected in services")
	}
	return order, nil
}
