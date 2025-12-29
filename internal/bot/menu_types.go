package bot

import (
	"context"
	"time"
)

// MenuType represents different menu screens in the bot.
type MenuType string

const (
	MenuMain      MenuType = "main"
	MenuTasks     MenuType = "tasks"
	MenuProfile   MenuType = "profile"
	MenuStats     MenuType = "stats"
	MenuMore      MenuType = "more"
	MenuAdmin     MenuType = "admin"
	MenuNearTasks MenuType = "near_tasks"
)

// MenuButton represents a single button in a menu.
type MenuButton struct {
	TextKey      string                 // i18n key for button text
	Handler      string                 // Handler function name or unique identifier
	Emoji        string                 // Optional emoji prefix
	SubMenu      MenuType               // If this button opens a submenu
	RequiresAuth bool                   // Whether user must be authenticated
	RequiresRole func(*Bot, int64) bool // Optional role check (e.g., isAdmin)
	InlineData   string                 // For inline buttons
}

// MenuDefinition represents a complete menu screen.
type MenuDefinition struct {
	Type     MenuType
	TitleKey string // i18n key for menu title (optional, sent as message)
	Buttons  []MenuButton
	Layout   []int // Button layout: [2, 2, 1] means 2+2+1 buttons per row
	HasBack  bool  // Whether to show back button
}

// MenuRegistry holds all menu definitions.
type MenuRegistry struct {
	menus map[MenuType]*MenuDefinition
}

// NewMenuRegistry creates and initializes the menu registry with all menu definitions.
func NewMenuRegistry() *MenuRegistry {
	registry := &MenuRegistry{
		menus: make(map[MenuType]*MenuDefinition),
	}

	// Define all menus
	registry.registerMainMenu()
	registry.registerTasksMenu()
	registry.registerProfileMenu()
	registry.registerStatsMenu()
	registry.registerMoreMenu()
	registry.registerAdminMenu()
	registry.registerNearTasksMenu()

	return registry
}

func (r *MenuRegistry) registerMainMenu() {
	r.menus[MenuMain] = &MenuDefinition{
		Type:    MenuMain,
		Layout:  []int{1, 1}, // 1 button per row
		HasBack: false,
		Buttons: []MenuButton{
			{
				TextKey:      "menu.tasks",
				SubMenu:      MenuTasks,
				RequiresAuth: true,
			},
			{
				TextKey:      "menu.profile",
				SubMenu:      MenuProfile,
				RequiresAuth: true,
			},
			{
				TextKey:      "menu.more",
				SubMenu:      MenuMore,
				RequiresAuth: true,
			},
			{
				TextKey:      "menu.logout",
				Handler:      "logout",
				RequiresAuth: true,
			},
		},
	}
}

func (r *MenuRegistry) registerTasksMenu() {
	r.menus[MenuTasks] = &MenuDefinition{
		Type:     MenuTasks,
		TitleKey: "tasks.title",
		Layout:   []int{1, 1}, // 1 button per row
		HasBack:  true,
		Buttons: []MenuButton{
			{
				TextKey: "menu.active_tasks",
				Handler: "active_tasks",
			},
			{
				TextKey: "menu.tasks_near",
				Handler: "near_tasks",
			},
		},
	}
}

func (r *MenuRegistry) registerProfileMenu() {
	r.menus[MenuProfile] = &MenuDefinition{
		Type:     MenuProfile,
		TitleKey: "profile.title",
		Layout:   []int{1, 1, 1}, // 1 button per row
		HasBack:  true,
		Buttons: []MenuButton{
			{
				TextKey: "menu.about_me",
				Handler: "info",
			},
			{
				TextKey: "menu.my_statistic",
				SubMenu: MenuStats,
			},
			{
				TextKey: "menu.create_report",
				Handler: "report",
			},
		},
	}
}

func (r *MenuRegistry) registerStatsMenu() {
	r.menus[MenuStats] = &MenuDefinition{
		Type:     MenuStats,
		TitleKey: "statistic.title",
		Layout:   []int{1, 1, 1}, // 1 button per row
		HasBack:  true,
		Buttons: []MenuButton{
			{
				TextKey: "menu.today",
				Handler: "statistic_today",
			},
			{
				TextKey: "menu.this_month",
				Handler: "statistic_month",
			},
			{
				TextKey: "menu.this_year",
				Handler: "statistic_year",
			},
		},
	}
}

func (r *MenuRegistry) registerMoreMenu() {
	r.menus[MenuMore] = &MenuDefinition{
		Type:     MenuMore,
		TitleKey: "more.title",
		Layout:   []int{1, 1, 1}, // 1 button per row
		HasBack:  true,
		Buttons: []MenuButton{
			{
				TextKey: "menu.language",
				Handler: "language",
			},
			{
				TextKey: "menu.report_issue",
				Handler: "report_issue",
			},
			{
				TextKey:      "menu.admin_panel",
				SubMenu:      MenuAdmin,
				RequiresRole: (*Bot).IsAdminCheck,
			},
		},
	}
}

func (r *MenuRegistry) registerAdminMenu() {
	r.menus[MenuAdmin] = &MenuDefinition{
		Type:     MenuAdmin,
		TitleKey: "admin.panel.title",
		Layout:   []int{1, 1, 1}, // 1 button per row
		HasBack:  true,
		Buttons: []MenuButton{
			{
				TextKey: "menu.broadcast",
				Handler: "broadcast_initiate",
			},
			{
				TextKey: "menu.geocoding_issues",
				Handler: "geocoding_issues",
			},
			{
				TextKey: "menu.geocoding_reset",
				Handler: "geocoding_reset",
			},
		},
	}
}

func (r *MenuRegistry) registerNearTasksMenu() {
	r.menus[MenuNearTasks] = &MenuDefinition{
		Type:    MenuNearTasks,
		Layout:  []int{1}, // Location button takes full width
		HasBack: true,
		Buttons: []MenuButton{
			{
				TextKey: "menu.send_location",
				Handler: "near_tasks_location",
			},
		},
	}
}

// Get retrieves a menu definition by type.
func (r *MenuRegistry) Get(menuType MenuType) *MenuDefinition {
	return r.menus[menuType]
}

// IsAdminCheck is a helper method to check if user is admin.
func (b *Bot) IsAdminCheck(userID int64) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	isAdmin, err := b.usrepo.IsAdmin(ctx, userID)
	if err != nil {
		b.log.Error("Failed to check admin status", "error", err, "userID", userID)
		return false
	}

	b.log.Debug("Admin check result", "userID", userID, "isAdmin", isAdmin)
	return isAdmin
}
