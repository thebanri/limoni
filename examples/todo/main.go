package main

import (
	"fmt"
	"os"
	"time"

	"github.com/thebanri/limoni/animation"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

type text struct {
	value string
	style cell.Style
}

func (t text) Draw(ctx cell.Context, buf *buffer.Buffer) {
	buf.SetString(ctx.Area.X, ctx.Area.Y, t.value, ctx.Style.Merge(t.style))
}

func (t text) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return uint16(len(t.value)), 1
}

type TodoTask struct {
	ID        int
	Title     string
	Category  string
	DueDate   time.Time
	Completed bool
}

type AppState struct {
	Tasks         []TodoTask
	TaskIDCounter int
	ActiveFilter  string // "All", "Active", "Completed"
	ProgressBar   *widgets.AnimatableProgressBar
	ShowHelp      bool

	InputState    *widgets.TextInputState
	CategoryState *widgets.SelectState
	DueDateState  *widgets.SelectState
	ListState     *widgets.ListState
	FilterState   *widgets.ListState
}

func main() {
	b := backend.NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing backend: %v\n", err)
		os.Exit(1)
	}
	defer b.Close()

	t, err := terminal.New(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing terminal: %v\n", err)
		os.Exit(1)
	}

	b.StartEventLoop()

	inputState := widgets.NewTextInputState()
	categoryState := widgets.NewSelectState()
	dueDateState := widgets.NewSelectState()
	listState := widgets.NewListState()
	listState.Selected = 0

	filterState := widgets.NewListState()
	filterState.Selected = 0

	// Create animatable progress bar
	progressBar := widgets.NewAnimatableProgressBar("todo_progress", 0.0)
	progressBar.FilledStyle = cell.Style{Fg: cell.NewColorRGB(0, 255, 128)}
	progressBar.EmptyStyle = cell.Style{Fg: cell.NewColorRGB(50, 50, 60)}

	state := &AppState{
		Tasks: []TodoTask{
			{ID: 1, Title: "Optimize Limoni layout flex engine", Category: "Work", DueDate: time.Now().AddDate(0, 0, 1), Completed: false},
			{ID: 2, Title: "Weekly grocery store shopping", Category: "Shopping", DueDate: time.Now(), Completed: true},
			{ID: 3, Title: "Finalize personal website dark theme", Category: "Personal", DueDate: time.Now().AddDate(0, 0, -2), Completed: false},
			{ID: 4, Title: "Verify all examples and automated tests", Category: "Work", DueDate: time.Now().AddDate(0, 0, 5), Completed: false},
		},
		TaskIDCounter: 4,
		ActiveFilter:  "All",
		ProgressBar:   progressBar,
		ShowHelp:      false,
		InputState:    inputState,
		CategoryState: categoryState,
		DueDateState:  dueDateState,
		ListState:     listState,
		FilterState:   filterState,
	}

	updateTargetProgress := func() {
		completedCount := 0
		for _, task := range state.Tasks {
			if task.Completed {
				completedCount++
			}
		}
		totalCount := len(state.Tasks)
		pct := 0.0
		if totalCount > 0 {
			pct = float64(completedCount) / float64(totalCount) * 100.0
		}
		// Smooth transition over 400ms using EaseOutCubic easing
		state.ProgressBar.AnimateTo(pct, 400*time.Millisecond, animation.EaseOutCubic)
	}

	updateTargetProgress()
	state.ProgressBar.ProgressBar.Value = state.ProgressBar.Anim.Value()

	t.FocusManager().SetFocused("todo_input")

	draw := func() {
		drawApp(t, state)
	}

	ticker := time.NewTicker(33 * time.Millisecond)
	defer ticker.Stop()

	draw()

	for {
		select {
		case ev, ok := <-b.Events():
			if !ok {
				return
			}
			switch ev.Type {
			case backend.EventKey:
				if state.ShowHelp {
					state.ShowHelp = false
					draw()
					continue
				}

				if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q') {
					return
				}

				if ev.Key.Type == backend.KeyTab {
					t.FocusManager().Next()
					draw()
					continue
				}

				focused := t.FocusManager().Focused()
				if focused != "todo_input" && ev.Key.Type == backend.KeyRune && (ev.Key.Ch == '?' || ev.Key.Ch == 'h') {
					state.ShowHelp = true
					draw()
					continue
				}

				switch focused {
				case "todo_input":
					if ev.Key.Type == backend.KeyEnter {
						title := state.InputState.Value()
						if title != "" {
							categories := []string{"Work", "Personal", "Shopping"}
							cat := categories[state.CategoryState.Selected%len(categories)]

							var dueDate time.Time
							switch state.DueDateState.Selected % 4 {
							case 0:
								dueDate = time.Now()
							case 1:
								dueDate = time.Now().AddDate(0, 0, 1)
							case 2:
								dueDate = time.Now().AddDate(0, 0, 3)
							case 3:
								dueDate = time.Now().AddDate(0, 0, 7)
							}

							state.TaskIDCounter++
							state.Tasks = append(state.Tasks, TodoTask{
								ID:        state.TaskIDCounter,
								Title:     title,
								Category:  cat,
								DueDate:   dueDate,
								Completed: false,
							})
							state.InputState.SetValue("")
							updateTargetProgress()
						}
					} else {
						state.InputState.HandleKey(ev.Key)
					}

				case "category_select":
					state.CategoryState.HandleKey(ev.Key, 3)

				case "due_date_select":
					state.DueDateState.HandleKey(ev.Key, 4)

				case "filter_list":
					filters := []string{"All", "Active", "Completed"}
					if ev.Key.Type == backend.KeyArrowUp {
						if state.FilterState.Selected > 0 {
							state.FilterState.Selected--
						}
					}
					if ev.Key.Type == backend.KeyArrowDown {
						if state.FilterState.Selected < len(filters)-1 {
							state.FilterState.Selected++
						}
					}
					state.ActiveFilter = filters[state.FilterState.Selected%len(filters)]

				case "task_list":
					filtered := getFilteredTasks(state)
					if len(filtered) > 0 {
						if ev.Key.Type == backend.KeyArrowUp {
							if state.ListState.Selected > 0 {
								state.ListState.Selected--
							}
						}
						if ev.Key.Type == backend.KeyArrowDown {
							if state.ListState.Selected < len(filtered)-1 {
								state.ListState.Selected++
							}
						}
						if ev.Key.Type == backend.KeySpace || ev.Key.Type == backend.KeyEnter {
							targetTask := filtered[state.ListState.Selected]
							for idx, task := range state.Tasks {
								if task.ID == targetTask.ID {
									state.Tasks[idx].Completed = !state.Tasks[idx].Completed
									updateTargetProgress()
									break
								}
							}
						}
						if (ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'd') || ev.Key.Type == backend.KeyBackspace {
							targetTask := filtered[state.ListState.Selected]
							for idx, task := range state.Tasks {
								if task.ID == targetTask.ID {
									state.Tasks = append(state.Tasks[:idx], state.Tasks[idx+1:]...)
									updateTargetProgress()
									break
								}
							}
							newFiltered := getFilteredTasks(state)
							if state.ListState.Selected >= len(newFiltered) && len(newFiltered) > 0 {
								state.ListState.Selected = len(newFiltered) - 1
							}
						}
					}
				}
				draw()

			case backend.EventMouse:
				t.RouteMouseEvent(ev.Mouse)

				if ev.Mouse.Button == backend.MouseLeft && !ev.Mouse.Drag {
					w, h, _ := b.Size()
					screenArea := cell.NewRect(0, 0, w, h)
					_, _, filterRect, listRect, _ := getLayoutRects(screenArea)

					innerFilter := cell.NewRect(filterRect.X+1, filterRect.Y+1, filterRect.Width-2, filterRect.Height-2)
					innerList := cell.NewRect(listRect.X+1, listRect.Y+1, listRect.Width-2, listRect.Height-2)

					if innerList.Contains(ev.Mouse.X, ev.Mouse.Y) {
						clickedY := int(ev.Mouse.Y - innerList.Y)
						targetIdx := state.ListState.Offset + clickedY
						filtered := getFilteredTasks(state)
						if targetIdx >= 0 && targetIdx < len(filtered) {
							targetTask := filtered[targetIdx]
							for idx, task := range state.Tasks {
								if task.ID == targetTask.ID {
									state.Tasks[idx].Completed = !state.Tasks[idx].Completed
									state.ListState.Selected = targetIdx
									updateTargetProgress()
									break
								}
							}
						}
					} else if innerFilter.Contains(ev.Mouse.X, ev.Mouse.Y) {
						clickedY := int(ev.Mouse.Y - innerFilter.Y)
						targetIdx := state.FilterState.Offset + clickedY
						filters := []string{"All", "Active", "Completed"}
						if targetIdx >= 0 && targetIdx < len(filters) {
							state.FilterState.Selected = targetIdx
							state.ActiveFilter = filters[targetIdx]
							state.ListState.Selected = 0
						}
					}
				}
				draw()

			case backend.EventResize:
				draw()
			}

		case <-ticker.C:
			if state.ProgressBar.Update(time.Now()) {
				draw()
			}
		}
	}
}

func getFilteredTasks(state *AppState) []TodoTask {
	var result []TodoTask
	for _, task := range state.Tasks {
		switch state.ActiveFilter {
		case "Active":
			if !task.Completed {
				result = append(result, task)
			}
		case "Completed":
			if task.Completed {
				result = append(result, task)
			}
		default:
			result = append(result, task)
		}
	}
	return result
}

func getLayoutRects(area cell.Rect) (headerRect, formRect, filterRect, listRect, footerRect cell.Rect) {
	rootLay := layout.NewFlexLayout(
		layout.Vertical,
		0,
		layout.Fixed(3), // Header
		layout.Fixed(5), // Add Form
		layout.Fill(),   // Content
		layout.Fixed(3), // Footer
	)
	chunks := rootLay.Split(area)
	headerRect = chunks[0]
	formRect = chunks[1]
	footerRect = chunks[3]

	contentLay := layout.NewFlexLayout(
		layout.Horizontal,
		1,
		layout.Percentage(30),
		layout.Percentage(70),
	)
	contentChunks := contentLay.Split(chunks[2])
	filterRect = contentChunks[0]
	listRect = contentChunks[1]
	return
}

func drawApp(t *terminal.Terminal, state *AppState) {
	t.Draw(func(f *terminal.Frame) {
		area := f.Buffer.Area
		f.SetTheme(widgets.DarkTheme())

		accentColor := cell.NewColorRGB(255, 180, 0)

		headerRect, formRect, filterRect, listRect, footerRect := getLayoutRects(area)

		// Header
		f.RenderWidget(widgets.Block{
			Title:          " LIMONI TASK & TODO MANAGER ",
			TitleAlignment: widgets.AlignCenter,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: accentColor},
			Child:          text{value: " Organize, plan, and categorize tasks with keyboard and mouse shortcuts ", style: cell.Style{Fg: cell.NewColorRGB(200, 200, 200)}},
		}, headerRect)

		// Add Form Split: TextInput (50%) + Category Select (25%) + DueDate Select (25%)
		formLay := layout.NewFlexLayout(
			layout.Horizontal,
			1,
			layout.Percentage(50),
			layout.Percentage(25),
			layout.Percentage(25),
		)
		formChunks := formLay.Split(formRect)

		focused := t.FocusManager().Focused()

		inputBorderCol := cell.NewColorRGB(60, 65, 80)
		if focused == "todo_input" {
			inputBorderCol = accentColor
		}
		f.RenderWidget(widgets.Block{
			Title:         " NEW TASK ",
			Borders:       widgets.BorderAll,
			BorderSymbols: widgets.SymbolsRounded,
			BorderStyle:   cell.Style{Fg: inputBorderCol},
			Child: widgets.TextInput{
				ID:               "todo_input",
				State:            state.InputState,
				Placeholder:      "Type a task title and press Enter...",
				Style:            cell.Style{Fg: cell.NewColorRGB(220, 220, 220)},
				FocusedStyle:     cell.Style{Fg: cell.NewColorRGB(255, 255, 255)},
				PlaceholderStyle: cell.Style{Fg: cell.NewColorRGB(120, 120, 130), Modifier: cell.ModifierItalic},
			},
		}, formChunks[0])

		selectBorderCol := cell.NewColorRGB(60, 65, 80)
		if focused == "category_select" {
			selectBorderCol = accentColor
		}
		f.RenderWidget(widgets.Block{
			Title: " CATEGORY ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: cell.Style{Fg: selectBorderCol},
			Child: widgets.Select{
				ID:            "category_select",
				Options:       []string{"Work", "Personal", "Shopping"},
				State:         state.CategoryState,
				Style:         cell.Style{Fg: cell.NewColorRGB(200, 200, 200), Bg: cell.NewColorRGB(25, 25, 30)},
				SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: accentColor, Modifier: cell.ModifierBold},
				HoverStyle:    cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(45, 45, 50)},
			},
		}, formChunks[1])

		dueBorderCol := cell.NewColorRGB(60, 65, 80)
		if focused == "due_date_select" {
			dueBorderCol = accentColor
		}
		f.RenderWidget(widgets.Block{
			Title: " DUE DATE ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: cell.Style{Fg: dueBorderCol},
			Child: widgets.Select{
				ID:            "due_date_select",
				Options:       []string{"Today", "Tomorrow", "In 3 Days", "In 1 Week"},
				State:         state.DueDateState,
				Style:         cell.Style{Fg: cell.NewColorRGB(200, 200, 200), Bg: cell.NewColorRGB(25, 25, 30)},
				SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: accentColor, Modifier: cell.ModifierBold},
				HoverStyle:    cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(45, 45, 50)},
			},
		}, formChunks[2])

		// Left Filters Block
		filterBorderCol := cell.NewColorRGB(60, 65, 80)
		if focused == "filter_list" {
			filterBorderCol = accentColor
		}
		filters := []string{"All", "Active", "Completed"}
		f.RenderWidget(widgets.Block{
			Title:         " FILTERS ",
			Borders:       widgets.BorderAll,
			BorderSymbols: widgets.SymbolsRounded,
			BorderStyle:   cell.Style{Fg: filterBorderCol},
			Child: widgets.List{
				ID:            "filter_list",
				Items:         filters,
				State:         state.FilterState,
				Style:         cell.Style{Fg: cell.NewColorRGB(180, 180, 190)},
				SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: accentColor, Modifier: cell.ModifierBold},
			},
		}, filterRect)

		// Right Tasks List Block
		taskBorderCol := cell.NewColorRGB(60, 65, 80)
		if focused == "task_list" {
			taskBorderCol = accentColor
		}

		filteredTasks := getFilteredTasks(state)
		taskItems := make([]string, len(filteredTasks))
		for i, t := range filteredTasks {
			check := "[ ]"
			if t.Completed {
				check = "[x]"
			}

			// Calculate remaining days
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			taskDate := time.Date(t.DueDate.Year(), t.DueDate.Month(), t.DueDate.Day(), 0, 0, 0, 0, t.DueDate.Location())

			days := int(taskDate.Sub(today).Hours() / 24)

			var dateTag string
			if t.Completed {
				dateTag = " (Completed) "
			} else {
				if days < 0 {
					dateTag = fmt.Sprintf(" [Overdue: %d d] ", -days)
				} else if days == 0 {
					dateTag = " [Today] "
				} else if days == 1 {
					dateTag = " [Tomorrow] "
				} else {
					dateTag = fmt.Sprintf(" [%d days left] ", days)
				}
			}

			taskItems[i] = fmt.Sprintf(" %s  %s  (%s)%s", check, t.Title, t.Category, dateTag)
		}

		listWidget := widgets.List{
			ID:            "task_list",
			Items:         taskItems,
			State:         state.ListState,
			Style:         cell.Style{Fg: cell.NewColorRGB(180, 180, 190)},
			SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(50, 60, 75), Modifier: cell.ModifierBold},
		}

		f.RenderWidget(widgets.Block{
			Title:         fmt.Sprintf(" TASK LIST (%s: %d) ", state.ActiveFilter, len(filteredTasks)),
			Borders:       widgets.BorderAll,
			BorderSymbols: widgets.SymbolsRounded,
			BorderStyle:   cell.Style{Fg: taskBorderCol},
			Child:         listWidget,
		}, listRect)

		// Footer & Progress Bar
		footerLay := layout.NewFlexLayout(
			layout.Vertical,
			0,
			layout.Fixed(2),
			layout.Fixed(1),
		)
		footerChunks := footerLay.Split(footerRect)

		// Progress bar
		f.RenderWidget(widgets.Block{
			Title:        fmt.Sprintf(" COMPLETION RATE: %%%d ", int(state.ProgressBar.Value)),
			Borders:      widgets.BorderNone,
			PaddingLeft:  1,
			PaddingRight: 1,
			Child:        state.ProgressBar,
		}, footerChunks[0])

		// Instructions
		instructions := " [Tab] Focus | [Space] Toggle Status | [d/Backspace] Delete | [?] Help | [q] Quit"
		f.RenderWidget(widgets.Block{
			Borders: widgets.BorderNone,
			Style:   cell.Style{Fg: cell.NewColorRGB(130, 130, 130)},
			Child:   text{value: instructions, style: cell.Style{Fg: cell.NewColorRGB(130, 130, 130)}},
		}, footerChunks[1])

		// Help Modal Popup Overlay
		if state.ShowHelp {
			helpW := uint16(58)
			helpH := uint16(12)
			helpArea := terminal.CenterRect(area, helpW, helpH)

			f.RegisterLayer("todo_help_dialog", terminal.LayerModal, helpArea, 3000, func() {
				state.ShowHelp = false
			})

			f.BeginLayer("todo_help_dialog")

			helpContent := "Limoni Task Manager Shortcuts:\n\n" +
				"  • [Tab]         : Cycle through interactive widgets.\n" +
				"  • [Arrow Up/Dn] : Browse task and filter lists.\n" +
				"  • [Space/Enter] : Toggle task completion status.\n" +
				"  • [d/Backspace] : Delete selected task permanently.\n" +
				"  • [?] or [h]    : Open this help manual.\n\n" +
				"Press any key to dismiss."

			f.RenderWidget(widgets.Block{
				Title:          " KEYBOARD SHORTCUTS & HELP ",
				TitleAlignment: widgets.AlignCenter,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsDouble,
				BorderStyle:    cell.Style{Fg: accentColor},
				Style:          cell.Style{Bg: cell.NewColorRGB(20, 25, 35)},
				PaddingLeft:    3,
				PaddingTop:     1,
				Child:          text{value: helpContent, style: cell.Style{Fg: cell.NewColorRGB(220, 220, 230)}},
			}, helpArea)
		}
	})
}
