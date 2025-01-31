package main

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/hako/durafmt"
	"github.com/ncruces/zenity"
)

type Contents interface {
	Update()
	Draw()
	Destroy()
	Trigger(int)
	ReceiveMessage(string)
}

type taskBGProgress struct {
	Current, Max int
	Task         *Task
	fillAmount   float32
}

func newTaskBGProgress(task *Task) *taskBGProgress {
	return &taskBGProgress{Task: task}
}

func (tbg *taskBGProgress) Draw() {

	rec := tbg.Task.Rect
	if tbg.Task.Board.Project.OutlineTasks.Checked {
		rec.Width -= 2
		rec.X++
		rec.Y++
		rec.Height -= 2
	}

	ratio := float32(0)

	if tbg.Current > 0 && tbg.Max > 0 {

		ratio = float32(tbg.Current) / float32(tbg.Max)

		if ratio > 1 {
			ratio = 1
		} else if ratio < 0 {
			ratio = 0
		}

	}

	tbg.fillAmount += (ratio - tbg.fillAmount) * 0.1
	rec.Width = tbg.fillAmount * rec.Width
	rl.DrawRectangleRec(rec, getThemeColor(GUI_INSIDE_HIGHLIGHTED))
}

func applyGlow(task *Task, color rl.Color) rl.Color {

	// if (task.Completable() && ((task.Complete() && task.Board.Project.CompleteTasksGlow.Checked) || (!task.Complete() && task.Board.Project.IncompleteTasksGlow.Checked))) || (task.Selected && task.Board.Project.SelectedTasksGlow.Checked) {
	if (task.IsCompletable() && ((task.Board.Project.CompleteTasksGlow.Checked) || (task.Board.Project.IncompleteTasksGlow.Checked))) || (task.Selected && task.Board.Project.SelectedTasksGlow.Checked) {

		glowVariance := float64(20)
		if task.Selected {
			glowVariance = 40
		}

		glow := int32(math.Sin(float64((rl.GetTime()*math.Pi*2-(float32(task.ID)*0.1))))*(glowVariance/2) + (glowVariance / 2))

		color = ColorAdd(color, -glow)
	}

	return color

}

func drawTaskBG(task *Task, fillColor rl.Color) {

	// task.Rect.Width = size.X
	// task.Rect.Height = size.Y

	outlineColor := getThemeColor(GUI_OUTLINE)

	if task.Selected {
		outlineColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
	} else if task.IsComplete() {
		outlineColor = getThemeColor(GUI_OUTLINE)
	}

	fillColor = applyGlow(task, fillColor)
	outlineColor = applyGlow(task, outlineColor)

	alpha := float32(task.Board.Project.TaskTransparency.Number()) / float32(task.Board.Project.TaskTransparency.Maximum)
	fillColor.A = uint8(float32(fillColor.A) * alpha)

	if task.Board.Project.OutlineTasks.Checked {
		rl.DrawRectangleRec(task.Rect, outlineColor)
		DrawRectExpanded(task.Rect, -1, fillColor)
	} else {
		rl.DrawRectangleRec(task.Rect, fillColor)
	}

	// Animate deadlines
	deadlineAnimation := task.Board.Project.DeadlineAnimation.CurrentChoice

	if task.IsCompletable() && task.DeadlineOn.Checked && !task.IsComplete() && deadlineAnimation < 4 {

		deadlineAlignment := deadlineAlignment(task)

		patternSrc := rl.Rectangle{task.Board.Project.Time * 16, 0, 16, 16}
		if deadlineAlignment < 0 {
			patternSrc.Y += 16
			patternSrc.X *= 4
		}
		patternSrc.Width = task.Rect.Width

		dst := task.Rect

		if task.Board.Project.OutlineTasks.Checked {
			patternSrc.X++
			patternSrc.Y++
			patternSrc.Width -= 2
			patternSrc.Height -= 2

			dst.X++
			dst.Y++
			dst.Width -= 2
			dst.Height -= 2

		}

		rl.DrawTexturePro(task.Board.Project.Patterns, patternSrc, dst, rl.Vector2{}, 0, getThemeColor(GUI_INSIDE_HIGHLIGHTED))

		if deadlineAnimation < 3 {
			src := rl.Rectangle{144, 0, 16, 16}
			dst := src
			dst.X = task.Rect.X - src.Width
			dst.Y = task.Rect.Y

			if deadlineAnimation == 0 || (deadlineAnimation == 1 && deadlineAlignment < 0) {
				dst.X += float32(math.Sin(float64(task.Board.Project.Time+((task.Rect.X+task.Rect.Y)*0.01))*math.Pi*2))*2 - 2
			}

			if deadlineAlignment == 0 {
				src.X += 16
			} else if deadlineAlignment < 0 {
				// Overdue!
				src.X += 32
			}

			rl.DrawTexturePro(task.Board.Project.GUI_Icons, src, dst, rl.Vector2{}, 0, rl.White)
		}

	}

}

func deadlineAlignment(task *Task) int {
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	targetDate := time.Date(task.DeadlineYear.Number(), time.Month(task.DeadlineMonth.CurrentChoice+1), task.DeadlineDay.Number(), 0, 0, 0, 0, now.Location())

	duration := targetDate.Sub(now).Truncate(time.Hour * 24)
	if duration.Seconds() > 0 {
		return 1
	} else if duration.Seconds() == 0 {
		return 0
	} else {
		return -1
	}
}

// DSTChange returns whether the timezone of the time given is different from now's timezone (i.e. from PST to PDT or vice-versa).
func DSTChange(startTime time.Time) bool {

	nowZone, _ := time.Now().Zone()
	startZone, _ := startTime.Zone()

	// Returns the offset amount of the difference between
	return nowZone != startZone

}

func deadlineText(task *Task) string {

	txt := ""

	if task.DeadlineOn.Checked && !task.IsComplete() {

		now := time.Now()
		now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		targetDate := time.Date(task.DeadlineYear.Number(), time.Month(task.DeadlineMonth.CurrentChoice+1), task.DeadlineDay.Number(), 0, 0, 0, 0, now.Location())

		// Don't truncate by time because it cuts off daylight savings time changes (where the time change date could be 23 or 25 hours, not just 24)
		duration := targetDate.Sub(now)

		if duration.Seconds() == 0 {
			txt += " : Due today"
		} else if duration.Seconds() > 0 {
			txt += " : Due in " + durafmt.Parse(duration).LimitFirstN(2).String()
		} else {
			txt += " : Overdue by " + durafmt.Parse(-duration).LimitFirstN(2).String() + "!"
		}

	}

	return txt

}

type CheckboxContents struct {
	Task          *Task
	bgProgress    *taskBGProgress
	URLButtons    *URLButtons
	TextSize      rl.Vector2
	DisplayedText string
}

func NewCheckboxContents(task *Task) *CheckboxContents {

	contents := &CheckboxContents{
		Task:       task,
		bgProgress: newTaskBGProgress(task),
		URLButtons: NewURLButtons(task),
	}

	return contents
}

// Update always runs, once per Content per Task for each Task on the currently viewed Board.
func (c *CheckboxContents) Update() {

	if c.Task.Selected && programSettings.Keybindings.On(KBCheckboxToggle) && c.Task.Board.Project.IsInNeutralState() {
		c.Trigger(TASK_TRIGGER_TOGGLE)
	}

}

// Draw only runs when the Task is visible.
func (c *CheckboxContents) Draw() {

	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

	cp := rl.Vector2{c.Task.Rect.X + 4, c.Task.Rect.Y}

	displaySize := rl.Vector2{32, 16}

	iconColor := getThemeColor(GUI_FONT_COLOR)

	isParent := len(c.Task.SubTasks) > 0
	completionCount, totalCount, completionRecCount, totalRecCount := c.Task.CountTotals()

	c.bgProgress.Current = 0
	c.bgProgress.Max = 1

	if isParent {
		c.bgProgress.Current = completionRecCount
		c.bgProgress.Max = totalRecCount
	} else if c.Task.IsComplete() {
		c.bgProgress.Current = 1
	}

	c.bgProgress.Draw()

	if c.Task.Board.Project.ShowIcons.Checked {

		srcIcon := rl.Rectangle{0, 0, 16, 16}

		if isParent {
			srcIcon.X = 128
			srcIcon.Y = 16
		}

		if c.Task.IsComplete() {
			srcIcon.X += 16
		}

		if c.Task.SmallButton(srcIcon.X, srcIcon.Y, 16, 16, c.Task.Rect.X, c.Task.Rect.Y) {
			c.Trigger(TASK_TRIGGER_TOGGLE)
			ConsumeMouseInput(rl.MouseLeftButton)
		}

		cp.X += 16

	}

	txt := c.Task.Description.Text()

	extendedText := false

	if strings.Contains(c.Task.Description.Text(), "\n") {
		extendedText = true
		txt = strings.Split(txt, "\n")[0]
	}

	// We want to scan the text before adding in the completion count or numerical prefixes, but after splitting for newlines as necessary
	c.URLButtons.ScanText(txt)

	if isParent {
		txt += fmt.Sprintf(" →%d/%d", completionCount, totalCount)
		if totalCount != totalRecCount {
			txt += fmt.Sprintf(" ↓%d/%d", completionRecCount, totalRecCount)
		}
	}

	if c.Task.PrefixText != "" {
		txt = c.Task.PrefixText + " " + txt
	}

	txt += deadlineText(c.Task)

	DrawText(cp, txt)

	if c.Task.PrefixText != "" {
		prefixSize, _ := TextSize(c.Task.PrefixText+" ", false)
		cp.X += prefixSize.X + 2
	}

	c.URLButtons.Draw(cp)

	if txt != c.DisplayedText {
		c.TextSize, _ = TextSize(txt, false)
		c.DisplayedText = txt
	}

	displaySize.X += c.TextSize.X

	if c.TextSize.Y > 0 {
		displaySize.Y = c.TextSize.Y
	}

	if extendedText {
		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, rl.Rectangle{112, 0, 16, 16}, rl.Rectangle{c.Task.Rect.X + displaySize.X - 12, cp.Y, 16, 16}, rl.Vector2{}, 0, iconColor)
		displaySize.X += 12
	}

	// We want to lock the size to the grid if possible
	displaySize = c.Task.Board.Project.RoundPositionToGrid(displaySize)

	if displaySize != c.Task.DisplaySize {
		c.Task.DisplaySize = displaySize
		c.Task.Board.TaskChanged = true
	}

}

func (c *CheckboxContents) Destroy() {}

func (c *CheckboxContents) ReceiveMessage(msg string) {

	if msg == MessageSettingsChange {
		c.DisplayedText = ""
	}

}

func (c *CheckboxContents) Trigger(trigger int) {

	if len(c.Task.SubTasks) == 0 {

		if trigger == TASK_TRIGGER_TOGGLE {
			c.Task.CompletionCheckbox.Checked = !c.Task.CompletionCheckbox.Checked
		} else if trigger == TASK_TRIGGER_SET {
			c.Task.CompletionCheckbox.Checked = true
		} else if trigger == TASK_TRIGGER_CLEAR {
			c.Task.CompletionCheckbox.Checked = false
		}

	} else {

		for _, task := range c.Task.SubTasks {

			if task.Contents != nil {

				task.Contents.Trigger(trigger)

			}

		}
	}

}

type ProgressionContents struct {
	Task          *Task
	bgProgress    *taskBGProgress
	URLButtons    *URLButtons
	DisplayedText string
	TextSize      rl.Vector2
}

func NewProgressionContents(task *Task) *ProgressionContents {

	contents := &ProgressionContents{
		Task:       task,
		bgProgress: newTaskBGProgress(task),
		URLButtons: NewURLButtons(task),
	}

	return contents

}

func (c *ProgressionContents) Update() {

	taskChanged := false

	if c.Task.Selected && c.Task.Board.Project.IsInNeutralState() {
		if programSettings.Keybindings.On(KBProgressToggle) {
			c.Trigger(TASK_TRIGGER_TOGGLE)
			taskChanged = true
		} else if programSettings.Keybindings.On(KBProgressUp) {
			c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionCurrent.Number() + 1)
			taskChanged = true
		} else if programSettings.Keybindings.On(KBProgressDown) {
			c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionCurrent.Number() - 1)
			taskChanged = true

		}
	}

	if taskChanged {
		c.Task.UndoChange = true
	}

}

func (c *ProgressionContents) Draw() {

	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

	c.bgProgress.Current = c.Task.CompletionProgressionCurrent.Number()
	c.bgProgress.Max = c.Task.CompletionProgressionMax.Number()
	c.bgProgress.Draw()

	cp := rl.Vector2{c.Task.Rect.X + 4, c.Task.Rect.Y}

	displaySize := rl.Vector2{48, 16}

	iconColor := getThemeColor(GUI_FONT_COLOR)

	if c.Task.Board.Project.ShowIcons.Checked {
		srcIcon := rl.Rectangle{32, 0, 16, 16}
		if c.Task.IsComplete() {
			srcIcon.X += 16
		}
		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, srcIcon, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, iconColor)
		cp.X += 16
		displaySize.X += 16
	}

	taskChanged := false

	if c.Task.Selected {

		if c.Task.SmallButton(112, 48, 16, 16, cp.X, cp.Y) {
			c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionCurrent.Number() - 1)
			ConsumeMouseInput(rl.MouseLeftButton)
			taskChanged = true
		}
		cp.X += 16

		if c.Task.SmallButton(96, 48, 16, 16, cp.X, cp.Y) {
			c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionCurrent.Number() + 1)
			ConsumeMouseInput(rl.MouseLeftButton)
			taskChanged = true
		}
		cp.X += 16

	}

	txt := c.Task.Description.Text()

	extendedText := false

	if strings.Contains(c.Task.Description.Text(), "\n") {
		extendedText = true
		txt = strings.Split(txt, "\n")[0]
	}

	c.URLButtons.ScanText(txt)

	if c.Task.PrefixText != "" {
		txt = c.Task.PrefixText + " " + txt
	}

	txt += fmt.Sprintf(" (%d/%d)", c.Task.CompletionProgressionCurrent.Number(), c.Task.CompletionProgressionMax.Number())

	cp.X += 4 // Give a bit more room before drawing the text

	txt += deadlineText(c.Task)

	if txt != c.DisplayedText {
		c.TextSize, _ = TextSize(txt, false)
		c.DisplayedText = txt
	}

	DrawText(cp, txt)

	if c.Task.PrefixText != "" {
		prefixSize, _ := TextSize(c.Task.PrefixText+" ", false)
		cp.X += prefixSize.X + 2
	}

	c.URLButtons.Draw(cp)

	displaySize.X += c.TextSize.X
	if c.TextSize.Y > 0 {
		displaySize.Y = c.TextSize.Y
	}

	if extendedText {
		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, rl.Rectangle{112, 0, 16, 16}, rl.Rectangle{c.Task.Rect.X + displaySize.X - 12, cp.Y, 16, 16}, rl.Vector2{}, 0, iconColor)
		displaySize.X += 12
	}

	// We want to lock the size to the grid if possible
	displaySize = c.Task.Board.Project.RoundPositionToGrid(displaySize)

	if displaySize != c.Task.DisplaySize {
		c.Task.DisplaySize = displaySize
		c.Task.Board.TaskChanged = true
	}

	if taskChanged {
		c.Task.UndoChange = true
	}

}

func (c *ProgressionContents) Destroy() {}

func (c *ProgressionContents) ReceiveMessage(msg string) {

	if msg == MessageSettingsChange {
		c.DisplayedText = ""
	}

}

func (c *ProgressionContents) Trigger(trigger int) {

	if len(c.Task.SubTasks) == 0 {

		if trigger == TASK_TRIGGER_TOGGLE {
			if c.Task.CompletionProgressionCurrent.Number() < c.Task.CompletionProgressionMax.Number() {
				c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionMax.Number())
			} else {
				c.Task.CompletionProgressionCurrent.SetNumber(0)
			}
		} else if trigger == TASK_TRIGGER_SET {
			c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionMax.Number())
		} else if trigger == TASK_TRIGGER_CLEAR {
			c.Task.CompletionProgressionCurrent.SetNumber(0)
		}

	}

}

type NoteContents struct {
	Task         *Task
	URLButtons   *URLButtons
	TextRenderer *TextRenderer
}

func NewNoteContents(task *Task) *NoteContents {

	contents := &NoteContents{
		Task:         task,
		URLButtons:   NewURLButtons(task),
		TextRenderer: NewTextRenderer(),
	}

	return contents

}

func (c *NoteContents) Update() {

	// This is here because we need it to set the size regardless of if it's onscreen or not
	c.TextRenderer.SetText(c.Task.Description.Text())

}

func (c *NoteContents) Draw() {

	drawTaskBG(c.Task, getThemeColor(GUI_NOTE_COLOR))

	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}

	displaySize := rl.Vector2{8, 16}

	iconColor := getThemeColor(GUI_FONT_COLOR)

	if c.Task.Board.Project.ShowIcons.Checked {
		srcIcon := rl.Rectangle{64, 0, 16, 16}
		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, srcIcon, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, iconColor)
		cp.X += 16
		displaySize.X += 16
	}

	cp.X += 2

	c.TextRenderer.Draw(cp)

	c.URLButtons.ScanText(c.Task.Description.Text())

	c.URLButtons.Draw(cp)

	displaySize.X += c.TextRenderer.Size.X
	if c.TextRenderer.Size.Y > 0 {
		displaySize.Y = c.TextRenderer.Size.Y
	}

	displaySize = c.Task.Board.Project.CeilingPositionToGrid(displaySize)

	if displaySize != c.Task.DisplaySize {
		c.Task.DisplaySize = displaySize
		c.Task.Board.TaskChanged = true
	}

}

func (c *NoteContents) Destroy() {

	c.TextRenderer.Destroy()

}

func (c *NoteContents) ReceiveMessage(msg string) {

	if msg == MessageSettingsChange {
		c.TextRenderer.RecreateTexture()
	}

}

func (c *NoteContents) Trigger(trigger int) {}

type ImageContents struct {
	Task            *Task
	Resource        *Resource
	GifPlayer       *GifPlayer
	LoadedPath      string
	DisplayedText   string
	TextSize        rl.Vector2
	ProgressBG      *taskBGProgress
	ResetSize       bool
	resizing        bool
	ChangedResource bool
}

func NewImageContents(task *Task) *ImageContents {

	contents := &ImageContents{
		Task:       task,
		ProgressBG: newTaskBGProgress(task),
	}

	contents.ProgressBG.Max = 100

	contents.LoadResource()

	return contents

}

func (c *ImageContents) Update() {

	if c.resizing && MouseReleased(rl.MouseLeftButton) {
		c.resizing = false
		c.Task.UndoChange = true
		c.Task.Board.TaskChanged = true // Have the board reorder if the size is different
	}

}

func (c *ImageContents) LoadResource() {

	if c.Task.Open {

		if c.Task.LoadMediaButton.Clicked {

			filepath := ""
			var err error

			patterns := []string{}
			patterns = append(patterns, PermutateCaseForString("png", "*.")...)
			patterns = append(patterns, PermutateCaseForString("bmp", "*.")...)
			patterns = append(patterns, PermutateCaseForString("jpeg", "*.")...)
			patterns = append(patterns, PermutateCaseForString("jpg", "*.")...)
			patterns = append(patterns, PermutateCaseForString("gif", "*.")...)
			patterns = append(patterns, PermutateCaseForString("dds", "*.")...)
			patterns = append(patterns, PermutateCaseForString("hdr", "*.")...)
			patterns = append(patterns, PermutateCaseForString("ktx", "*.")...)
			patterns = append(patterns, PermutateCaseForString("astc", "*.")...)

			filepath, err = zenity.SelectFile(zenity.Title("Select image file"), zenity.FileFilters{{Name: "Image File", Patterns: patterns}})

			if err == nil && filepath != "" {
				c.Task.FilePathTextbox.SetText(filepath)
			}

		}

		// Manually changed the image filepath by keyboard or by Load button
		if c.Task.FilePathTextbox.Changed {
			c.ChangedResource = true
		}

		if c.Task.ResetImageSizeButton.Clicked {

			if c.Resource != nil {

				if c.Resource.IsTexture() {
					c.Task.DisplaySize.X = float32(c.Resource.Texture().Width)
					c.Task.DisplaySize.Y = float32(c.Resource.Texture().Height)
				} else {
					c.Task.DisplaySize.X = float32(c.Resource.Gif().Width)
					c.Task.DisplaySize.Y = float32(c.Resource.Gif().Height)
				}

				c.Task.Board.TaskChanged = true

			} else {
				c.Task.Board.Project.Log("Cannot reset image size if it's invalid or loading.")
			}

		}

	}

	fp := c.Task.FilePathTextbox.Text()

	if !c.Task.Open && c.LoadedPath != fp {

		c.LoadedPath = fp

		newResource := c.Task.Board.Project.LoadResource(fp)

		if c.ChangedResource && newResource != c.Resource {
			c.ResetSize = true
		}

		c.ChangedResource = false
		c.Resource = newResource

	}

	if c.Resource != nil {

		if c.Resource.State() == RESOURCE_STATE_READY {

			if c.Resource.IsGif() && (c.GifPlayer == nil || c.GifPlayer.Animation != c.Resource.Gif()) {
				c.GifPlayer = NewGifPlayer(c.Resource.Gif())
			}

			if c.ResetSize {

				c.ResetSize = false

				valid := true

				width := float32(0)
				height := float32(0)

				if c.Resource.IsTexture() {
					width = float32(c.Resource.Texture().Width)
					height = float32(c.Resource.Texture().Height)
				} else if c.Resource.IsGif() {
					width = c.Resource.Gif().Width
					height = c.Resource.Gif().Height
				} else {
					valid = false
				}

				if valid {

					yAspectRatio := float32(height / width)
					xAspectRatio := float32(width / height)

					coverage := c.Task.Board.Project.ScreenSize.X / camera.Zoom * 0.25

					if width > height {
						c.Task.DisplaySize.X = coverage
						c.Task.DisplaySize.Y = coverage * yAspectRatio
					} else {
						c.Task.DisplaySize.X = coverage * xAspectRatio
						c.Task.DisplaySize.Y = coverage
					}

				} else {
					c.Resource = nil
					c.Task.Board.Project.Log("Cannot load file: [%s]\nAre you sure it's an image file?", c.Task.FilePathTextbox.Text())
				}

				c.Task.Board.TaskChanged = true

				c.Task.DisplaySize = c.Task.Board.Project.RoundPositionToGrid(c.Task.DisplaySize)

			}

		} else if c.Resource.State() == RESOURCE_STATE_DELETED {
			c.Resource = nil
			c.LoadedPath = ""
		}

	}

}

func (c *ImageContents) Draw() {

	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

	project := c.Task.Board.Project
	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}
	text := ""

	c.LoadResource()

	if c.Resource != nil {

		switch c.Resource.State() {

		case RESOURCE_STATE_READY:

			mp := GetWorldMousePosition()

			var tex rl.Texture2D

			if c.Resource.IsTexture() {
				tex = c.Resource.Texture()
			} else if c.Resource.IsGif() {
				tex = c.GifPlayer.GetTexture()
				c.GifPlayer.Update(project.AdjustedFrameTime())
			}

			pos := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}

			src := rl.Rectangle{0, 0, float32(tex.Width), float32(tex.Height)}
			dst := rl.Rectangle{c.Task.Rect.X, c.Task.Rect.Y, c.Task.Rect.Width, c.Task.Rect.Height}

			if project.OutlineTasks.Checked {
				src.X++
				src.Y++
				src.Width -= 2
				src.Height -= 2

				dst.X++
				dst.Y++
				dst.Width -= 2
				dst.Height -= 2
			}

			color := rl.White

			if project.GraphicalTasksTransparent.Checked {
				alpha := float32(project.TaskTransparency.Number()) / float32(project.TaskTransparency.Maximum)
				color.A = uint8(float32(color.A) * alpha)
			}
			rl.DrawTexturePro(tex, src, dst, rl.Vector2{}, 0, color)

			grabSize := float32(math.Min(float64(dst.Width), float64(dst.Height)) * 0.05)

			if c.Task.Selected && c.Task.Board.Project.IsInNeutralState() {

				// Draw resize controls

				if grabSize <= 5 {
					grabSize = float32(5)
				}

				corner := rl.Rectangle{pos.X + dst.Width - grabSize, pos.Y + dst.Height - grabSize, grabSize, grabSize}

				if MousePressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mp, corner) {
					c.resizing = true
					c.Task.DisplaySize.X = c.Task.Position.X + c.Task.DisplaySize.X
					c.Task.DisplaySize.Y = c.Task.Position.Y + c.Task.DisplaySize.Y
					c.Task.Board.SendMessage(MessageSelect, map[string]interface{}{"task": c.Task})
				}

				DrawRectExpanded(corner, 1, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))
				rl.DrawRectangleRec(corner, getThemeColor(GUI_INSIDE))

				// corners := []rl.Rectangle{
				// 	{pos.X, pos.Y, grabSize, grabSize},
				// 	{pos.X + dst.Width - grabSize, pos.Y, grabSize, grabSize},
				// 	{pos.X + dst.Width - grabSize, pos.Y + dst.Height - grabSize, grabSize, grabSize},
				// 	{pos.X, pos.Y + dst.Height - grabSize, grabSize, grabSize},
				// }

				// for i, corner := range corners {

				// 	if MousePressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mp, corner) {
				// 		c.resizingImage = true
				// 		c.grabbingCorner = i
				// 		c.bottomCorner.X = c.Task.Position.X + c.Task.DisplaySize.X
				// 		c.bottomCorner.Y = c.Task.Position.Y + c.Task.DisplaySize.Y
				// 		c.Task.Board.SendMessage(MessageSelect, map[string]interface{}{"task": c.Task})
				// 	}

				// 	rl.DrawRectangleRec(corner, rl.Black)

				// }

				if c.resizing {

					c.Task.Board.Project.Selecting = false

					c.Task.Dragging = false

					c.Task.DisplaySize.X = mp.X + (grabSize / 2) - c.Task.Position.X
					c.Task.DisplaySize.Y = mp.Y + (grabSize / 2) - c.Task.Position.Y

					if !programSettings.Keybindings.On(KBUnlockImageASR) {
						asr := float32(tex.Height) / float32(tex.Width)
						c.Task.DisplaySize.Y = c.Task.DisplaySize.X * asr
						// if c.grabbingCorner == 0 {
						// 	c.Task.Position.Y = c.Task.Position.X * asr
						// } else if c.grabbingCorner == 1 {
						// 	c.Task.Position.Y = c.bottomCorner.Y - (c.bottomCorner.X * asr)
						// } else if c.grabbingCorner == 2 {
						// 	c.bottomCorner.Y = c.bottomCorner.X * asr
						// } else {
						// c.bottomCorner.Y = c.bottomCorner.X * asr
						// }
					}

					if !programSettings.Keybindings.On(KBUnlockImageGrid) {
						c.Task.DisplaySize = project.RoundPositionToGrid(c.Task.DisplaySize)
						c.Task.Position = project.RoundPositionToGrid(c.Task.Position)
					}

					// c.Task.DisplaySize.X = c.bottomCorner.X - c.Task.Position.X
					// c.Task.DisplaySize.Y = c.bottomCorner.Y - c.Task.Position.Y

					c.Task.Rect.X = c.Task.Position.X
					c.Task.Rect.Y = c.Task.Position.Y
					c.Task.Rect.Width = c.Task.DisplaySize.X
					c.Task.Rect.Height = c.Task.DisplaySize.Y

				}

			}

		case RESOURCE_STATE_DOWNLOADING:
			// Some resources have no visible progress when downloading
			progress := c.Resource.Progress()
			if progress >= 0 {
				text = fmt.Sprintf("Downloading [%s]... [%d%%]", c.Resource.Filename(), progress)
				c.ProgressBG.Current = progress
				c.ProgressBG.Draw()
			} else {
				text = fmt.Sprintf("Downloading [%s]...", c.Resource.Filename())
			}

		case RESOURCE_STATE_LOADING:

			if FileExists(c.Resource.LocalFilepath) {
				text = fmt.Sprintf("Loading image [%s]... [%d%%]", c.Resource.Filename(), c.Resource.Progress())
				c.ProgressBG.Current = c.Resource.Progress()
				c.ProgressBG.Draw()
			} else {
				text = fmt.Sprintf("Non-existant image [%s]", c.Resource.Filename())
			}

		}

	} else {
		text = "No image loaded."
	}

	if text != "" {
		c.Task.TempDisplaySize = rl.Vector2{16, 16}
		if project.ShowIcons.Checked {
			rl.DrawTexturePro(project.GUI_Icons, rl.Rectangle{96, 0, 16, 16}, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, getThemeColor(GUI_FONT_COLOR))
			cp.X += 16
			c.Task.TempDisplaySize.X += 16
		}

		DrawText(cp, text)

		if text != c.DisplayedText {
			c.TextSize, _ = TextSize(text, false)
			c.DisplayedText = text
		}

		c.Task.TempDisplaySize.X += c.TextSize.X

		c.Task.TempDisplaySize = c.Task.Board.Project.RoundPositionToGrid(c.Task.TempDisplaySize)

	}

	if c.Task.DisplaySize.X < 16 {
		c.Task.DisplaySize.X = 16
	}
	if c.Task.DisplaySize.Y < 16 {
		c.Task.DisplaySize.Y = 16
	}

}

func (c *ImageContents) Destroy() {

	if c.GifPlayer != nil {
		c.GifPlayer.Destroy()
	}

}

func (c *ImageContents) ReceiveMessage(msg string) {}

func (c *ImageContents) Trigger(trigger int) {}

type TimerContents struct {
	Task          *Task
	TimerValue    float32
	TargetDate    time.Time
	TextSize      rl.Vector2
	DisplayedText string
	Initialized   bool
}

func NewTimerContents(task *Task) *TimerContents {

	contents := &TimerContents{
		Task: task,
	}

	contents.CalculateTimeLeft() // Attempt to set the time on creation

	return contents
}

func (c *TimerContents) CalculateTimeLeft() {

	now := time.Now()

	switch c.Task.TimerMode.CurrentChoice {

	case TIMER_TYPE_COUNTDOWN:
		// We check to see if the countdown GUI elements have changed because otherwise having the Task open to, say,
		// edit the Timer Name would effectively pause the timer as the value would always be set.
		if c.Task.TimerMode.Changed || !c.Initialized || c.Task.CountdownMinute.Changed || c.Task.CountdownSecond.Changed || !c.Task.TimerRunning || c.Task.Board.Project.Loading {
			c.TimerValue = float32(c.Task.CountdownMinute.Number()*60 + c.Task.CountdownSecond.Number())
		}
		c.TargetDate = time.Time{}

	case TIMER_TYPE_DAILY:

		// Get a solid start that is the beginning of the week. nextDate starts as today, minus how far into the week we are
		weekStart := time.Date(now.Year(), now.Month(), now.Day()-int(now.Weekday()), c.Task.DailyHour.Number(), c.Task.DailyMinute.Number(), 0, 0, now.Location())

		nextDate := time.Time{}

		// Calculate when the next time the Timer should go off is (i.e. a Timer could go off multiple days, so we check each valid day).
		for dayIndex, enabled := range c.Task.DailyDay.EnabledOptionsAsArray() {

			if !enabled {
				continue
			}

			day := weekStart.AddDate(0, 0, dayIndex)

			if nextDate.IsZero() || day.After(nextDate) {
				nextDate = day
			}

		}

		if !nextDate.After(now) {
			nextDate = nextDate.AddDate(0, 0, 7)
		}

		c.TargetDate = nextDate

	case TIMER_TYPE_DATE:

		c.TargetDate = time.Date(c.Task.DeadlineYear.Number(), time.Month(c.Task.DeadlineMonth.CurrentChoice+1), c.Task.DeadlineDay.Number(), 23, 59, 59, 0, now.Location())

	case TIMER_TYPE_STOPWATCH:

		if c.Task.TimerMode.Changed {
			c.TimerValue = 0
		}

	}

}

func (c *TimerContents) Update() {

	c.Initialized = true // This is here to allow for deserializing Tasks to undo or redo correctly, as Deserializing recreates the contents of a Task

	if c.Task.Open {
		c.CalculateTimeLeft()
	}

	if c.Task.TimerRunning {

		now := time.Now()

		switch c.Task.TimerMode.CurrentChoice {

		case TIMER_TYPE_STOPWATCH:
			c.TimerValue += deltaTime // Stopwatches count up because they have no limit; we're using raw delta time because we want it to count regardless of what's going on
		default:

			if c.TargetDate.IsZero() {
				c.TimerValue -= deltaTime // We count down, not up, otherwise
			} else {
				c.TimerValue = float32(c.TargetDate.Sub(now).Seconds())
			}

			if c.TimerValue <= 0 {

				c.Task.TimerRunning = false
				c.TimeUp()
				c.CalculateTimeLeft()

				if c.Task.TimerRepeating.Checked && c.Task.TimerMode.CurrentChoice != TIMER_TYPE_DATE {
					c.Trigger(TASK_TRIGGER_SET)
				}

			}

		}

	}

	if c.Task.Selected && programSettings.Keybindings.On(KBStartTimer) && c.Task.Board.Project.IsInNeutralState() {
		c.Trigger(TASK_TRIGGER_TOGGLE)
	}

}

func (c *TimerContents) TimeUp() {

	project := c.Task.Board.Project

	project.Log("Timer [%s] went off.", c.Task.TimerName.Text())

	if c.Task.TimerTriggerMode.CurrentChoice != TASK_TRIGGER_NONE {

		triggeredTasks := []*Task{}

		alreadyTriggered := func(task *Task) bool {
			for _, t := range triggeredTasks {
				if t == task {
					return true
				}
			}
			return false
		}

		var triggerNeighbor func(neighbor *Task)

		triggerNeighbor = func(neighbor *Task) {

			if alreadyTriggered(neighbor) {
				return
			}

			triggeredTasks = append(triggeredTasks, neighbor)

			if neighbor.Is(TASK_TYPE_LINE) {

				for _, ending := range neighbor.LineEndings {

					if pointingTo := ending.Contents.(*LineContents).PointingTo; pointingTo != nil {
						triggerNeighbor(pointingTo)
					}

				}

			} else if neighbor.Contents != nil {

				// We have to capture a state of the item before triggering, otherwise we can't really undo it
				neighbor.Board.UndoHistory.Capture(NewUndoState(neighbor), true)

				neighbor.Contents.Trigger(c.Task.TimerTriggerMode.CurrentChoice)

				effect := "set"
				if c.Task.TimerTriggerMode.CurrentChoice == TASK_TRIGGER_TOGGLE {
					effect = "toggled"
				} else if c.Task.TimerTriggerMode.CurrentChoice == TASK_TRIGGER_CLEAR {
					effect = "un-set"
				}

				project.Log("Timer [%s] %s Task at [%d, %d].", c.Task.TimerName.Text(), effect, int32(neighbor.Position.X), int32(neighbor.Position.Y))
			}

		}

		if c.Task.TaskBelow != nil {
			triggerNeighbor(c.Task.TaskBelow)
		}

		if c.Task.TaskAbove != nil && !c.Task.TaskAbove.Is(TASK_TYPE_TIMER) {
			triggerNeighbor(c.Task.TaskAbove)
		}

		if c.Task.TaskRight != nil && !c.Task.TaskRight.Is(TASK_TYPE_TIMER) {
			triggerNeighbor(c.Task.TaskRight)
		}

		if c.Task.TaskLeft != nil && !c.Task.TaskLeft.Is(TASK_TYPE_TIMER) {
			triggerNeighbor(c.Task.TaskLeft)
		}

		if c.Task.TaskUnder != nil && !c.Task.TaskUnder.Is(TASK_TYPE_TIMER) {
			triggerNeighbor(c.Task.TaskUnder)
		}

	}

}

func (c *TimerContents) FormatText(minutes, seconds, milliseconds int) string {

	if milliseconds < 0 {
		return fmt.Sprintf("%02d:%02d", minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d:%02d", minutes, seconds, milliseconds)

}

func (c *TimerContents) Draw() {

	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

	project := c.Task.Board.Project
	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}

	displaySize := rl.Vector2{48, 16}

	if project.ShowIcons.Checked {
		rl.DrawTexturePro(project.GUI_Icons, rl.Rectangle{0, 16, 16, 16}, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, getThemeColor(GUI_FONT_COLOR))
		cp.X += 16
		displaySize.X += 16
	}

	srcX := float32(16)
	if c.Task.TimerRunning {
		srcX += 16
	}

	if c.Task.SmallButton(srcX, 16, 16, 16, cp.X, cp.Y) {
		c.Trigger(TASK_TRIGGER_TOGGLE)
		ConsumeMouseInput(rl.MouseLeftButton)
	}

	cp.X += 16

	if c.Task.SmallButton(48, 16, 16, 16, cp.X, cp.Y) {
		c.CalculateTimeLeft()
		ConsumeMouseInput(rl.MouseLeftButton)
		if c.Task.TimerMode.CurrentChoice == TIMER_TYPE_STOPWATCH {
			c.TimerValue = 0
		}
	}

	cp.X += 20 // Give a bit more room for the text

	text := c.Task.TimerName.Text() + " : "

	switch c.Task.TimerMode.CurrentChoice {

	case TIMER_TYPE_COUNTDOWN:

		time := int(c.TimerValue)
		minutes := time / 60
		seconds := time - (minutes * 60)

		currentTime := c.FormatText(minutes, seconds, -1)
		maxTime := c.FormatText(c.Task.CountdownMinute.Number(), c.Task.CountdownSecond.Number(), -1)

		text += currentTime + " / " + maxTime

	case TIMER_TYPE_DAILY:
		fallthrough
	case TIMER_TYPE_DATE:

		targetDateText := c.TargetDate.Format(" (Jan 2 2006)")

		if c.Task.TimerRunning {

			text += durafmt.Parse(time.Duration(c.TimerValue)*time.Second).LimitFirstN(2).String() + targetDateText

			if DSTChange(c.TargetDate) {
				text += " (DST change)"
			}
		} else {
			text += "Timer stopped."
		}

	case TIMER_TYPE_STOPWATCH:
		time := int(c.TimerValue * 100)
		minutes := time / 100 / 60
		seconds := time/100 - (minutes * 60)
		milliseconds := (time - (minutes * 6000) - (seconds * 100))

		currentTime := c.FormatText(minutes, seconds, milliseconds)

		text += currentTime
	}

	if text != "" {
		DrawText(cp, text)
		if text != c.DisplayedText {
			c.TextSize, _ = TextSize(text, false)
			c.DisplayedText = text
		}
		displaySize.X += c.TextSize.X
	}

	if displaySize.X < 16 {
		displaySize.X = 16
	}
	if displaySize.Y < 16 {
		displaySize.Y = 16
	}

	displaySize = c.Task.Board.Project.RoundPositionToGrid(displaySize)

	if displaySize != c.Task.DisplaySize {
		c.Task.DisplaySize = displaySize
		c.Task.Board.TaskChanged = true
	}

}

func (c *TimerContents) Destroy() {}

func (c *TimerContents) ReceiveMessage(msg string) {

	if msg == MessageSettingsChange {

		c.DisplayedText = ""

	} else if msg == MessageTaskDeserialization {
		// If undo or redo, recalculate the time left.
		c.CalculateTimeLeft()
	}

}

func (c *TimerContents) Trigger(trigger int) {

	if c.Task.TimerMode.CurrentChoice == TIMER_TYPE_STOPWATCH || c.TimerValue > 0 || !c.TargetDate.IsZero() {
		if trigger == TASK_TRIGGER_TOGGLE {
			c.Task.TimerRunning = !c.Task.TimerRunning
		} else if trigger == TASK_TRIGGER_SET {
			c.Task.TimerRunning = true
		} else if trigger == TASK_TRIGGER_CLEAR {
			c.Task.TimerRunning = false
		}

		c.Task.UndoChange = true
	}

}

type LineContents struct {
	Task       *Task
	PointingTo *Task
}

func NewLineContents(task *Task) *LineContents {
	return &LineContents{
		Task: task,
	}
}

func (c *LineContents) Update() {

	cycleDirection := 0

	if c.Task.Board.Project.IsInNeutralState() {

		if programSettings.Keybindings.On(KBSelectNextLineEnding) {
			cycleDirection = 1
		} else if programSettings.Keybindings.On(KBSelectPrevLineEnding) {
			cycleDirection = -1
		}

	}

	if c.Task.LineStart == nil && cycleDirection != 0 {

		selections := []*Task{}

		for _, ending := range c.Task.LineEndings {
			selections = append(selections, ending)
		}

		sort.Slice(selections, func(i, j int) bool {
			ba := selections[i]
			bb := selections[j]
			if ba.Position.Y != bb.Position.Y {
				return ba.Position.Y < bb.Position.Y
			}
			return ba.Position.X < bb.Position.X
		})

		selections = append([]*Task{c.Task}, selections...)

		for i, selection := range selections {

			if selection.Selected {

				var nextTask *Task

				if cycleDirection > 0 {

					if i < len(selections)-1 {
						nextTask = selections[i+1]
					} else {
						nextTask = selections[0]
					}

				} else {

					if i > 0 {
						nextTask = selections[i-1]
					} else {
						nextTask = selections[len(selections)-1]
					}

				}

				board := c.Task.Board
				board.SendMessage(MessageSelect, map[string]interface{}{"task": nextTask})
				board.FocusViewOnSelectedTasks()

				break

			}

		}

	}

}

func (c *LineContents) DrawLines() {
	if c.Task.LineStart != nil {
		outlinesOn := c.Task.Board.Project.OutlineTasks.Checked
		outlineColor := getThemeColor(GUI_INSIDE)
		fillColor := getThemeColor(GUI_FONT_COLOR)
		cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}
		cp.X += c.Task.Rect.Width / 2
		cp.Y += c.Task.Rect.Height / 2
		ep := rl.Vector2{c.Task.LineStart.Rect.X, c.Task.LineStart.Rect.Y}
		ep.X += c.Task.LineStart.Rect.Width / 2
		ep.Y += c.Task.LineStart.Rect.Height / 2
		if c.Task.LineStart.LineBezier.Checked {
			if outlinesOn {
				rl.DrawLineBezier(cp, ep, 4, outlineColor)
			}
			rl.DrawLineBezier(cp, ep, 2, fillColor)
		} else {
			if outlinesOn {
				rl.DrawLineEx(cp, ep, 4, outlineColor)
			}
			rl.DrawLineEx(cp, ep, 2, fillColor)
		}
		if c.Task.Selected {
			txt := strconv.Itoa(int(rl.Vector2Angle(cp, ep))) + "°"
			sz, _ := TextSize(txt, false)
			r := rl.Rectangle{cp.X + (ep.X - cp.X) / 2, cp.Y + (ep.Y - cp.Y) / 2, sz.X, sz.Y}
			DrawRectExpanded(r, 1, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))
			rl.DrawRectangleRec(r, getThemeColor(GUI_INSIDE))
			DrawText(rl.Vector2{r.X, r.Y}, txt)
		}
	}
}

func (c *LineContents) Draw() {

	outlinesOn := c.Task.Board.Project.OutlineTasks.Checked
	outlineColor := getThemeColor(GUI_INSIDE)
	fillColor := getThemeColor(GUI_FONT_COLOR)

	guiIcons := c.Task.Board.Project.GUI_Icons

	src := rl.Rectangle{128, 32, 16, 16}
	dst := rl.Rectangle{c.Task.Rect.X + (src.Width / 2), c.Task.Rect.Y + (src.Height / 2), src.Width, src.Height}

	rotation := float32(0)

	var noDraw bool
	if c.Task.LineStart != nil {

		src.X += 16

		c.PointingTo = nil

		if c.Task.TaskUnder != nil {
			src.X += 16
			rotation = 0
			c.PointingTo = c.Task.TaskUnder
		} else if c.Task.TaskBelow != nil && c.Task.TaskBelow != c.Task.LineStart {
			rotation += 90
			c.PointingTo = c.Task.TaskBelow
		} else if c.Task.TaskLeft != nil && c.Task.TaskLeft != c.Task.LineStart {
			rotation += 180
			c.PointingTo = c.Task.TaskLeft
		} else if c.Task.TaskAbove != nil && c.Task.TaskAbove != c.Task.LineStart {
			rotation -= 90
			c.PointingTo = c.Task.TaskAbove
		} else if c.Task.TaskRight != nil && c.Task.TaskRight != c.Task.LineStart {
			c.PointingTo = c.Task.TaskRight
		} else {
			angle := rl.Vector2Angle(c.Task.LineStart.Position, c.Task.Position)
			rotation = angle
		}

		noDraw = c.Task.LineStart.LineHeads.Checked
	} else {
		noDraw = c.Task.LineHeads.Checked
	}

	if outlinesOn && !noDraw {
		rl.DrawTexturePro(guiIcons, src, dst, rl.Vector2{src.Width / 2, src.Height / 2}, rotation, outlineColor)
	}

	src.Y += 16

	if !noDraw {
		rl.DrawTexturePro(guiIcons, src, dst, rl.Vector2{src.Width / 2, src.Height / 2}, rotation, fillColor)
	}

	c.Task.DisplaySize.X = 16
	c.Task.DisplaySize.Y = 16

}

func (c *LineContents) Trigger(triggerMode int) {}

func (c *LineContents) Destroy() {

	if c.Task.LineStart != nil {

		for index, ending := range c.Task.LineStart.LineEndings {
			if ending == c.Task {
				c.Task.LineStart.LineEndings = append(c.Task.LineStart.LineEndings[:index], c.Task.LineStart.LineEndings[index+1:]...)
				break
			}
		}

	} else {

		existingEndings := c.Task.LineEndings[:]

		c.Task.LineEndings = []*Task{}

		for _, ending := range existingEndings {
			ending.Board.DeleteTask(ending)
		}

		c.Task.UndoChange = false

	}

}

func (c *LineContents) ReceiveMessage(msg string) {

	if msg == MessageTaskDeserialization {

		if c.Task.LineStart == nil && !c.Task.Is(TASK_TYPE_LINE) {
			c.Destroy()
		}

	}

}

type MapContents struct {
	Task     *Task
	resizing bool
}

func NewMapContents(task *Task) *MapContents {

	return &MapContents{
		Task: task,
	}

}

func (c *MapContents) Update() {

	if c.resizing && MouseReleased(rl.MouseLeftButton) {
		c.resizing = false
		c.Task.UndoChange = true
		c.Task.Board.TaskChanged = true
	}

	if c.Task.MapImage == nil {

		c.Task.MapImage = NewMapImage(c.Task)
		c.Task.DisplaySize.X = c.Task.MapImage.Width()
		c.Task.DisplaySize.Y = c.Task.MapImage.Height() + float32(c.Task.Board.Project.GridSize)

	}

}

func (c *MapContents) Draw() {

	rl.DrawRectangleRec(c.Task.Rect, rl.Color{0, 0, 0, 64})

	bgColor := getThemeColor(GUI_INSIDE)

	if c.Task.MapImage.EditTool != MapEditToolNone {
		bgColor = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
		c.Task.Dragging = false
	}

	// Draw Map header
	oldHeight := c.Task.Rect.Height
	c.Task.Rect.Height = 16
	drawTaskBG(c.Task, bgColor)
	c.Task.Rect.Height = oldHeight

	project := c.Task.Board.Project
	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}

	if project.ShowIcons.Checked {
		rl.DrawTexturePro(project.GUI_Icons, rl.Rectangle{0, 32, 16, 16}, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, getThemeColor(GUI_FONT_COLOR))
		cp.X += 16
	}

	if c.Task.MapImage != nil {

		c.Task.Locked = c.Task.MapImage.EditTool != MapEditToolNone || c.resizing

		grabSize := float32(8)

		corner := rl.Rectangle{c.Task.Rect.X + c.Task.Rect.Width - grabSize, c.Task.Rect.Y + c.Task.Rect.Height - grabSize, grabSize, grabSize}

		if c.Task.Selected {

			mp := GetWorldMousePosition()

			if MousePressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mp, corner) {
				c.resizing = true
			}

			DrawRectExpanded(corner, 1, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))
			rl.DrawRectangleRec(corner, getThemeColor(GUI_INSIDE))

			if c.resizing {

				c.Task.MapImage.EditTool = MapEditToolNone

				c.Task.Board.Project.Selecting = false

				mp.X += 4
				mp.Y -= 4

				c.Task.MapImage.Resize(mp.X+(grabSize/2)-c.Task.Position.X, mp.Y+(grabSize/2)-c.Task.Position.Y)

			}

		}

		if c.Task.Locked {
			c.Task.Dragging = false
		}

		texture := c.Task.MapImage.Texture.Texture
		src := rl.Rectangle{0, 0, 512, 512}
		dst := rl.Rectangle{c.Task.Rect.X, c.Task.Rect.Y + 16, float32(texture.Width), float32(texture.Height)}
		src.Height *= -1

		rl.DrawTexturePro(texture, src, dst, rl.Vector2{}, 0, rl.White)

		// We call MapImage.Draw() after drawing the texture from the map image because MapImage.Draw() handles drawing
		// the selection rectangle as well
		c.Task.MapImage.Draw()

		// Shadow underneath the map header
		src = rl.Rectangle{216, 16, 8, 8}
		dst = rl.Rectangle{c.Task.Rect.X + 1, c.Task.Rect.Y + 16, c.Task.Rect.Width - 2, 8}
		shadowColor := rl.Black
		shadowColor.A = 128
		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, src, dst, rl.Vector2{}, 0, shadowColor)

		if c.Task.Selected {
			DrawRectExpanded(corner, 1, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))
			rl.DrawRectangleRec(corner, getThemeColor(GUI_INSIDE))
		}

		c.Task.DisplaySize.X = c.Task.MapImage.Width()
		c.Task.DisplaySize.Y = c.Task.MapImage.Height() + 16

	}

}

func (c *MapContents) Destroy() {}

func (c *MapContents) ReceiveMessage(msg string) {}

func (c *MapContents) Trigger(triggerMode int) {}

type WhiteboardContents struct {
	Task     *Task
	resizing bool
}

func NewWhiteboardContents(task *Task) *WhiteboardContents {
	return &WhiteboardContents{
		Task: task,
	}
}

func (c *WhiteboardContents) Update() {

	if c.resizing && MouseReleased(rl.MouseLeftButton) {
		c.resizing = false
		c.Task.UndoChange = true
		c.Task.Board.TaskChanged = true
	}

	if c.Task.Whiteboard == nil {

		c.Task.Whiteboard = NewWhiteboard(c.Task)
		c.Task.DisplaySize.X = float32(c.Task.Whiteboard.Width)
		c.Task.DisplaySize.Y = float32(c.Task.Whiteboard.Height) + float32(c.Task.Board.Project.GridSize)

	}

}

func (c *WhiteboardContents) Draw() {

	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}
	project := c.Task.Board.Project

	if project.ShowIcons.Checked {
		rl.DrawTexturePro(project.GUI_Icons, rl.Rectangle{64, 16, 16, 16}, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, getThemeColor(GUI_FONT_COLOR))
	}

	if c.Task.Whiteboard != nil {

		c.Task.Whiteboard.Draw()

		gs := float32(project.GridSize)

		texture := c.Task.Whiteboard.Texture.Texture
		src := rl.Rectangle{0, 0, float32(texture.Width), float32(texture.Height)}
		dst := rl.Rectangle{c.Task.Rect.X + 1, c.Task.Rect.Y + 16 + 1, src.Width - 2, src.Height - 2}
		src.Height *= -1

		rl.DrawTexturePro(texture, src, dst, rl.Vector2{}, 0, rl.White)

		if c.Task.Selected {

			mp := GetWorldMousePosition()

			grabSize := float32(8)

			corner := rl.Rectangle{c.Task.Rect.X + c.Task.Rect.Width - grabSize, c.Task.Rect.Y + c.Task.Rect.Height - grabSize, grabSize, grabSize}

			if MousePressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mp, corner) {
				c.resizing = true
			}

			DrawRectExpanded(corner, 1, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))
			rl.DrawRectangleRec(corner, getThemeColor(GUI_INSIDE))

			if c.resizing {

				c.Task.Whiteboard.Editing = false
				c.Task.Board.Project.Selecting = false

				mp.X += 4
				mp.Y -= 4

				c.Task.Whiteboard.Resize(mp.X+(grabSize/2)-c.Task.Position.X, mp.Y+(grabSize/2)-c.Task.Position.Y-gs)

			}

		}

		c.Task.DisplaySize.X = float32(c.Task.Whiteboard.Width)
		c.Task.DisplaySize.Y = float32(c.Task.Whiteboard.Height) + gs

	}

	c.Task.Locked = c.Task.Whiteboard.Editing || c.resizing

	// Shadow underneath the whiteboard header
	src := rl.Rectangle{216, 16, 8, 8}
	dst := rl.Rectangle{c.Task.Rect.X + 1, c.Task.Rect.Y + 16, c.Task.Rect.Width - 2, 8}
	shadowColor := rl.Black
	shadowColor.A = 128
	rl.DrawTexturePro(project.GUI_Icons, src, dst, rl.Vector2{}, 0, shadowColor)

}

func (c *WhiteboardContents) Destroy() {}

func (c *WhiteboardContents) Trigger(triggerMode int) {

	if triggerMode == TASK_TRIGGER_TOGGLE {
		c.Task.Whiteboard.Invert()
	} else if triggerMode == TASK_TRIGGER_SET {
		c.Task.Whiteboard.Clear()
		c.Task.Whiteboard.Invert()
	} else if triggerMode == TASK_TRIGGER_CLEAR {
		c.Task.Whiteboard.Clear()
	}

}

func (c *WhiteboardContents) ReceiveMessage(msg string) {

	if msg == MessageThemeChange {
		c.Task.Whiteboard.Deserialize(c.Task.Whiteboard.Serialize())
	}

}

type TableContents struct {
	Task           *Task
	RenderTexture  rl.RenderTexture2D
	StripesPattern rl.Texture2D
}

func NewTableContents(task *Task) *TableContents {

	res := task.Board.Project.LoadResource(LocalPath("assets", "diagonal_stripes.png")).Texture()

	return &TableContents{
		Task: task,
		// For some reason, smaller heights mess up the size of the rendering???
		RenderTexture:  rl.LoadRenderTexture(128, 128),
		StripesPattern: res,
	}

}

func (c *TableContents) Update() {

	if c.Task.TableData == nil {
		c.Task.TableData = NewTableData(c.Task)
	}

	c.Task.TableData.Update()

}

func (c *TableContents) Draw() {

	createUndo := false

	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE_DISABLED))

	if c.Task.TableData != nil {

		gs := float32(c.Task.Board.Project.GridSize)

		displaySize := rl.Vector2{gs * float32(len(c.Task.TableData.Columns)+1), gs * float32(len(c.Task.TableData.Rows)+1)}

		longestX := float32(0)
		longestY := float32(0)

		for _, element := range c.Task.TableData.Rows {

			if len(element.Textbox.Text()) > 0 {

				size, _ := TextSize(element.Textbox.Text(), false)
				if size.X > longestX {
					longestX = size.X
				}

			}

		}

		for _, element := range c.Task.TableData.Columns {

			if len(element.Textbox.Text()) > 0 {

				if c.Task.Board.Project.TableColumnsRotatedVertical.Checked {

					lineSpacing = float32(c.Task.Board.Project.TableColumnVerticalSpacing.Number()) / 100

					size, _ := TextHeight(element.TextVertically(), false)

					if size > longestY {
						longestY = size
					}

					lineSpacing = 1

				} else {

					size, _ := TextSize(element.Textbox.Text(), false)

					if size.X > longestY {
						longestY = size.X
					}

				}

			}

		}

		locked := c.Task.Board.Project.RoundPositionToGrid(rl.Vector2{longestX, longestY})

		longestX = locked.X
		longestY = locked.Y

		displaySize.X += longestX
		displaySize.Y += longestY

		pos := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}
		pos.Y += gs + longestY

		for i, element := range c.Task.TableData.Rows {

			rec := rl.Rectangle{pos.X + 1, pos.Y, longestX + gs - 1, gs}

			color := getThemeColor(GUI_NOTE_COLOR)
			if c.Task.IsComplete() {
				color = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
			}

			if i%2 == 1 {
				if IsColorLight(color) {
					color = ColorAdd(color, -20)
				} else {
					color = ColorAdd(color, 20)
				}
			}

			color = applyGlow(c.Task, color)

			if i >= len(c.Task.TableData.Rows)-1 {
				rec.Height--
			}

			rl.DrawRectangleRec(rec, color)

			DrawText(rl.Vector2{pos.X + 2, pos.Y + 2}, element.Textbox.Text())
			pos.Y += rec.Height
		}

		pos = rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}
		pos.X += gs + longestX

		for i, element := range c.Task.TableData.Columns {

			rec := rl.Rectangle{pos.X, pos.Y + 1, gs, longestY + gs - 1}

			color := getThemeColor(GUI_INSIDE)

			if i%2 == 1 {
				if IsColorLight(color) {
					color = ColorAdd(color, -20)
				} else {
					color = ColorAdd(color, 20)
				}
			}

			if c.Task.IsComplete() {
				color = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
			}

			color = applyGlow(c.Task, color)

			if i >= len(c.Task.TableData.Columns)-1 {
				rec.Width--
			}

			rl.DrawRectangleRec(rec, color)

			if c.Task.Board.Project.TableColumnsRotatedVertical.Checked {

				lineSpacing = float32(c.Task.Board.Project.TableColumnVerticalSpacing.Number()) / 100

				p := pos
				// p.X += gs / 4
				text := element.TextVertically()
				width := rl.MeasureTextEx(font, text, float32(programSettings.FontSize), spacing)
				p.X += gs/2 - width.X/2
				DrawText(p, text)

				lineSpacing = 1 // Can't forget to set line spacing back SPECIFICALLY for drawing the text

			} else {

				rl.EndMode2D()

				rl.BeginTextureMode(c.RenderTexture)
				rl.ClearBackground(rl.Color{0, 0, 0, 0})
				DrawText(rl.Vector2{1, 0}, element.Textbox.Text())
				rl.EndTextureMode()

				rl.BeginMode2D(camera)

				src := rl.Rectangle{0, 0, float32(c.RenderTexture.Texture.Width), float32(c.RenderTexture.Texture.Height)}
				dst := rl.Rectangle{pos.X + gs/2 - 2, pos.Y + gs/2 + 2, src.Width, src.Height}
				src.Height *= -1

				rl.DrawTexturePro(c.RenderTexture.Texture, src, dst, rl.Vector2{gs / 2, gs / 2}, 90, rl.White)

			}

			pos.X += gs

		}

		gridWidth := float32(len(c.Task.TableData.Columns)) * gs
		gridHeight := float32(len(c.Task.TableData.Rows)) * gs

		pos = rl.Vector2{c.Task.Rect.X + c.Task.Rect.Width - gridWidth, c.Task.Rect.Y + c.Task.Rect.Height - gridHeight}

		src := rl.Rectangle{0, 64, 16, 16}
		dst := rl.Rectangle{pos.X, pos.Y, 16, 16}

		worldGUI = true

		lockTask := false

		for y := range c.Task.TableData.Completions {

			for x := range c.Task.TableData.Completions[y] {

				value := c.Task.TableData.Completions[y][x]
				dst.X = pos.X + (float32(x) * gs)
				dst.Y = pos.Y + (float32(y) * gs)

				if value == 0 {
					src.X = 0
				} else if value == 1 {
					src.X = 16
				} else {
					src.X = 32
				}

				if rl.CheckCollisionPointRec(GetWorldMousePosition(), dst) {
					lockTask = true
				}

				style := NewButtonStyle()
				style.IconSrcRec = src

				if value == 1 {
					style.IconColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
				} else if value == 2 {
					style.IconColor = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
				} else {
					style.IconColor = getThemeColor(GUI_INSIDE)
				}

				style.ShadowOn = false // Buttons shouldn't have shadows here because they're on Tasks, which already handle their own shadows
				style.RightClick = true

				if imButton(dst, "", style) {

					if !c.Task.Board.Project.TaskOpen && !c.Task.Board.Project.ProjectSettingsOpen && c.Task.Board.Project.PopupAction == "" {

						if MousePressed(rl.MouseLeftButton) {

							if value == 1 {
								c.Task.TableData.Completions[y][x] = 0
							} else {
								c.Task.TableData.Completions[y][x] = 1
							}
							ConsumeMouseInput(rl.MouseLeftButton)

						} else if MousePressed(rl.MouseRightButton) {

							if value == 2 {
								c.Task.TableData.Completions[y][x] = 0
							} else {
								c.Task.TableData.Completions[y][x] = 2
							}
							ConsumeMouseInput(rl.MouseRightButton)

						}

						createUndo = true

					}

				}

				// rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, src, dst, rl.Vector2{}, 0, rl.White)

			}

		}

		// rl.DrawRectangleRec(rl.Rectangle{c.Task.Rect.X, c.Task.Rect.Y, 16, 16})

		src = rl.Rectangle{1, 1, c.Task.Rect.Width - gridWidth - 1, c.Task.Rect.Height - gridHeight - 1}
		dst = src
		dst.X = c.Task.Rect.X + 1
		dst.Y = c.Task.Rect.Y + 1
		dst.Width--
		dst.Height--
		rl.DrawTexturePro(c.StripesPattern, src, dst, rl.Vector2{}, 0, getThemeColor(GUI_INSIDE))

		shadowColor := rl.Black
		shadowColor.A = 128

		src = rl.Rectangle{216, 16, 8, 8}
		dst = rl.Rectangle{pos.X, pos.Y, gridWidth, 8}
		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, src, dst, rl.Vector2{}, 0, shadowColor)

		src = rl.Rectangle{224, 8, 8, 8}
		dst = rl.Rectangle{pos.X, pos.Y, 8, gridHeight}
		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, src, dst, rl.Vector2{}, 0, shadowColor)

		c.Task.Locked = lockTask

		worldGUI = false

		displaySize.X += 2
		displaySize.Y += 2

		displaySize = c.Task.Board.Project.RoundPositionToGrid(displaySize)

		if c.Task.DisplaySize != displaySize {
			c.Task.DisplaySize = displaySize
			c.Task.Board.TaskChanged = true // Have the board reorder if the size is different
		}

	}

	if createUndo {
		c.Task.UndoChange = true
	}

}

func (c *TableContents) Destroy() {}

func (c *TableContents) Trigger(triggerMode int) {

	for y := range c.Task.TableData.Completions {

		for x := range c.Task.TableData.Completions[y] {

			if triggerMode == TASK_TRIGGER_SET {

				c.Task.TableData.Completions[y][x] = 1

			} else if triggerMode == TASK_TRIGGER_CLEAR {

				c.Task.TableData.Completions[y][x] = 0

			} else if triggerMode == TASK_TRIGGER_TOGGLE {

				value := c.Task.TableData.Completions[y][x]
				if value == 0 {
					value = 1
				} else {
					value = 0
				}
				c.Task.TableData.Completions[y][x] = value

			}

		}

	}

	c.Task.UndoChange = true

}

func (c *TableContents) ReceiveMessage(msg string) {

	if msg == MessageDoubleClick && c.Task.TableData != nil {
		c.Task.TableData.SetPanel()
	}

}
