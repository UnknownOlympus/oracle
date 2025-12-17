# Multi-Language (i18n) Implementation for Oracle Bot

This document describes the internationalization (i18n) implementation added to the Oracle Telegram bot.

## Overview

The bot now supports multiple languages with the following features:
- **Supported Languages**: English (en) and Ukrainian (uk)
- **Auto-detection**: Automatically detects user language from Telegram settings
- **User preference storage**: Language preference is stored in the database
- **Easy switching**: Users can change language via `/language` command

## Architecture

### Components

1. **i18n Package** (`internal/i18n/`)
   - `i18n.go`: Core localizer with translation loading and retrieval
   - `locales/en.json`: English translations
   - `locales/uk.json`: Ukrainian translations

2. **Database**
   - `bot_users.locale`: Stores user language preference
   - Migration: `db-migrator/migrations/009_add_locale_column.sql`

3. **Repository Methods**
   - `SetUserLanguage(ctx, telegramID, langCode)`: Save language preference
   - `GetUserLanguage(ctx, telegramID)`: Retrieve language preference

4. **Bot Integration**
   - Helper methods: `t()` and `tWithData()` for getting translations
   - `getUserLanguage()`: Retrieves user language with auto-detection
   - Dynamic menu builders for i18n support

## Usage

### For Users

1. **Auto-detection**: When a user first starts the bot, their language is automatically detected from their Telegram settings

2. **Manual selection**: Users can change language anytime using:
   ```
   /language
   ```

3. **Supported languages**:
   - 🇬🇧 English
   - 🇺🇦 Українська

### For Developers

#### Adding Translations

1. Add the translation key to both language files:

**English** (`internal/i18n/locales/en.json`):
```json
{
  "my.new.key": "Hello, {name}!"
}
```

**Ukrainian** (`internal/i18n/locales/uk.json`):
```json
{
  "my.new.key": "Привіт, {name}!"
}
```

#### Using Translations in Handlers

**Simple translation**:
```go
func (b *Bot) myHandler(ctx telebot.Context) error {
    timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    message := b.t(timeoutCtx, ctx, "my.translation.key")
    return ctx.Send(message)
}
```

**Translation with placeholders**:
```go
func (b *Bot) myHandler(ctx telebot.Context) error {
    timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    message := b.tWithData(timeoutCtx, ctx, "my.new.key", map[string]interface{}{
        "name": "John",
    })
    return ctx.Send(message)
}
```

#### Creating Dynamic Menus

Always use the dynamic menu builders for i18n support:

```go
// Build main menu (unauthenticated users)
menu := b.buildMainMenu(ctx, tCtx)

// Build auth menu (authenticated users)
menu := b.buildAuthMenuWithTranslations(ctx, tCtx, isAdmin)

// Build statistics menu
menu := b.buildStatMenu(ctx, tCtx)

// Build near tasks menu
menu := b.buildNearMenu(ctx, tCtx)
```

## Translation Keys Structure

Translation keys follow a hierarchical structure:

```
category.subcategory.specific_key
```

### Categories:
- `welcome.*` - Welcome messages
- `error.*` - Error messages
- `login.*` - Authentication related
- `logout.*` - Logout related
- `menu.*` - Menu button labels
- `info.*` - User information
- `tasks.*` - Task related messages
- `comment.*` - Comment functionality
- `report.*` - Report generation
- `statistic.*` - Statistics
- `general.*` - General messages
- `admin.*` - Admin features
- `language.*` - Language selection

## Database Migration

The `locale` column has already been added to the `bot_users` table via migration `009_add_locale_column.sql`.

No manual database changes are required.

## Implementation Details

### Language Detection Flow

1. User sends `/start` or any command
2. Bot calls `getUserLanguage(ctx, tCtx)`
3. Check database for saved language preference
4. If not set, detect from `tCtx.Sender().LanguageCode`
5. Normalize language code (e.g., "en-US" → "en", "ua" → "uk")
6. Save detected language to database (asynchronously)
7. Return language code for translation

### Fallback Mechanism

1. If a translation key is not found in the requested language, it falls back to English
2. If still not found in English, returns the key itself
3. This ensures the bot never shows empty messages

## Handler Updates

The following handlers have been updated to support i18n:
- ✅ `startHandler` - Welcome messages
- ✅ `authHandler` - Login prompt
- ✅ `loginInputHandler` - Login responses
- ✅ `textHandler` - General text handling
- ✅ `languageHandler` - Language selection (NEW)
- ✅ `languageChangeHandler` - Language change confirmation (NEW)

### Remaining Handlers

The following handlers still need to be updated (but will use fallback to English):
- `logoutHandler`
- `infoHandler`
- `activeTasksHandler`
- `taskDetailsHandler`
- `reportHandler`
- `nearTasksHandler`
- `commentAcceptHandler`
- `statisticHandler*`
- Admin handlers

These can be updated following the same pattern as the updated handlers.

## Testing

### Manual Testing

1. Start the bot: `/start`
2. Check that it detects your Telegram language
3. Change language: `/language`
4. Select a different language
5. Verify all messages appear in the selected language
6. Test with English Telegram user
7. Test with Ukrainian Telegram user

### Adding New Language

To add a new language (e.g., German - `de`):

1. Create `internal/i18n/locales/de.json` with all translations
2. Update `NewLocalizer()` in `i18n.go`:
   ```go
   languages := []string{"en", "uk", "de"}
   ```
3. Update `NormalizeLanguageCode()` to handle the new language code
4. Add button to language selection menu in `language_handlers.go`
5. Add callback handler for the new language

## Benefits

- ✅ **Idiomatic Go**: Uses standard patterns and interfaces
- ✅ **Existing handlers work**: All existing functionality preserved
- ✅ **Auto-detection**: Seamless UX for new users
- ✅ **Database persistence**: Language preference survives bot restarts
- ✅ **Easy to extend**: Adding new languages or translations is straightforward
- ✅ **Type-safe**: All translations use string keys with fallback
- ✅ **Performance**: Translations loaded once at startup, minimal overhead

## Notes

- Translation files are embedded in the binary using `//go:embed`
- No external files needed at runtime
- Localizer is thread-safe (uses `sync.RWMutex`)
- Language detection happens asynchronously to avoid blocking user interactions
- Cache clearing may be needed when language changes for menu regeneration
