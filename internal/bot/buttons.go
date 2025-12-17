package bot

import (
	"context"

	"gopkg.in/telebot.v4"
)

// buildMainMenu creates the main menu for unauthenticated users with translated text.
func (b *Bot) buildMainMenu(ctx context.Context, tCtx telebot.Context) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	btnLogin := menu.Text(b.t(ctx, tCtx, "menu.login"))
	menu.Reply(menu.Row(btnLogin))
	return menu
}

// buildAuthMenuWithTranslations creates the authenticated user menu with translated text.
func (b *Bot) buildAuthMenuWithTranslations(
	ctx context.Context,
	tCtx telebot.Context,
	isAdmin bool,
) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	btnInfo := menu.Text(b.t(ctx, tCtx, "menu.about_me"))
	btnActiveTasks := menu.Text(b.t(ctx, tCtx, "menu.active_tasks"))
	btnNear := menu.Text(b.t(ctx, tCtx, "menu.tasks_near"))
	btnStatistic := menu.Text(b.t(ctx, tCtx, "menu.my_statistic"))
	btnReport := menu.Text(b.t(ctx, tCtx, "menu.create_report"))
	btnLanguage := menu.Text(b.t(ctx, tCtx, "menu.language"))
	btnLogout := menu.Text(b.t(ctx, tCtx, "menu.logout"))

	rows := []telebot.Row{
		menu.Row(btnInfo),
		menu.Row(btnActiveTasks),
		menu.Row(btnNear),
		menu.Row(btnStatistic),
		menu.Row(btnReport),
	}

	if isAdmin {
		btnAdmin := menu.Text(b.t(ctx, tCtx, "menu.admin_panel"))
		rows = append(rows, menu.Row(btnAdmin))
	}

	rows = append(rows, menu.Row(btnLanguage), menu.Row(btnLogout))
	menu.Reply(rows...)

	return menu
}

// buildStatMenu creates the statistics menu with translated text.
func (b *Bot) buildStatMenu(ctx context.Context, tCtx telebot.Context) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	btnToday := menu.Text(b.t(ctx, tCtx, "menu.today"))
	btnMonth := menu.Text(b.t(ctx, tCtx, "menu.this_month"))
	btnYear := menu.Text(b.t(ctx, tCtx, "menu.this_year"))
	btnBack := menu.Text(b.t(ctx, tCtx, "menu.back"))

	menu.Reply(
		menu.Row(btnToday),
		menu.Row(btnMonth),
		menu.Row(btnYear),
		menu.Row(btnBack),
	)

	return menu
}

// buildNearMenu creates the near tasks menu with translated text.
func (b *Bot) buildNearMenu(ctx context.Context, tCtx telebot.Context) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	btnLocation := menu.Location(b.t(ctx, tCtx, "menu.send_location"))
	btnBack := menu.Text(b.t(ctx, tCtx, "menu.back"))

	menu.Reply(
		menu.Row(btnLocation),
		menu.Row(btnBack),
	)

	return menu
}

// buildAdminMenu creates the admin menu with translated text.
func (b *Bot) buildAdminMenu(ctx context.Context, tCtx telebot.Context) *telebot.ReplyMarkup {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}

	btnBroadcast := menu.Text(b.t(ctx, tCtx, "menu.broadcast"))
	btnBack := menu.Text(b.t(ctx, tCtx, "menu.back"))

	menu.Reply(
		menu.Row(btnBroadcast),
		menu.Row(btnBack),
	)

	return menu
}
