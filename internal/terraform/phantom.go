package terraform

import (
	"encoding/json"
	"reflect"
	"sort"
)

type PhantomResult struct {
	PhantomCount     int
	RealCount        int
	PhantomAddresses []string
}

func DetectPhantomChanges(changes []PlanChange) PhantomResult {
	result := PhantomResult{
		PhantomAddresses: make([]string, 0),
	}

	for i := range changes {
		if changes[i].Action != ActionUpdate {
			continue
		}

		if IsPhantomChange(&changes[i]) {
			result.PhantomCount++
			result.PhantomAddresses = append(result.PhantomAddresses, changes[i].Resource.Address)
			changes[i].IsPhantom = true
		} else {
			result.RealCount++
		}
	}

	return result
}

func IsPhantomChange(change *PlanChange) bool {
	if change.Action != ActionUpdate {
		return false
	}

	for _, diff := range change.AttributeDiffs {
		oldNorm := normalizeJSON(diff.OldValue)
		newNorm := normalizeJSON(diff.NewValue)
		if oldNorm != newNorm {
			return false
		}
	}

	return len(change.AttributeDiffs) > 0
}

func normalizeJSON(s string) string {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	normalized := normalizeValue(v)
	b, err := json.Marshal(normalized)
	if err != nil {
		return s
	}
	return string(b)
}

func normalizeValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[string]interface{}:
		normalized := make(map[string]interface{})
		for k, child := range val {
			if child == nil {
				continue
			}
			normalized[k] = normalizeValue(child)
		}
		return normalized

	case []interface{}:
		type keyed struct {
			key string
			val interface{}
		}
		items := make([]keyed, len(val))
		for i, elem := range val {
			norm := normalizeValue(elem)
			b, _ := json.Marshal(norm)
			items[i] = keyed{key: string(b), val: norm}
		}
		sort.Slice(items, func(i, j int) bool {
			return items[i].key < items[j].key
		})
		result := make([]interface{}, len(items))
		for i, item := range items {
			result[i] = item.val
		}
		return result

	default:
		return v
	}
}

func NormalizedEqual(a, b interface{}) bool {
	return reflect.DeepEqual(normalizeValue(a), normalizeValue(b))
}
