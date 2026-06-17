import {createSlice, PayloadAction} from '@reduxjs/toolkit';

interface OfflineState {
  isOnline: boolean;
  pendingCount: number;
  lastSyncTime: string | null;
  syncError: string | null;
}

const initialState: OfflineState = {
  isOnline: true,
  pendingCount: 0,
  lastSyncTime: null,
  syncError: null,
};

// Slice
const offlineSlice = createSlice({
  name: 'offline',
  initialState,
  reducers: {
    setOnlineStatus: (state, action: PayloadAction<boolean>) => {
      state.isOnline = action.payload;
      if (action.payload) {
        // Came back online — clear sync error
        state.syncError = null;
      }
    },
    setPendingCount: (state, action: PayloadAction<number>) => {
      state.pendingCount = action.payload;
    },
    setSyncError: (state, action: PayloadAction<string | null>) => {
      state.syncError = action.payload;
    },
    setSyncTime: (state, action: PayloadAction<string>) => {
      state.lastSyncTime = action.payload;
      state.syncError = null;
    },
  },
});

export const {setOnlineStatus, setPendingCount, setSyncError, setSyncTime} =
  offlineSlice.actions;
export default offlineSlice.reducer;
