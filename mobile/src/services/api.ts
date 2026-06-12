import { createApiClient } from '@db-backup/api-client';

// Create API service using shared API client
export const apiService = createApiClient({
  baseURL: process.env.REACT_APP_API_URL || 'http://localhost:8080/api',
  timeout: 30000,
  getAuthToken: async () => {
    // In React Native, get token from AsyncStorage
    const AsyncStorage = await import('@react-native-async-storage/async-storage');
    return await AsyncStorage.default.getItem('auth_token');
  },
});
