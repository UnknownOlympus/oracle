package bot

import (
	"context"

	"gopkg.in/telebot.v4"
)

// buildMainMenu creates the main menu for unauthenticated users with translated text.
// DEPRECATED: This function is kept for backward compatibility during migration.
// New code should use menuBuilder.Build(MenuMain) instead.
func (b *Bot) buildMainMenu(ctx context.Context, tCtx telebot.Context) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	btnLogin := menu.Text(b.t(ctx, tCtx, "menu.login"))
	menu.Reply(menu.Row(btnLogin))
	return menu
}

// buildAuthMenuWithTranslations creates the authenticated user menu with translated text.
// DEPRECATED: Use menuBuilder.Build(MenuMain) instead.
func (b *Bot) buildAuthMenuWithTranslations(
	ctx context.Context,
	tCtx telebot.Context,
	isAdmin bool,
) *telebot.ReplyMarkup {
	// Use new menu builder system
	return b.menuBuilder.Build(ctx, tCtx, MenuMain, tCtx.Sender().ID)
}

// buildStatMenu creates the statistics menu with translated text.
// DEPRECATED: Use menuBuilder.Build(MenuStats) instead.
func (b *Bot) buildStatMenu(ctx context.Context, tCtx telebot.Context) *telebot.ReplyMarkup {
	return b.menuBuilder.Build(ctx, tCtx, MenuStats, tCtx.Sender().ID)
}

// buildNearMenu creates the near tasks menu with translated text.
// DEPRECATED: Use menuBuilder.Build(MenuNearTasks) instead.
func (b *Bot) buildNearMenu(ctx context.Context, tCtx telebot.Context) *telebot.ReplyMarkup {
	return b.menuBuilder.Build(ctx, tCtx, MenuNearTasks, tCtx.Sender().ID)
}

// buildAdminMenu creates the admin menu with translated text.
// DEPRECATED: Use menuBuilder.Build(MenuAdmin) instead.
func (b *Bot) buildAdminMenu(ctx context.Context, tCtx telebot.Context) *telebot.ReplyMarkup {
	return b.menuBuilder.Build(ctx, tCtx, MenuAdmin, tCtx.Sender().ID)
}
