// Linux Platform Integration
// GNOME Shell Extension, KDE Plasma Widget, System Tray, D-Bus Notifications

use serde::{Deserialize, Serialize};

// ============================================================================
// Types
// ============================================================================

#[derive(Debug, Serialize, Deserialize)]
pub struct GnomePanelButton {
    pub id: String,
    pub label: String,
    pub icon: Option<String>,
    pub menu: Option<Vec<GnomeMenuItem>>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct GnomeMenuItem {
    pub id: String,
    #[serde(rename = "type")]
    pub item_type: String,
    pub label: Option<String>,
    pub icon: Option<String>,
    pub sensitive: Option<bool>,
    pub submenu: Option<Vec<GnomeMenuItem>>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct PlasmaWidgetContent {
    pub title: String,
    pub subtitle: Option<String>,
    pub items: Option<Vec<PlasmaWidgetItem>>,
    pub actions: Option<Vec<PlasmaWidgetAction>>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct PlasmaWidgetItem {
    pub id: String,
    pub text: String,
    pub icon: Option<String>,
    pub sub_text: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct PlasmaWidgetAction {
    pub id: String,
    pub text: String,
    pub icon: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SystemTrayIcon {
    pub icon: String,
    pub tooltip: String,
    pub menu: Vec<SystemTrayMenuItem>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SystemTrayMenuItem {
    pub id: String,
    #[serde(rename = "type")]
    pub item_type: String,
    pub label: Option<String>,
    pub icon: Option<String>,
    pub enabled: Option<bool>,
    pub checked: Option<bool>,
    pub submenu: Option<Vec<SystemTrayMenuItem>>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct LinuxNotification {
    pub summary: String,
    pub body: String,
    pub icon: Option<String>,
    pub urgency: Option<String>,
    pub timeout: Option<i32>,
    pub actions: Option<Vec<NotificationAction>>,
    pub hints: Option<serde_json::Value>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct NotificationAction {
    pub id: String,
    pub label: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct DistributionInfo {
    pub name: String,
    pub version: String,
    pub id: String,
}

// ============================================================================
// GNOME Shell Extension
// ============================================================================

#[tauri::command]
pub async fn linux_install_gnome_extension(uuid: String) -> Result<(), String> {
    #[cfg(target_os = "linux")]
    {
        log::info!("Installing GNOME Shell extension: {}", uuid);

        // GNOME Shell extensions are typically installed to:
        // ~/.local/share/gnome-shell/extensions/{uuid}/
        // This would require creating the extension files

        Ok(())
    }

    #[cfg(not(target_os = "linux"))]
    {
        Err("GNOME Shell extensions only available on Linux".to_string())
    }
}

#[tauri::command]
pub async fn linux_enable_gnome_extension(uuid: String) -> Result<(), String> {
    #[cfg(target_os = "linux")]
    {
        use std::process::Command;

        log::info!("Enabling GNOME Shell extension: {}", uuid);

        // Use gnome-extensions command if available
        let _ = Command::new("gnome-extensions")
            .args(&["enable", &uuid])
            .output();

        Ok(())
    }

    #[cfg(not(target_os = "linux"))]
    {
        Err("GNOME Shell extensions only available on Linux".to_string())
    }
}

#[tauri::command]
pub async fn linux_disable_gnome_extension(uuid: String) -> Result<(), String> {
    #[cfg(target_os = "linux")]
    {
        use std::process::Command;

        log::info!("Disabling GNOME Shell extension: {}", uuid);

        let _ = Command::new("gnome-extensions")
            .args(&["disable", &uuid])
            .output();

        Ok(())
    }

    #[cfg(not(target_os = "linux"))]
    {
        Err("GNOME Shell extensions only available on Linux".to_string())
    }
}

#[tauri::command]
pub async fn linux_update_gnome_panel_button(button: GnomePanelButton) -> Result<(), String> {
    #[cfg(target_os = "linux")]
    {
        log::info!("Updating GNOME panel button: {} - {}", button.id, button.label);

        // This would communicate with the GNOME Shell extension via D-Bus
        Ok(())
    }

    #[cfg(not(target_os = "linux"))]
    {
        Err("GNOME panel buttons only available on Linux".to_string())
    }
}

#[tauri::command]
pub async fn linux_gnome_show_notification(
    title: String,
    message: String,
) -> Result<(), String> {
    #[cfg(target_os = "linux")]
    {
        log::info!("Showing GNOME notification: {} - {}", title, message);

        // Use D-Bus notification
        linux_send_notification(LinuxNotification {
            summary: title,
            body: message,
            icon: None,
            urgency: Some("normal".to_string()),
            timeout: Some(-1),
            actions: None,
            hints: None,
        })
        .await?;

        Ok(())
    }

    #[cfg(not(target_os = "linux"))]
    {
        Err("GNOME notifications only available on Linux".to_string())
    }
}

#[tauri::command]
pub async fn linux_is_gnome_running() -> Result<bool, String> {
    #[cfg(target_os = "linux")]
    {
        use std::env;

        // Check if running GNOME
        let desktop_session = env::var("DESKTOP_SESSION").unwrap_or_default();
        let xdg_current_desktop = env::var("XDG_CURRENT_DESKTOP").unwrap_or_default();

        let is_gnome = desktop_session.to_lowercase().contains("gnome")
            || xdg_current_desktop.to_lowercase().contains("gnome");

        Ok(is_gnome)
    }

    #[cfg(not(target_os = "linux"))]
    {
        Ok(false)
    }
}

// ============================================================================
// KDE Plasma Widget
// ============================================================================

#[tauri::command]
pub async fn linux_install_plasma_widget(widget_id: String) -> Result<(), String> {
    #[cfg(target_os = "linux")]
    {
        log::info!("Installing KDE Plasma widget: {}", widget_id);

        // Plasma widgets are installed via plasmapkg2 command
        // This would require creating the widget package
        Ok(())
    }

    #[cfg(not(target_os = "linux"))]
    {
        Err("KDE Plasma widgets only available on Linux".to_string())
    }
}

#[tauri::command]
pub async fn linux_update_plasma_widget(
    widget_id: String,
    content: PlasmaWidgetContent,
) -> Result<(), String> {
    #[cfg(target_os = "linux")]
    {
        log::info!(
            "Updating KDE Plasma widget: {} - {}",
            widget_id,
            content.title
        );

        // This would communicate with the Plasma widget via D-Bus
        Ok(())
    }

    #[cfg(not(target_os = "linux"))]
    {
        Err("KDE Plasma widgets only available on Linux".to_string())
    }
}

#[tauri::command]
pub async fn linux_is_kde_running() -> Result<bool, String> {
    #[cfg(target_os = "linux")]
    {
        use std::env;

        // Check if running KDE Plasma
        let desktop_session = env::var("DESKTOP_SESSION").unwrap_or_default();
        let xdg_current_desktop = env::var("XDG_CURRENT_DESKTOP").unwrap_or_default();

        let is_kde = desktop_session.to_lowercase().contains("plasma")
            || desktop_session.to_lowercase().contains("kde")
            || xdg_current_desktop.to_lowercase().contains("kde");

        Ok(is_kde)
    }

    #[cfg(not(target_os = "linux"))]
    {
        Ok(false)
    }
}

// ============================================================================
// System Tray (Cross-DE)
// ============================================================================

#[tauri::command]
pub async fn linux_create_system_tray(icon: SystemTrayIcon) -> Result<(), String> {
    #[cfg(target_os = "linux")]
    {
        log::info!(
            "Creating Linux system tray: {} ({})",
            icon.tooltip,
            icon.icon
        );

        // System tray is handled by Tauri's built-in SystemTray
        // This would configure the tray menu
        Ok(())
    }

    #[cfg(not(target_os = "linux"))]
    {
        Err("Linux system tray only available on Linux".to_string())
    }
}

#[tauri::command]
pub async fn linux_update_tray_icon(
    icon: String,
    tooltip: Option<String>,
) -> Result<(), String> {
    #[cfg(target_os = "linux")]
    {
        log::info!("Updating Linux tray icon: {} ({:?})", icon, tooltip);

        // Update system tray icon and tooltip
        Ok(())
    }

    #[cfg(not(target_os = "linux"))]
    {
        Err("Linux system tray only available on Linux".to_string())
    }
}

#[tauri::command]
pub async fn linux_update_tray_menu(menu: Vec<SystemTrayMenuItem>) -> Result<(), String> {
    #[cfg(target_os = "linux")]
    {
        log::info!("Updating Linux tray menu with {} items", menu.len());

        // Update system tray menu
        Ok(())
    }

    #[cfg(not(target_os = "linux"))]
    {
        Err("Linux system tray only available on Linux".to_string())
    }
}

#[tauri::command]
pub async fn linux_tray_show_notification(
    title: String,
    message: String,
    icon: Option<String>,
) -> Result<(), String> {
    #[cfg(target_os = "linux")]
    {
        log::info!("Showing Linux tray notification: {} - {}", title, message);

        linux_send_notification(LinuxNotification {
            summary: title,
            body: message,
            icon,
            urgency: Some("normal".to_string()),
            timeout: Some(-1),
            actions: None,
            hints: None,
        })
        .await
    }

    #[cfg(not(target_os = "linux"))]
    {
        Err("Linux tray notifications only available on Linux".to_string())
    }
}

#[tauri::command]
pub async fn linux_remove_system_tray() -> Result<(), String> {
    #[cfg(target_os = "linux")]
    {
        log::info!("Removing Linux system tray");
        Ok(())
    }

    #[cfg(not(target_os = "linux"))]
    {
        Err("Linux system tray only available on Linux".to_string())
    }
}

// ============================================================================
// D-Bus Notifications
// ============================================================================

#[tauri::command]
pub async fn linux_send_notification(notification: LinuxNotification) -> Result<i32, String> {
    #[cfg(target_os = "linux")]
    {
        log::info!(
            "Sending D-Bus notification: {} - {}",
            notification.summary,
            notification.body
        );

        // Use Tauri's built-in notification for basic support
        tauri::api::notification::Notification::new("com.dbbackup.desktop")
            .title(&notification.summary)
            .body(&notification.body)
            .show()
            .map_err(|e| e.to_string())?;

        // For full D-Bus notification support, would use dbus crate
        // Return a mock notification ID
        Ok(1)
    }

    #[cfg(not(target_os = "linux"))]
    {
        Err("D-Bus notifications only available on Linux".to_string())
    }
}

#[tauri::command]
pub async fn linux_close_notification(notification_id: i32) -> Result<(), String> {
    #[cfg(target_os = "linux")]
    {
        log::info!("Closing Linux notification: {}", notification_id);

        // Would use D-Bus to close notification
        Ok(())
    }

    #[cfg(not(target_os = "linux"))]
    {
        Err("D-Bus notifications only available on Linux".to_string())
    }
}

// ============================================================================
// Distribution Info
// ============================================================================

#[tauri::command]
pub async fn linux_get_distribution_info() -> Result<DistributionInfo, String> {
    #[cfg(target_os = "linux")]
    {
        use std::fs;

        // Read /etc/os-release
        let os_release = fs::read_to_string("/etc/os-release")
            .unwrap_or_else(|_| String::from(""));

        let mut name = String::from("unknown");
        let mut version = String::from("unknown");
        let mut id = String::from("unknown");

        for line in os_release.lines() {
            if let Some(value) = line.strip_prefix("NAME=") {
                name = value.trim_matches('"').to_string();
            } else if let Some(value) = line.strip_prefix("VERSION=") {
                version = value.trim_matches('"').to_string();
            } else if let Some(value) = line.strip_prefix("ID=") {
                id = value.trim_matches('"').to_string();
            }
        }

        Ok(DistributionInfo { name, version, id })
    }

    #[cfg(not(target_os = "linux"))]
    {
        Err("Distribution info only available on Linux".to_string())
    }
}
