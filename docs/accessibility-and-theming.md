# ♿ Accessibility (A11y) and Theming

Limoni treats accessibility as a foundational component of the core engine rather than an afterthought (`core/accessibility`).

---

## 1. Semantic Accessibility Tree (`accessibility.Tree`)

When widgets are rendered, they register structured semantic nodes alongside terminal cells:
- **Role**: `RoleButton`, `RoleTable`, `RoleInput`, `RoleDialog`, `RoleProgress`, etc.
- **Attributes**: Label, current value, description, and state (e.g., "Selected", "Disabled", "60% Complete").
- Screen reader software and automated testing agents can inspect this structured tree directly rather than attempting to heuristically parse raw screen text.

---

## 2. Standard A11y Modes

1. **High Contrast Mode (`accessibility.ModeHighContrast`)**:
   - Automatically remaps low-contrast pastel or muted tones to WCAG AAA compliant high-contrast color pairs (e.g., stark black, white, and yellow).
2. **`NO_COLOR` Specification**:
   - Respects the standard `NO_COLOR=1` environment variable. When active, color escape codes are completely stripped, and semantic visual states fall back to high-contrast unicode symbols and formatting modifiers (bold, underline).
3. **Reduced Motion (`accessibility.ModeReducedMotion`)**:
   - Automatically scales animation durations and physics steps to zero for users with vestibular sensitivities, instantly snapping components to their final states.
