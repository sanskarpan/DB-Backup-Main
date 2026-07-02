// ENHANCED Desktop App - Rust Backend
// Implements 15+ new features for Phase 26.1 + Platform Integration for Phase 26.17

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::sync::Mutex;
use tauri::{
    AppHandle, CustomMenuItem, Manager, SystemTray, SystemTrayEvent, SystemTrayMenu,
    SystemTrayMenuItem, Window, GlobalShortcutManager,
};
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use auto_launch::AutoLaunch;

// Platform integration module
mod platform;

// Enhanced State management
struct AppState {
    api_url: Mutex<String>,
    api_key: Mutex<Option<String>>,
    notifications_enabled: Mutex<bool>,
    theme: Mutex<String>, // NEW: Theme support (light, dark, auto)
    language: Mutex<String>, // NEW: i18n support
    auto_launch: Mutex<bool>, // NEW: Auto-launch setting
    keyboard_shortcuts: Mutex<KeyboardShortcuts>, // NEW: Custom shortcuts
}

#[derive(Debug, Serialize, Deserialize, Clone)]
struct KeyboardShortcuts {
    quick_search: String,       // Default: "CommandOrControl+K"
    new_backup: String,          // Default: "CommandOrControl+N"
    refresh: String,             // Default: "CommandOrControl+R"
    toggle_theme: String,        // Default: "CommandOrControl+Shift+D"
    open_settings: String,       // Default: "CommandOrControl+,"
}

impl Default for KeyboardShortcuts {
    fn default() -> Self {
        Self {
            quick_search: "CommandOrControl+K".to_string(),
            new_backup: "CommandOrControl+N".to_string(),
            refresh: "CommandOrControl+R".to_string(),
            toggle_theme: "CommandOrControl+Shift+D".to_string(),
            open_settings: "CommandOrControl+,".to_string(),
        }
    }
}

#[derive(Debug, Serialize, Deserialize)]
struct BackupJob {
    id: String,
    database_id: String,
    database_name: String,
    status: String,
    created_at: DateTime<Utc>,
    size: Option<i64>,
    duration: Option<i64>,
    tables: Option<Vec<String>>, // NEW: Table list for preview
    schema: Option<String>, // NEW: Schema preview
}

#[derive(Debug, Serialize, Deserialize)]
struct BackupConfig {
    api_url: String,
    api_key: String,
    auto_backup_enabled: bool,
    backup_interval_hours: i32,
    notifications_enabled: bool,
    theme: String, // NEW
    language: String, // NEW
    auto_launch: bool, // NEW
    keyboard_shortcuts: KeyboardShortcuts, // NEW
}

#[derive(Debug, Serialize, Deserialize)]
struct NotificationPayload {
    title: String,
    body: String,
    icon: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
struct SearchResult {
    id: String,
    title: String,
    description: String,
    category: String, // "backup", "database", "setting"
    score: f64,
}

// ============================================================================
// PERSISTED CONFIG (survives app restarts)
// ============================================================================

/// Config that is written to disk so settings (including the API key) survive
/// application restarts. Stored as JSON in the platform config directory.
#[derive(Debug, Serialize, Deserialize)]
struct PersistedConfig {
    api_url: String,
    api_key: Option<String>,
    notifications_enabled: bool,
    theme: String,
    language: String,
    auto_launch: bool,
    keyboard_shortcuts: KeyboardShortcuts,
}

impl Default for PersistedConfig {
    fn default() -> Self {
        Self {
            api_url: "http://localhost:8080".to_string(),
            api_key: None,
            notifications_enabled: true,
            theme: "light".to_string(),
            language: "en".to_string(),
            auto_launch: false,
            keyboard_shortcuts: KeyboardShortcuts::default(),
        }
    }
}

/// Location of the persisted config file (e.g. ~/.config/dbbackup/desktop/config.json).
fn config_file_path() -> Option<std::path::PathBuf> {
    directories::ProjectDirs::from("com", "dbbackup", "desktop")
        .map(|dirs| dirs.config_dir().join("config.json"))
}

/// Load persisted config from disk, falling back to defaults when the file is
/// missing or unreadable.
fn load_persisted_config() -> PersistedConfig {
    if let Some(path) = config_file_path() {
        if let Ok(contents) = std::fs::read_to_string(&path) {
            if let Ok(cfg) = serde_json::from_str::<PersistedConfig>(&contents) {
                return cfg;
            }
        }
    }
    PersistedConfig::default()
}

/// Persist config to disk, creating the config directory if needed.
fn save_persisted_config(cfg: &PersistedConfig) -> Result<(), String> {
    let path = config_file_path().ok_or("Could not determine config directory")?;
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent).map_err(|e| e.to_string())?;
    }
    let json = serde_json::to_string_pretty(cfg).map_err(|e| e.to_string())?;
    std::fs::write(&path, json).map_err(|e| e.to_string())?;
    Ok(())
}

// ============================================================================
// TAURI COMMANDS - ORIGINAL
// ============================================================================

#[tauri::command]
async fn get_backups(
    state: tauri::State<'_, AppState>,
    limit: Option<i32>,
) -> Result<Vec<BackupJob>, String> {
    let api_url = state.api_url.lock().unwrap().clone();
    let api_key = state.api_key.lock().unwrap().clone();

    if api_key.is_none() {
        return Err("API key not configured".to_string());
    }

    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/backups", api_url);

    let response = client
        .get(&url)
        .header("Authorization", format!("Bearer {}", api_key.unwrap()))
        .query(&[("limit", limit.unwrap_or(50).to_string())])
        .send()
        .await
        .map_err(|e| e.to_string())?;

    if !response.status().is_success() {
        return Err(format!("API error: {}", response.status()));
    }

    let backups: Vec<BackupJob> = response.json().await.map_err(|e| e.to_string())?;
    Ok(backups)
}

#[tauri::command]
async fn create_backup(
    state: tauri::State<'_, AppState>,
    database_id: String,
    options: serde_json::Value,
) -> Result<BackupJob, String> {
    let api_url = state.api_url.lock().unwrap().clone();
    let api_key = state.api_key.lock().unwrap().clone();

    if api_key.is_none() {
        return Err("API key not configured".to_string());
    }

    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/backups", api_url);

    let mut payload = serde_json::json!({
        "database_id": database_id,
    });

    if let Some(obj) = payload.as_object_mut() {
        if let Some(opts) = options.as_object() {
            for (key, value) in opts {
                obj.insert(key.clone(), value.clone());
            }
        }
    }

    let response = client
        .post(&url)
        .header("Authorization", format!("Bearer {}", api_key.unwrap()))
        .json(&payload)
        .send()
        .await
        .map_err(|e| e.to_string())?;

    if !response.status().is_success() {
        return Err(format!("API error: {}", response.status()));
    }

    let backup: BackupJob = response.json().await.map_err(|e| e.to_string())?;
    Ok(backup)
}

// ============================================================================
// NEW COMMANDS - ENHANCEMENTS
// ============================================================================

/// NEW: Search across backups, databases, settings
#[tauri::command]
async fn quick_search(
    state: tauri::State<'_, AppState>,
    query: String,
) -> Result<Vec<SearchResult>, String> {
    let api_url = state.api_url.lock().unwrap().clone();
    let api_key = state.api_key.lock().unwrap().clone();

    if api_key.is_none() {
        return Err("API key not configured".to_string());
    }

    let client = reqwest::Client::new();
    // Backend exposes catalog search at /api/v1/catalog/search (see backend server.go)
    let url = format!("{}/api/v1/catalog/search", api_url);

    let response = client
        .get(&url)
        .header("Authorization", format!("Bearer {}", api_key.unwrap()))
        .query(&[("q", query)])
        .send()
        .await
        .map_err(|e| e.to_string())?;

    if !response.status().is_success() {
        return Err(format!("API error: {}", response.status()));
    }

    let results: Vec<SearchResult> = response.json().await.map_err(|e| e.to_string())?;
    Ok(results)
}

/// NEW: Toggle theme (light/dark)
#[tauri::command]
async fn toggle_theme(state: tauri::State<'_, AppState>) -> Result<String, String> {
    let mut theme = state.theme.lock().unwrap();
    *theme = match theme.as_str() {
        "light" => "dark".to_string(),
        "dark" => "light".to_string(),
        _ => "light".to_string(),
    };
    Ok(theme.clone())
}

/// NEW: Get current theme
#[tauri::command]
async fn get_theme(state: tauri::State<'_, AppState>) -> Result<String, String> {
    let theme = state.theme.lock().unwrap().clone();
    Ok(theme)
}

/// NEW: Set language
#[tauri::command]
async fn set_language(
    state: tauri::State<'_, AppState>,
    language: String,
) -> Result<(), String> {
    *state.language.lock().unwrap() = language;
    Ok(())
}

/// NEW: Get current language
#[tauri::command]
async fn get_language(state: tauri::State<'_, AppState>) -> Result<String, String> {
    let language = state.language.lock().unwrap().clone();
    Ok(language)
}

/// NEW: Enable/disable auto-launch
#[tauri::command]
async fn set_auto_launch(
    state: tauri::State<'_, AppState>,
    enabled: bool,
) -> Result<(), String> {
    *state.auto_launch.lock().unwrap() = enabled;

    let app_name = "DB Backup Desktop";
    let app_path = std::env::current_exe().map_err(|e| e.to_string())?;

    let auto = AutoLaunch::new(
        app_name,
        app_path.to_str().unwrap(),
        &[] as &[&str],
    );

    if enabled {
        auto.enable().map_err(|e| e.to_string())?;
    } else {
        auto.disable().map_err(|e| e.to_string())?;
    }

    Ok(())
}

/// NEW: Get auto-launch status
#[tauri::command]
async fn get_auto_launch(state: tauri::State<'_, AppState>) -> Result<bool, String> {
    let enabled = *state.auto_launch.lock().unwrap();
    Ok(enabled)
}

/// NEW: Update keyboard shortcuts
#[tauri::command]
async fn update_shortcuts(
    state: tauri::State<'_, AppState>,
    shortcuts: KeyboardShortcuts,
) -> Result<(), String> {
    *state.keyboard_shortcuts.lock().unwrap() = shortcuts;
    Ok(())
}

/// NEW: Get keyboard shortcuts
#[tauri::command]
async fn get_shortcuts(state: tauri::State<'_, AppState>) -> Result<KeyboardShortcuts, String> {
    let shortcuts = state.keyboard_shortcuts.lock().unwrap().clone();
    Ok(shortcuts)
}

// Original commands
#[tauri::command]
async fn get_config(state: tauri::State<'_, AppState>) -> Result<BackupConfig, String> {
    let api_url = state.api_url.lock().unwrap().clone();
    let api_key = state.api_key.lock().unwrap().clone();
    let notifications_enabled = *state.notifications_enabled.lock().unwrap();
    let theme = state.theme.lock().unwrap().clone();
    let language = state.language.lock().unwrap().clone();
    let auto_launch = *state.auto_launch.lock().unwrap();
    let keyboard_shortcuts = state.keyboard_shortcuts.lock().unwrap().clone();

    Ok(BackupConfig {
        api_url,
        api_key: api_key.unwrap_or_default(),
        auto_backup_enabled: false,
        backup_interval_hours: 24,
        notifications_enabled,
        theme,
        language,
        auto_launch,
        keyboard_shortcuts,
    })
}

#[tauri::command]
async fn update_config(
    state: tauri::State<'_, AppState>,
    config: BackupConfig,
) -> Result<(), String> {
    *state.api_url.lock().unwrap() = config.api_url.clone();
    *state.api_key.lock().unwrap() = Some(config.api_key.clone());
    *state.notifications_enabled.lock().unwrap() = config.notifications_enabled;
    *state.theme.lock().unwrap() = config.theme.clone();
    *state.language.lock().unwrap() = config.language.clone();
    *state.auto_launch.lock().unwrap() = config.auto_launch;
    *state.keyboard_shortcuts.lock().unwrap() = config.keyboard_shortcuts.clone();

    // Persist to disk so settings (including the API key) survive restarts.
    let persisted = PersistedConfig {
        api_url: config.api_url,
        api_key: if config.api_key.is_empty() {
            None
        } else {
            Some(config.api_key)
        },
        notifications_enabled: config.notifications_enabled,
        theme: config.theme,
        language: config.language,
        auto_launch: config.auto_launch,
        keyboard_shortcuts: config.keyboard_shortcuts,
    };
    save_persisted_config(&persisted)?;

    Ok(())
}

#[tauri::command]
async fn send_notification(
    app: AppHandle,
    payload: NotificationPayload,
) -> Result<(), String> {
    let identifier = app.config().tauri.bundle.identifier.clone();
    tauri::api::notification::Notification::new(identifier)
        .title(&payload.title)
        .body(&payload.body)
        .show()
        .map_err(|e| e.to_string())?;

    Ok(())
}

#[tauri::command]
async fn open_logs_directory() -> Result<String, String> {
    let dirs = directories::ProjectDirs::from("com", "dbbackup", "desktop")
        .ok_or("Could not determine logs directory")?;

    let log_dir = dirs.data_local_dir();
    Ok(log_dir.to_string_lossy().to_string())
}

#[tauri::command]
async fn check_for_updates(app: AppHandle) -> Result<(), String> {
    app.updater()
        .check()
        .await
        .map_err(|e| e.to_string())?;

    Ok(())
}

#[tauri::command]
fn show_window(window: Window) {
    window.show().unwrap();
    window.set_focus().unwrap();
}

#[tauri::command]
fn hide_window(window: Window) {
    window.hide().unwrap();
}

// System tray menu
fn create_tray_menu() -> SystemTray {
    let quit = CustomMenuItem::new("quit".to_string(), "Quit");
    let show = CustomMenuItem::new("show".to_string(), "Show Window");
    let hide = CustomMenuItem::new("hide".to_string(), "Hide Window");
    let backups = CustomMenuItem::new("backups".to_string(), "View Backups");
    let new_backup = CustomMenuItem::new("new_backup".to_string(), "New Backup");
    let toggle_theme = CustomMenuItem::new("toggle_theme".to_string(), "Toggle Theme");

    let tray_menu = SystemTrayMenu::new()
        .add_item(show)
        .add_item(hide)
        .add_native_item(SystemTrayMenuItem::Separator)
        .add_item(backups)
        .add_item(new_backup)
        .add_native_item(SystemTrayMenuItem::Separator)
        .add_item(toggle_theme)
        .add_native_item(SystemTrayMenuItem::Separator)
        .add_item(quit);

    SystemTray::new().with_menu(tray_menu)
}

// System tray event handler
fn handle_tray_event(app: &AppHandle, event: SystemTrayEvent) {
    match event {
        SystemTrayEvent::LeftClick {
            position: _,
            size: _,
            ..
        } => {
            let window = app.get_window("main").unwrap();
            window.show().unwrap();
            window.set_focus().unwrap();
        }
        SystemTrayEvent::MenuItemClick { id, .. } => match id.as_str() {
            "quit" => {
                std::process::exit(0);
            }
            "show" => {
                let window = app.get_window("main").unwrap();
                window.show().unwrap();
                window.set_focus().unwrap();
            }
            "hide" => {
                let window = app.get_window("main").unwrap();
                window.hide().unwrap();
            }
            "backups" => {
                let window = app.get_window("main").unwrap();
                window.show().unwrap();
                window.set_focus().unwrap();
                window.eval("window.location.hash = '/backups'").unwrap();
            }
            "new_backup" => {
                let window = app.get_window("main").unwrap();
                window.show().unwrap();
                window.set_focus().unwrap();
                window.eval("window.location.hash = '/backups/new'").unwrap();
            }
            "toggle_theme" => {
                // Mutate theme state in Rust and emit event to frontend.
                // Do NOT use window.eval with window.__TAURI__ — withGlobalTauri
                // is false in tauri.conf.json so that object is not injected.
                let state: tauri::State<AppState> = app.state();
                let new_theme = {
                    let mut theme = state.theme.lock().unwrap();
                    let next = if *theme == "dark" { "light" } else { "dark" };
                    *theme = next.to_string();
                    next.to_string()
                };
                let _ = app.emit_all("theme-changed", new_theme);
            }
            _ => {}
        },
        _ => {}
    }
}

fn main() {
    env_logger::init();

    // Load persisted settings (including the API key) so they survive restarts.
    let persisted = load_persisted_config();

    let app_state = AppState {
        api_url: Mutex::new(persisted.api_url),
        api_key: Mutex::new(persisted.api_key),
        notifications_enabled: Mutex::new(persisted.notifications_enabled),
        theme: Mutex::new(persisted.theme),
        language: Mutex::new(persisted.language),
        auto_launch: Mutex::new(persisted.auto_launch),
        keyboard_shortcuts: Mutex::new(persisted.keyboard_shortcuts),
    };

    tauri::Builder::default()
        .manage(app_state)
        .system_tray(create_tray_menu())
        .on_system_tray_event(handle_tray_event)
        .invoke_handler(tauri::generate_handler![
            // Original
            get_backups,
            create_backup,
            get_config,
            update_config,
            send_notification,
            open_logs_directory,
            check_for_updates,
            show_window,
            hide_window,
            // Enhanced features
            quick_search,
            toggle_theme,
            get_theme,
            set_language,
            get_language,
            set_auto_launch,
            get_auto_launch,
            update_shortcuts,
            get_shortcuts,
            // Cross-platform features
            platform::register_file_association,
            platform::unregister_file_association,
            platform::register_context_menu_items,
            platform::update_context_menu_item,
            platform::remove_context_menu_item,
            platform::register_protocol_handler,
            platform::unregister_protocol_handler,
            platform::show_custom_dialog,
            platform::show_progress_dialog,
            platform::register_shell_command,
            platform::unregister_shell_command,
            // Windows platform
            platform::windows_send_toast_notification,
            platform::windows_close_notification,
            platform::windows_update_live_tile,
            platform::windows_clear_live_tile,
            platform::windows_set_tile_badge,
            platform::windows_update_jump_list,
            platform::windows_clear_jump_list,
            platform::windows_set_thumbnail_buttons,
            platform::windows_update_thumbnail_button,
            platform::windows_get_version,
            platform::windows_show_task_dialog,
            // macOS platform
            platform::macos_update_control_center_widget,
            platform::macos_clear_control_center_widget,
            platform::macos_set_touch_bar_layout,
            platform::macos_update_touch_bar_button,
            platform::macos_clear_touch_bar,
            platform::macos_has_touch_bar,
            platform::macos_update_menu_bar,
            platform::macos_set_menu_bar_icon,
            platform::macos_set_menu_bar_title,
            platform::macos_update_menu_bar_item,
            platform::macos_register_quick_look_plugin,
            platform::macos_generate_quick_look_preview,
            platform::macos_generate_quick_look_thumbnail,
            platform::macos_set_finder_badge,
            platform::macos_add_finder_context_menu_items,
            platform::macos_register_finder_sync_extension,
            platform::macos_add_notification_center_widget,
            platform::macos_update_notification_center_widget,
            platform::macos_remove_notification_center_widget,
            platform::macos_send_notification,
            platform::macos_close_notification,
            platform::macos_get_version,
            platform::macos_show_sheet,
            // Linux platform
            platform::linux_install_gnome_extension,
            platform::linux_enable_gnome_extension,
            platform::linux_disable_gnome_extension,
            platform::linux_update_gnome_panel_button,
            platform::linux_gnome_show_notification,
            platform::linux_is_gnome_running,
            platform::linux_install_plasma_widget,
            platform::linux_update_plasma_widget,
            platform::linux_is_kde_running,
            platform::linux_create_system_tray,
            platform::linux_update_tray_icon,
            platform::linux_update_tray_menu,
            platform::linux_tray_show_notification,
            platform::linux_remove_system_tray,
            platform::linux_send_notification,
            platform::linux_close_notification,
            platform::linux_get_distribution_info,
        ])
        .setup(|app| {
            // Register global shortcuts
            let mut shortcut_manager = app.global_shortcut_manager();

            // Quick search (Cmd/Ctrl + K)
            shortcut_manager
                .register("CommandOrControl+K", move || {
                    // Show quick search dialog
                    println!("Quick search triggered");
                })
                .unwrap();

            // New backup (Cmd/Ctrl + N)
            shortcut_manager
                .register("CommandOrControl+N", move || {
                    println!("New backup triggered");
                })
                .unwrap();

            // Startup notification
            let identifier = app.config().tauri.bundle.identifier.clone();
            tauri::async_runtime::spawn(async move {
                tokio::time::sleep(tokio::time::Duration::from_secs(2)).await;
                let _ = tauri::api::notification::Notification::new(identifier)
                    .title("DB Backup Desktop")
                    .body("Application started successfully! Press Cmd/Ctrl+K for quick search")
                    .show();
            });

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
