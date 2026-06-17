import { createApiClient } from '@db-backup/api-client';
import AsyncStorage from '@react-native-async-storage/async-storage';

// In React Native, REACT_APP_ env vars are not available.
// Define the API base URL directly here or via a constants file per environment.
const API_BASE_URL = 'http://localhost:8080/api/v1';

// Create API service using shared API client
export const apiService = createApiClient({
  baseURL: API_BASE_URL,
  timeout: 30000,
  getAuthToken: async () => {
    return AsyncStorage.getItem('auth_token');
  },
});
