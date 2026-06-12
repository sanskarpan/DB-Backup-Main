import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

// Translation resources
const resources = {
  en: {
    translation: {
      // App
      appName: 'DB Backup Desktop',
      // Navigation
      backups: 'Backups',
      activity: 'Activity',
      settings: 'Settings',
      // Actions
      newBackup: 'New Backup',
      refresh: 'Refresh',
      export: 'Export',
      preview: 'Preview',
      restore: 'Restore',
      delete: 'Delete',
      cancel: 'Cancel',
      save: 'Save',
      close: 'Close',
      // Quick Search
      quickSearch: 'Quick Search',
      searchPlaceholder: 'Search backups, databases, settings...',
      noResults: 'No results found',
      // Backup Status
      completed: 'Completed',
      running: 'Running',
      failed: 'Failed',
      pending: 'Pending',
      // Settings
      apiUrl: 'API URL',
      apiKey: 'API Key',
      theme: 'Theme',
      language: 'Language',
      autoLaunch: 'Auto-launch on startup',
      notifications: 'Enable desktop notifications',
      keyboardShortcuts: 'Keyboard Shortcuts',
      // Themes
      light: 'Light',
      dark: 'Dark',
      auto: 'Auto',
      // Export
      exportFormat: 'Export Format',
      includeMetadata: 'Include Metadata',
      dateRange: 'Date Range',
      exportSuccess: 'Export completed successfully',
      // Messages
      loading: 'Loading...',
      error: 'Error',
      success: 'Success',
    }
  },
  es: {
    translation: {
      appName: 'Copia de Seguridad DB',
      backups: 'Copias de Seguridad',
      activity: 'Actividad',
      settings: 'Configuración',
      newBackup: 'Nueva Copia',
      refresh: 'Actualizar',
      // ... add more Spanish translations
    }
  },
  fr: {
    translation: {
      appName: 'Sauvegarde DB',
      backups: 'Sauvegardes',
      activity: 'Activité',
      settings: 'Paramètres',
      // ... add more French translations
    }
  },
  de: {
    translation: {
      appName: 'DB Sicherung',
      backups: 'Sicherungen',
      activity: 'Aktivität',
      settings: 'Einstellungen',
      // ... add more German translations
    }
  }
};

i18n
  .use(initReactI18next)
  .init({
    resources,
    lng: 'en',
    fallbackLng: 'en',
    interpolation: {
      escapeValue: false,
    },
  });

export default i18n;
