package bot

import (
	"context"
	"fmt"
	"strings"

	"gopkg.in/telebot.v4"
)

// MenuBuilder handles dynamic menu generation with i18n support.
type MenuBuilder struct {
	bot      *Bot
	registry *MenuRegistry
	navStack *NavigationStack
}

// NewMenuBuilder creates a new menu builder instance.
func NewMenuBuilder(bot *Bot) *MenuBuilder {
	return &MenuBuilder{
		bot:      bot,
		registry: NewMenuRegistry(),
		navStack: NewNavigationStack(),
	}
}

// Build generates a telebot.ReplyMarkup from a menu definition.
func (mb *MenuBuilder) Build(
	ctx context.Context,
	tCtx telebot.Context,
	menuType MenuType,
	userID int64,
) *telebot.ReplyMarkup {
	menuDef := mb.registry.Get(menuType)
	if menuDef == nil {
		mb.bot.log.Error("Menu definition not found", "menuType", menuType)
		return mb.buildFallbackMenu(ctx, tCtx)
	}

	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	// Collect visible buttons based on permissions
	visibleButtons := mb.filterVisibleButtons(menuDef.Buttons, userID)

	// Build rows based on layout
	rows := mb.buildRows(ctx, tCtx, menu, visibleButtons, menuDef.Layout)

	// Add back button if needed
	if menuDef.HasBack {
		btnBack := menu.Text(mb.bot.t(ctx, tCtx, "menu.back"))
		rows = append(rows, menu.Row(btnBack))
	}

	menu.Reply(rows...)
	return menu
}

// filterVisibleButtons returns only buttons that user has permission to see.
func (mb *MenuBuilder) filterVisibleButtons(buttons []MenuButton, userID int64) []MenuButton {
	visible := make([]MenuButton, 0, len(buttons))

	for _, btn := range buttons {
		// Check role requirement
		if btn.RequiresRole != nil {
			hasRole := btn.RequiresRole(mb.bot, userID)
			mb.bot.log.Debug("Button role check", "button", btn.TextKey, "userID", userID, "hasRole", hasRole)
			if !hasRole {
				continue
			}
		}
		visible = append(visible, btn)
	}

	mb.bot.log.Debug("Filtered buttons", "total", len(buttons), "visible", len(visible), "userID", userID)
	return visible
}

// buildRows creates telebot.Row slices based on button layout.
func (mb *MenuBuilder) buildRows(
	ctx context.Context,
	tCtx telebot.Context,
	menu *telebot.ReplyMarkup,
	buttons []MenuButton,
	layout []int,
) []telebot.Row {
	rows := make([]telebot.Row, 0)
	buttonIdx := 0

	for _, rowSize := range layout {
		if buttonIdx >= len(buttons) {
			break
		}

		rowButtons := make([]telebot.Btn, 0, rowSize)
		for i := 0; i < rowSize && buttonIdx < len(buttons); i++ {
			btn := buttons[buttonIdx]
			buttonIdx++

			// Build button text with emoji
			buttonText := mb.buildButtonText(ctx, tCtx, btn)

			// Check if it's a location button
			if btn.TextKey == "menu.send_location" {
				rowButtons = append(rowButtons, menu.Location(buttonText))
			} else {
				rowButtons = append(rowButtons, menu.Text(buttonText))
			}
		}

		if len(rowButtons) > 0 {
			rows = append(rows, menu.Row(rowButtons...))
		}
	}

	// Handle remaining buttons if any
	for buttonIdx < len(buttons) {
		btn := buttons[buttonIdx]
		buttonText := mb.buildButtonText(ctx, tCtx, btn)
		rows = append(rows, menu.Row(menu.Text(buttonText)))
		buttonIdx++
	}

	return rows
}

// buildButtonText constructs button text with optional emoji.
func (mb *MenuBuilder) buildButtonText(ctx context.Context, tCtx telebot.Context, btn MenuButton) string {
	text := mb.bot.t(ctx, tCtx, btn.TextKey)
	if btn.Emoji != "" {
		return fmt.Sprintf("%s %s", btn.Emoji, text)
	}
	return text
}

// buildFallbackMenu creates a safe fallback menu in case of errors.
func (mb *MenuBuilder) buildFallbackMenu(ctx context.Context, tCtx telebot.Context) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	btnBack := menu.Text(mb.bot.t(ctx, tCtx, "menu.back"))
	menu.Reply(menu.Row(btnBack))
	return menu
}

// ShowMenu sends a menu to the user with optional message.
// If trackNavigation is false, the menu won't be added to navigation history (used for back navigation).
func (mb *MenuBuilder) ShowMenu(
	ctx context.Context,
	tCtx telebot.Context,
	menuType MenuType,
	userID int64,
	messageKey string,
	trackNavigation bool,
) error {
	menu := mb.Build(ctx, tCtx, menuType, userID)

	// Track navigation only if requested
	if trackNavigation {
		mb.navStack.Push(userID, menuType)
	}

	// Determine if we should send a message with the menu
	var message string

	if messageKey != "" {
		message = mb.bot.t(ctx, tCtx, messageKey)
	} else {
		// Check if menu has a title
		menuDef := mb.registry.Get(menuType)
		if menuDef != nil && menuDef.TitleKey != "" {
			message = mb.bot.t(ctx, tCtx, menuDef.TitleKey)
		} else {
			// User a default welcome message
			message = mb.bot.t(ctx, tCtx, "general.welcome_back")
		}
	}

	return tCtx.Send(message, menu)
}

// NavigateBack returns user to previous menu.
func (mb *MenuBuilder) NavigateBack(
	ctx context.Context,
	tCtx telebot.Context,
	userID int64,
) error {
	// Pop current menu
	mb.navStack.Pop(userID)

	// Get previous menu (or default to main)
	prevMenu := mb.navStack.Current(userID)
	if prevMenu == "" {
		prevMenu = MenuMain
	}

	// Show the previous menu without tracking (already in stack)
	return mb.ShowMenu(ctx, tCtx, prevMenu, userID, "", false)
}

// ResolveHandlerFromButtonText looks up which handler to call based on button text.
// This is used in routeTextHandler to map button clicks to handler functions.
func (mb *MenuBuilder) ResolveHandlerFromButtonText(
	ctx context.Context,
	tCtx telebot.Context,
	buttonText string,
) (string, MenuType) {
	lang := mb.bot.getUserLanguage(ctx, tCtx)

	// Try both current language and fallback
	languages := []string{lang}
	if lang != "en" {
		languages = append(languages, "en")
	} else {
		languages = append(languages, "uk")
	}

	// Search all menus for matching button
	for _, menuType := range []MenuType{MenuMain, MenuTasks, MenuProfile, MenuStats, MenuMore, MenuAdmin} {
		menuDef := mb.registry.Get(menuType)
		if menuDef == nil {
			continue
		}

		for _, btn := range menuDef.Buttons {
			for _, checkLang := range languages {
				// Get the text from i18n (which already includes emojis)
				expectedText := mb.bot.localizer.Get(checkLang, btn.TextKey)

				// Only add emoji prefix if it's set in the button definition AND not already in the i18n text
				if btn.Emoji != "" && !strings.HasPrefix(expectedText, btn.Emoji) {
					expectedText = fmt.Sprintf("%s %s", btn.Emoji, expectedText)
				}

				if buttonText == expectedText {
					return btn.Handler, btn.SubMenu
				}
			}
		}
	}

	return "", ""
}
