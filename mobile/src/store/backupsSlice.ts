import {createSlice, createAsyncThunk, PayloadAction} from '@reduxjs/toolkit';
import AsyncStorage from '@react-native-async-storage/async-storage';
import {offlineService} from '../services/offline';
import {apiService} from '../services/api';

interface Backup {
  id: string;
  database_id: string;
  database_name: string;
  status: string;
  created_at: string;
  size?: number;
  duration?: number;
}

interface BackupsState {
  backups: Backup[];
  loading: boolean;
  error: string | null;
  syncing: boolean;
  lastSync: string | null;
  isCachedData: boolean;
}

const initialState: BackupsState = {
  backups: [],
  loading: false,
  error: null,
  syncing: false,
  lastSync: null,
  isCachedData: false,
};

// Async thunks
export const fetchBackups = createAsyncThunk(
  'backups/fetchBackups',
  async (_, {rejectWithValue}) => {
    try {
      // Try to fetch from API. listBackups() returns the parsed
      // BackupListResponse payload directly ({ backups, total, page, page_size }).
      const response = await apiService.listBackups();
      const backups = response.backups ?? [];

      // Save to local storage for offline access
      try {
        await AsyncStorage.setItem('backups', JSON.stringify(backups));
        await offlineService.saveBackups(backups);
      } catch (storageError) {
        console.warn('[fetchBackups] Failed to persist backups locally:', storageError);
      }

      return {data: backups, isCachedData: false};
    } catch (error: any) {
      // Auth errors should not fall back to cache
      if (error?.response?.status === 401 || error?.response?.status === 403) {
        return rejectWithValue('Unauthorized: please log in again');
      }

      // For other errors (network, 5xx), fall back to cache
      const cachedBackups = await AsyncStorage.getItem('backups');
      if (cachedBackups) {
        return {data: JSON.parse(cachedBackups), isCachedData: true};
      }

      // Try loading from SQLite
      const offlineBackups = await offlineService.getBackups();
      if (offlineBackups.length > 0) {
        return {data: offlineBackups, isCachedData: true};
      }

      return rejectWithValue('Failed to load backups');
    }
  },
);

export const createBackup = createAsyncThunk(
  'backups/createBackup',
  async (
    {databaseId, options}: {databaseId: string; options: any},
    {rejectWithValue},
  ) => {
    try {
      // The client takes a single config object; map our thunk args onto it.
      const backup = await apiService.createBackup({
        database_id: databaseId,
        ...(options || {}),
      });
      // The shared Backup type marks database_name optional; the slice's local
      // Backup shape requires it, so narrow to the slice type.
      return backup as unknown as (typeof initialState.backups)[number];
    } catch (error) {
      // Queue for offline sync
      await offlineService.queueBackupRequest({
        databaseId,
        options,
        timestamp: new Date().toISOString(),
      });
      return rejectWithValue('Queued for sync when online');
    }
  },
);

export const syncOfflineData = createAsyncThunk(
  'backups/syncOfflineData',
  async (_, {dispatch}) => {
    try {
      // Get pending requests from offline queue
      const pendingRequests = await offlineService.getPendingRequests();

      for (const request of pendingRequests) {
        if (request.type === 'create_backup') {
          try {
            await apiService.createBackup({
              database_id: request.data.databaseId,
              ...(request.data.options || {}),
            });
            await offlineService.markRequestProcessed(request.id);
          } catch (requestError) {
            console.error(
              `[syncOfflineData] Failed to sync request ${request.id}:`,
              requestError,
            );
            // Continue processing remaining requests
          }
        }
      }

      // Fetch latest data
      await dispatch(fetchBackups());

      return {success: true};
    } catch (error) {
      console.error('[syncOfflineData] Sync failed:', error);
      throw error;
    }
  },
);

// Slice
const backupsSlice = createSlice({
  name: 'backups',
  initialState,
  reducers: {
    clearError: state => {
      state.error = null;
    },
    updateBackupStatus: (
      state,
      action: PayloadAction<{id: string; status: string}>,
    ) => {
      const backup = state.backups.find(b => b.id === action.payload.id);
      if (backup) {
        backup.status = action.payload.status;
      }
    },
  },
  extraReducers: builder => {
    // Fetch backups
    builder.addCase(fetchBackups.pending, state => {
      state.loading = true;
      state.error = null;
    });
    builder.addCase(fetchBackups.fulfilled, (state, action) => {
      state.loading = false;
      state.backups = action.payload.data;
      state.isCachedData = action.payload.isCachedData;
      state.lastSync = new Date().toISOString();
    });
    builder.addCase(fetchBackups.rejected, (state, action) => {
      state.loading = false;
      state.error = action.payload as string;
    });

    // Create backup
    builder.addCase(createBackup.pending, state => {
      state.loading = true;
      state.error = null;
    });
    builder.addCase(createBackup.fulfilled, (state, action) => {
      state.loading = false;
      state.backups.unshift(action.payload);
    });
    builder.addCase(createBackup.rejected, (state, action) => {
      state.loading = false;
      state.error = action.payload as string;
    });

    // Sync offline data
    builder.addCase(syncOfflineData.pending, state => {
      state.syncing = true;
    });
    builder.addCase(syncOfflineData.fulfilled, state => {
      state.syncing = false;
      state.lastSync = new Date().toISOString();
    });
    builder.addCase(syncOfflineData.rejected, state => {
      state.syncing = false;
    });
  },
});

export const {clearError, updateBackupStatus} = backupsSlice.actions;
export default backupsSlice.reducer;
