package accessibility

import "fmt"

// ValidateTree checks structural invariants required by screen-reader and
// inspector consumers: IDs must be unique and interactive nodes need labels.
func ValidateTree(nodes []AccessibilityNode) error {
	seen := make(map[string]struct{})
	for _, node := range nodes {
		if err := validateNode(node, seen); err != nil {
			return err
		}
	}
	return nil
}

func validateNode(node AccessibilityNode, seen map[string]struct{}) error {
	if node.ID != "" {
		if _, exists := seen[node.ID]; exists {
			return fmt.Errorf("accessibility: duplicate node ID %q", node.ID)
		}
		seen[node.ID] = struct{}{}
	}
	if (node.Role == RoleButton || node.Role == RoleInput || node.Role == RoleCheckbox) && node.Label == "" {
		return fmt.Errorf("accessibility: interactive node %q has no label", node.ID)
	}
	for _, child := range node.Children {
		if err := validateNode(child, seen); err != nil {
			return err
		}
	}
	return nil
}
