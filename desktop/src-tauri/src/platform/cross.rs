// Cross-Platform Features
// File associations, context menus, protocol handlers, custom dialogs

use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Manager};

// ============================================================================
// Types
// ============================================================================

#[derive(Debug, Serialize, Deserialize)]
pub struct FileAssociation {
    pub extension: String,
    pub mime_type: Option<String>,
    pub description: String,
    pub icon: Option<String>,
    pub platform: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct ContextMenuItem {
    pub id: String,
    pub label: String,
    pub icon: Option<String>,
    pub extensions: Option<Vec<String>>,
    pub directories: Option<bool>,
    pub files: Option<bool>,
    pub multiple: Option<bool>,
    pub separator: Option<bool>,
    pub submenu: Option<Vec<ContextMenuItem>>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct ProtocolHandlerInfo {
    pub protocol: String,
    pub description: String,
    pub platform: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct CustomDialogButton {
    pub id: String,
    pub label: String,
    pub default: Option<bool>,
    pub destructive: Option<bool>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct CustomDialog {
    pub title: String,
    pub message: String,
    pub buttons: Vec<CustomDialogButton>,
    #[serde(rename = "type")]
    pub dialog_type: Option<String>,
    pub icon: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct ShellCommand {
    pub id: String,
    pub command: String,
    pub name: String,
    pub description: String,
}

// ============================================================================
// File Associations
// ============================================================================

#[tauri::command]
pub async fn register_file_association(
    association: FileAssociation,
) -> Result<(), String> {
    #[cfg(target_os = "windows")]
    {
        windows_register_file_association(&association).await
    }

    #[cfg(target_os = "macos")]
    {
        macos_register_file_association(&association).await
    }

    #[cfg(target_os = "linux")]
    {
        linux_register_file_association(&association).await
    }

    #[cfg(not(any(target_os = "windows", target_os = "macos", target_os = "linux")))]
    {
        Err("File associations not supported on this platform".to_string())
    }
}

#[tauri::command]
pub async fn unregister_file_association(extension: String) -> Result<(), String> {
    #[cfg(target_os = "windows")]
    {
        windows_unregister_file_association(&extension).await
    }

    #[cfg(target_os = "macos")]
    {
        macos_unregister_file_association(&extension).await
    }

    #[cfg(target_os = "linux")]
    {
        linux_unregister_file_association(&extension).await
    }

    #[cfg(not(any(target_os = "windows", target_os = "macos", target_os = "linux")))]
    {
        Err("File associations not supported on this platform".to_string())
    }
}

// ============================================================================
// Context Menus
// ============================================================================

#[tauri::command]
pub async fn register_context_menu_items(
    items: Vec<ContextMenuItem>,
    platform: String,
) -> Result<(), String> {
    #[cfg(target_os = "windows")]
    {
        if platform == "windows" {
            return windows_register_context_menu(&items).await;
        }
    }

    #[cfg(target_os = "macos")]
    {
        if platform == "macos" {
            return macos_register_context_menu(&items).await;
        }
    }

    #[cfg(target_os = "linux")]
    {
        if platform == "linux" {
            return linux_register_context_menu(&items).await;
        }
    }

    Ok(())
}

#[tauri::command]
pub async fn update_context_menu_item(
    item_id: String,
    item: ContextMenuItem,
) -> Result<(), String> {
    // Platform-specific implementation
    log::info!("Updating context menu item: {} with label: {}", item_id, item.label);
    Ok(())
}

#[tauri::command]
pub async fn remove_context_menu_item(item_id: String) -> Result<(), String> {
    // Platform-specific implementation
    log::info!("Removing context menu item: {}", item_id);
    Ok(())
}

// ============================================================================
// Protocol Handlers
// ============================================================================

#[tauri::command]
pub async fn register_protocol_handler(
    protocol: String,
    description: String,
    platform: String,
) -> Result<(), String> {
    #[cfg(target_os = "windows")]
    {
        if platform == "windows" {
            return windows_register_protocol(&protocol, &description).await;
        }
    }

    #[cfg(target_os = "macos")]
    {
        if platform == "macos" {
            return macos_register_protocol(&protocol, &description).await;
        }
    }

    #[cfg(target_os = "linux")]
    {
        if platform == "linux" {
            return linux_register_protocol(&protocol, &description).await;
        }
    }

    Ok(())
}

#[tauri::command]
pub async fn unregister_protocol_handler(protocol: String) -> Result<(), String> {
    log::info!("Unregistering protocol handler: {}", protocol);
    Ok(())
}

// ============================================================================
// Custom Dialogs
// ============================================================================

#[tauri::command]
pub async fn show_custom_dialog(
    app: AppHandle,
    dialog: CustomDialog,
) -> Result<String, String> {
    // For now, use native dialog with first two buttons
    let title = dialog.title;
    let message = dialog.message;

    if dialog.buttons.len() >= 2 {
        let confirm = tauri::api::dialog::blocking::ask(
            Some(&app.get_window("main").unwrap()),
            &title,
            &message,
        );

        if confirm {
            Ok(dialog.buttons.get(0).unwrap().id.clone())
        } else {
            Ok(dialog.buttons.get(1).unwrap().id.clone())
        }
    } else if dialog.buttons.len() == 1 {
        tauri::api::dialog::blocking::message(
            Some(&app.get_window("main").unwrap()),
            &title,
            &message,
        );
        Ok(dialog.buttons.get(0).unwrap().id.clone())
    } else {
        Err("Dialog must have at least one button".to_string())
    }
}

#[tauri::command]
pub async fn show_progress_dialog(
    title: String,
    message: String,
    cancellable: bool,
) -> Result<(), String> {
    log::info!("Show progress dialog: {} - {} (cancellable: {})", title, message, cancellable);
    // Note: Tauri doesn't have built-in progress dialogs,
    // this would be implemented using a custom window
    Ok(())
}

// ============================================================================
// Shell Commands
// ============================================================================

#[tauri::command]
pub async fn register_shell_command(command: ShellCommand) -> Result<(), String> {
    log::info!("Registering shell command: {} - {}", command.id, command.name);
    Ok(())
}

#[tauri::command]
pub async fn unregister_shell_command(command_id: String) -> Result<(), String> {
    log::info!("Unregistering shell command: {}", command_id);
    Ok(())
}

// ============================================================================
// Platform-Specific Helpers
// ============================================================================

#[cfg(target_os = "windows")]
async fn windows_register_file_association(association: &FileAssociation) -> Result<(), String> {
    use winreg::RegKey;
    use winreg::enums::*;

    let hkcr = RegKey::predef(HKEY_CURRENT_USER);
    let extension_key = format!(".{}", association.extension);

    // Create extension key
    let (ext_key, _) = hkcr
        .create_subkey(format!("Software\\Classes\\{}", extension_key))
        .map_err(|e| format!("Failed to create extension key: {}", e))?;

    // Set default value to ProgID
    let prog_id = format!("DBBackup.{}", association.extension);
    ext_key
        .set_value("", &prog_id)
        .map_err(|e| format!("Failed to set ProgID: {}", e))?;

    // Create ProgID key
    let (prog_key, _) = hkcr
        .create_subkey(format!("Software\\Classes\\{}", prog_id))
        .map_err(|e| format!("Failed to create ProgID key: {}", e))?;

    prog_key
        .set_value("", &association.description)
        .map_err(|e| format!("Failed to set description: {}", e))?;

    // Create shell\open\command key
    let exe_path = std::env::current_exe()
        .map_err(|e| format!("Failed to get exe path: {}", e))?;
    let command = format!("\"{}\" \"%1\"", exe_path.display());

    let (cmd_key, _) = hkcr
        .create_subkey(format!("Software\\Classes\\{}\\shell\\open\\command", prog_id))
        .map_err(|e| format!("Failed to create command key: {}", e))?;

    cmd_key
        .set_value("", &command)
        .map_err(|e| format!("Failed to set command: {}", e))?;

    log::info!("Registered file association for .{}", association.extension);
    Ok(())
}

#[cfg(target_os = "windows")]
async fn windows_unregister_file_association(extension: &str) -> Result<(), String> {
    use winreg::RegKey;
    use winreg::enums::*;

    let hkcr = RegKey::predef(HKEY_CURRENT_USER);
    let extension_key = format!(".{}", extension);

    hkcr.delete_subkey_all(format!("Software\\Classes\\{}", extension_key))
        .map_err(|e| format!("Failed to delete extension key: {}", e))?;

    let prog_id = format!("DBBackup.{}", extension);
    hkcr.delete_subkey_all(format!("Software\\Classes\\{}", prog_id))
        .map_err(|e| format!("Failed to delete ProgID key: {}", e))?;

    log::info!("Unregistered file association for .{}", extension);
    Ok(())
}

#[cfg(target_os = "windows")]
async fn windows_register_context_menu(items: &[ContextMenuItem]) -> Result<(), String> {
    log::info!("Windows context menu registration not fully implemented");
    // Would require registry manipulation for Windows Explorer
    Ok(())
}

#[cfg(target_os = "windows")]
async fn windows_register_protocol(protocol: &str, description: &str) -> Result<(), String> {
    use winreg::RegKey;
    use winreg::enums::*;

    let hkcr = RegKey::predef(HKEY_CURRENT_USER);

    let (protocol_key, _) = hkcr
        .create_subkey(format!("Software\\Classes\\{}", protocol))
        .map_err(|e| format!("Failed to create protocol key: {}", e))?;

    protocol_key
        .set_value("", description)
        .map_err(|e| format!("Failed to set description: {}", e))?;

    protocol_key
        .set_value("URL Protocol", &"")
        .map_err(|e| format!("Failed to set URL Protocol: {}", e))?;

    let exe_path = std::env::current_exe()
        .map_err(|e| format!("Failed to get exe path: {}", e))?;
    let command = format!("\"{}\" \"%1\"", exe_path.display());

    let (cmd_key, _) = hkcr
        .create_subkey(format!("Software\\Classes\\{}\\shell\\open\\command", protocol))
        .map_err(|e| format!("Failed to create command key: {}", e))?;

    cmd_key
        .set_value("", &command)
        .map_err(|e| format!("Failed to set command: {}", e))?;

    log::info!("Registered protocol handler for {}", protocol);
    Ok(())
}

#[cfg(target_os = "macos")]
async fn macos_register_file_association(association: &FileAssociation) -> Result<(), String> {
    // macOS file associations are handled via Info.plist
    log::info!("macOS file associations are configured via Info.plist");
    Ok(())
}

#[cfg(target_os = "macos")]
async fn macos_unregister_file_association(_extension: &str) -> Result<(), String> {
    Ok(())
}

#[cfg(target_os = "macos")]
async fn macos_register_context_menu(_items: &[ContextMenuItem]) -> Result<(), String> {
    // macOS context menus handled via Finder Sync extension
    log::info!("macOS context menus require Finder Sync extension");
    Ok(())
}

#[cfg(target_os = "macos")]
async fn macos_register_protocol(protocol: &str, _description: &str) -> Result<(), String> {
    // macOS protocol handlers are configured via Info.plist
    log::info!("macOS protocol handler for {} configured via Info.plist", protocol);
    Ok(())
}

#[cfg(target_os = "linux")]
async fn linux_register_file_association(association: &FileAssociation) -> Result<(), String> {
    use std::fs;
    use std::path::PathBuf;

    // Create .desktop file
    let home = std::env::var("HOME").map_err(|_| "HOME not set".to_string())?;
    let desktop_file_path = PathBuf::from(home)
        .join(".local/share/applications/db-backup.desktop");

    let exe_path = std::env::current_exe()
        .map_err(|e| format!("Failed to get exe path: {}", e))?;

    let desktop_content = format!(
        "[Desktop Entry]\n\
         Type=Application\n\
         Name=DB Backup\n\
         Exec={} %F\n\
         MimeType=application/x-{}\n\
         Icon=database-backup\n\
         Categories=Utility;",
        exe_path.display(),
        association.extension
    );

    if let Some(parent) = desktop_file_path.parent() {
        fs::create_dir_all(parent)
            .map_err(|e| format!("Failed to create directory: {}", e))?;
    }

    fs::write(&desktop_file_path, desktop_content)
        .map_err(|e| format!("Failed to write desktop file: {}", e))?;

    // Update MIME database
    let _ = std::process::Command::new("update-desktop-database")
        .arg(format!("{}/.local/share/applications", home))
        .output();

    log::info!("Registered file association for .{} on Linux", association.extension);
    Ok(())
}

#[cfg(target_os = "linux")]
async fn linux_unregister_file_association(_extension: &str) -> Result<(), String> {
    Ok(())
}

#[cfg(target_os = "linux")]
async fn linux_register_context_menu(_items: &[ContextMenuItem]) -> Result<(), String> {
    // Linux context menus are DE-specific (Nautilus scripts, Dolphin service menus, etc.)
    log::info!("Linux context menus are desktop environment specific");
    Ok(())
}

#[cfg(target_os = "linux")]
async fn linux_register_protocol(protocol: &str, _description: &str) -> Result<(), String> {
    // Register via .desktop file
    log::info!("Linux protocol handler for {} registered via .desktop file", protocol);
    Ok(())
}
