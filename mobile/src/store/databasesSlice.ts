import {createSlice, createAsyncThunk, PayloadAction} from '@reduxjs/toolkit';
import AsyncStorage from '@react-native-async-storage/async-storage';
import {apiService} from '../services/api';

interface Database {
  id: string;
  name: string;
  type: string;
  host: string;
  port?: number;
  database_name?: string;
  status?: string;
  created_at?: string;
}

interface DatabasesState {
  databases: Database[];
  loading: boolean;
  error: string | null;
}

const initialState: DatabasesState = {
  databases: [],
  loading: false,
  error: null,
};

// Async thunks
export const fetchDatabases = createAsyncThunk(
  'databases/fetchDatabases',
  async (_, {rejectWithValue}) => {
    try {
      const response = await apiService.getDatabases();

      try {
        await AsyncStorage.setItem('databases', JSON.stringify(response.data));
      } catch (storageError) {
        console.warn('[fetchDatabases] Failed to persist databases locally:', storageError);
      }

      return response.data;
    } catch (error: any) {
      if (error?.response?.status === 401 || error?.response?.status === 403) {
        return rejectWithValue('Unauthorized: please log in again');
      }

      const cachedDatabases = await AsyncStorage.getItem('databases');
      if (cachedDatabases) {
        return JSON.parse(cachedDatabases);
      }

      return rejectWithValue('Failed to load databases');
    }
  },
);

export const createDatabase = createAsyncThunk(
  'databases/createDatabase',
  async (databaseData: Partial<Database>, {rejectWithValue}) => {
    try {
      const response = await apiService.createDatabase(databaseData);
      return response.data;
    } catch (error: any) {
      return rejectWithValue(
        error?.response?.data?.message || 'Failed to create database',
      );
    }
  },
);

export const deleteDatabase = createAsyncThunk(
  'databases/deleteDatabase',
  async (databaseId: string, {rejectWithValue}) => {
    try {
      await apiService.deleteDatabase(databaseId);
      return databaseId;
    } catch (error: any) {
      return rejectWithValue(
        error?.response?.data?.message || 'Failed to delete database',
      );
    }
  },
);

// Slice
const databasesSlice = createSlice({
  name: 'databases',
  initialState,
  reducers: {
    clearError: state => {
      state.error = null;
    },
  },
  extraReducers: builder => {
    // Fetch databases
    builder.addCase(fetchDatabases.pending, state => {
      state.loading = true;
      state.error = null;
    });
    builder.addCase(fetchDatabases.fulfilled, (state, action) => {
      state.loading = false;
      state.databases = action.payload;
    });
    builder.addCase(fetchDatabases.rejected, (state, action) => {
      state.loading = false;
      state.error = action.payload as string;
    });

    // Create database
    builder.addCase(createDatabase.pending, state => {
      state.loading = true;
      state.error = null;
    });
    builder.addCase(createDatabase.fulfilled, (state, action) => {
      state.loading = false;
      state.databases.unshift(action.payload);
    });
    builder.addCase(createDatabase.rejected, (state, action) => {
      state.loading = false;
      state.error = action.payload as string;
    });

    // Delete database
    builder.addCase(deleteDatabase.pending, state => {
      state.loading = true;
      state.error = null;
    });
    builder.addCase(deleteDatabase.fulfilled, (state, action) => {
      state.loading = false;
      state.databases = state.databases.filter(db => db.id !== action.payload);
    });
    builder.addCase(deleteDatabase.rejected, (state, action) => {
      state.loading = false;
      state.error = action.payload as string;
    });
  },
});

export const {clearError} = databasesSlice.actions;
export default databasesSlice.reducer;
