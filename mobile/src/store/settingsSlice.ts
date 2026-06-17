import {createSlice, createAsyncThunk, PayloadAction} from '@reduxjs/toolkit';
import AsyncStorage from '@react-native-async-storage/async-storage';

interface SettingsState {
  apiUrl: string;
  theme: 'light' | 'dark';
  notifications: boolean;
  autoSync: boolean;
  authToken: string | null;
}

const initialState: SettingsState = {
  apiUrl: 'http://localhost:8080/api/v1',
  theme: 'light',
  notifications: true,
  autoSync: true,
  authToken: null,
};

const SETTINGS_STORAGE_KEY = 'app_settings';

// Load settings from AsyncStorage on app start
export const loadSettings = createAsyncThunk(
  'settings/loadSettings',
  async (_, {rejectWithValue}) => {
    try {
      const [stored, authToken] = await Promise.all([
        AsyncStorage.getItem(SETTINGS_STORAGE_KEY),
        AsyncStorage.getItem('auth_token'),
      ]);
      const parsed: Partial<SettingsState> = stored
        ? (JSON.parse(stored) as Partial<SettingsState>)
        : {};
      if (authToken) {
        parsed.authToken = authToken;
      }
      return parsed;
    } catch (error) {
      console.warn('[loadSettings] Failed to load settings:', error);
      return rejectWithValue('Failed to load settings');
    }
  },
);

// Persist settings to AsyncStorage
const persistSettings = async (settings: SettingsState) => {
  try {
    await AsyncStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(settings));
  } catch (error) {
    console.warn('[persistSettings] Failed to save settings:', error);
  }
};

// Slice
const settingsSlice = createSlice({
  name: 'settings',
  initialState,
  reducers: {
    setApiUrl: (state, action: PayloadAction<string>) => {
      state.apiUrl = action.payload;
      persistSettings({...state, apiUrl: action.payload});
    },
    setTheme: (state, action: PayloadAction<'light' | 'dark'>) => {
      state.theme = action.payload;
      persistSettings({...state, theme: action.payload});
    },
    toggleNotifications: state => {
      state.notifications = !state.notifications;
      persistSettings({...state});
    },
    toggleAutoSync: state => {
      state.autoSync = !state.autoSync;
      persistSettings({...state});
    },
    setAuthToken: (state, action: PayloadAction<string | null>) => {
      state.authToken = action.payload;
    },
  },
  extraReducers: builder => {
    builder.addCase(loadSettings.fulfilled, (state, action) => {
      const loaded = action.payload as Partial<SettingsState>;
      if (loaded.apiUrl !== undefined) {
        state.apiUrl = loaded.apiUrl;
      }
      if (loaded.theme !== undefined) {
        state.theme = loaded.theme;
      }
      if (loaded.notifications !== undefined) {
        state.notifications = loaded.notifications;
      }
      if (loaded.autoSync !== undefined) {
        state.autoSync = loaded.autoSync;
      }
      if (loaded.authToken !== undefined) {
        state.authToken = loaded.authToken;
      }
    });
  },
});

export const {
  setApiUrl,
  setTheme,
  toggleNotifications,
  toggleAutoSync,
  setAuthToken,
} = settingsSlice.actions;
export default settingsSlice.reducer;
