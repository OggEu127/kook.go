package kook

import "fmt"

func compatParams[T any](method string, args []any, legacy func([]any) (T, bool)) (T, error) {
	var zero T
	if len(args) == 1 {
		if params, ok := args[0].(T); ok {
			return params, nil
		}
	}
	if legacy != nil {
		if params, ok := legacy(args); ok {
			return params, nil
		}
	}
	return zero, NewValidationErrorWithValue("args", fmt.Sprintf("%s参数格式无效", method), fmt.Sprint(args))
}

func compatString(value any) (string, bool) {
	result, ok := value.(string)
	return result, ok
}

func compatInt(value any) (int, bool) {
	result, ok := value.(int)
	return result, ok
}

func optionalPositiveInt(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}

func optionalPositiveInt64(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	return &value
}

func stringPointer(value string) *string { return &value }
