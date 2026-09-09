package component

// ForEach maps a slice of items of type T to a slice of Components using the provided mapping function.
// The resulting slice can be directly spread into container layouts such as VStack(...) or HStack(...).
//
// Example:
//
//	limoni.VStack(
//	    limoni.ForEach(todos, func(todo Todo, i int) limoni.Component {
//	        return limoni.Text(fmt.Sprintf("%d. %s", i+1, todo.Title))
//	    })...,
//	)
func ForEach[T any](items []T, fn func(item T, index int) Component) []Component {
	if len(items) == 0 {
		return nil
	}
	result := make([]Component, len(items))
	for i, item := range items {
		result[i] = fn(item, i)
	}
	return result
}
