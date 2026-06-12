import {configureStore} from '@reduxjs/toolkit';
import backupsReducer from './backupsSlice';
import databasesReducer from './databasesSlice';
import settingsReducer from './settingsSlice';
import offlineReducer from './offlineSlice';

export const store = configureStore({
  reducer: {
    backups: backupsReducer,
    databases: databasesReducer,
    settings: settingsReducer,
    offline: offlineReducer,
  },
  middleware: getDefaultMiddleware =>
    getDefaultMiddleware({
      serializableCheck: false,
    }),
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
