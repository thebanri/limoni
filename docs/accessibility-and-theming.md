# ♿ Accessibility (A11y) and Theming

Limoni treats accessibility as a foundational component of the core engine rather than an afterthought (`core/accessibility`).

---

## 1. Semantic Accessibility Tree (`accessibility.Tree`)

When widgets are rendered, they register structured semantic nodes alongside terminal cells:
- **Role**: `RoleButton`, `RoleTable`, `RoleInput`, `RoleDialog`, `RoleProgress`, etc.
- **Attributes**: Label, current value, description, and state (e.g., "Selected", "Disabled", "60% Complete").
- **Position**: `Position` and `SetSize` mirror ARIA's `aria-posinset` / `aria-setsize`, which is what lets a reader say "3 of 20". They are integers, not a preformatted string, because a widget rebuilds its node every frame — formatting would allocate on the draw path. `LineMode` renders them when a tree is actually consumed.
- Screen reader software and automated testing agents can inspect this structured tree directly rather than attempting to heuristically parse raw screen text.

### Widget coverage

A widget that does not implement `accessibility.Provider` is silent to a screen
reader, however well it renders. Covered today:

| Widget | Role | What the node carries |
| :--- | :--- | :--- |
| `List` | `RoleList` | selected item text, position in set |
| `Select` | `RoleList` | selected option, `expanded` when open |
| `Tabs` | `RoleList` | active tab title, position in set |
| `Table` | `RoleTable` | selected row's first cell, row position |
| `TreeView` | `RoleTree` | selected node ID |
| `Dialog` | `RoleDialog` | title, message, sub-message, button count |
| `TextInput` / `TextArea` | `RoleInput` | current value |
| `Checkbox` / `RadioButton` | `RoleCheckbox` / `RoleRadioButton` | checked state |
| `Slider` | `RoleSlider` | current value |
| `ProgressBar` | `RoleProgress` | completion |
| `ColorPicker` | — | selected colour |
| `Paragraph` | `RoleGeneric` | its text, which is otherwise unannounced |

`widgets.Accessible` is the escape hatch: embed it to describe any widget that
has no node of its own. `Table` and `List` deliberately describe the *selected*
row or item rather than enumerating children — a table may hold a million rows,
and a child per row would allocate on every frame.

---

## 2. Standard A11y Modes

1. **High Contrast Mode (`accessibility.ModeHighContrast`)**:
   - Automatically remaps low-contrast pastel or muted tones to WCAG AAA compliant high-contrast color pairs (e.g., stark black, white, and yellow).
2. **`NO_COLOR` Specification**:
   - Respects the standard `NO_COLOR=1` environment variable. When active, color escape codes are completely stripped, and semantic visual states fall back to high-contrast unicode symbols and formatting modifiers (bold, underline).
3. **Reduced Motion (`accessibility.ModeReducedMotion`)**:
   - Automatically scales animation durations and physics steps to zero for users with vestibular sensitivities, instantly snapping components to their final states.
