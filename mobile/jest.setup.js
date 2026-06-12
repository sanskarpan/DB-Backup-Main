import '@testing-library/jest-native/extend-expect';

// Mock AsyncStorage
jest.mock('@react-native-async-storage/async-storage', () =>
  require('@react-native-async-storage/async-storage/jest/async-storage-mock')
);

// Mock React Native modules
jest.mock('react-native-vector-icons/MaterialIcons', () => 'Icon');
jest.mock('react-native-push-notification', () => ({
  configure: jest.fn(),
  localNotification: jest.fn(),
}));
jest.mock('react-native-background-fetch', () => ({
  configure: jest.fn(),
  start: jest.fn(),
  stop: jest.fn(),
}));
jest.mock('react-native-device-info', () => {
  const mockMethods = {
    getVersion: () => '1.0.0',
    getBuildNumber: () => '1',
    getBatteryLevel: jest.fn(() => Promise.resolve(0.85)),
    isBatteryCharging: jest.fn(() => Promise.resolve(false)),
    isCharging: jest.fn(() => Promise.resolve(false)),
    getPowerState: jest.fn(() => Promise.resolve({
      batteryLevel: 0.85,
      batteryState: 'unplugged',
      lowPowerMode: false
    })),
  };

  return {
    __esModule: true,
    default: mockMethods,
    ...mockMethods,
  };
});
jest.mock('@react-native-community/netinfo', () => ({
  fetch: jest.fn(() => Promise.resolve({ isConnected: true })),
  addEventListener: jest.fn(),
}));

// Mock SQLite with in-memory storage
const mockDatabase = {
  backups: [],
  pending_requests: [],
  sync_status: null,
};

// Reset mock database before each test
beforeEach(() => {
  mockDatabase.backups = [];
  mockDatabase.pending_requests = [];
  mockDatabase.sync_status = null;
});

jest.mock('react-native-sqlite-storage', () => ({
  DEBUG: jest.fn(),
  enablePromise: jest.fn(),
  openDatabase: jest.fn(() => Promise.resolve({
    transaction: jest.fn((callback) => {
      const mockTransaction = {
        executeSql: jest.fn((query, params, successCallback) => {
          if (successCallback) {
            successCallback(null, { rows: { length: 0, item: () => ({}) } });
          }
        }),
      };
      callback(mockTransaction);
      return Promise.resolve();
    }),
    executeSql: jest.fn((query, params) => {
      // Handle different SQL queries
      const queryLower = query.toLowerCase();

      if (queryLower.includes('insert or replace into backups')) {
        // Insert backup with 8 params: id, database_id, database_name, status, created_at, size, duration, data
        const backupData = {
          id: params[0],
          database_id: params[1],
          database_name: params[2],
          status: params[3],
          created_at: params[4],
          size: params[5],
          duration: params[6],
          data: params[7], // JSON string
        };
        // Remove existing backup with same id
        mockDatabase.backups = mockDatabase.backups.filter(b => {
          const existingData = JSON.parse(b.data);
          return existingData.id !== params[0];
        });
        mockDatabase.backups.push(backupData);
        return Promise.resolve([{ rows: { length: 0, item: () => ({}), raw: () => [] } }]);
      } else if (queryLower.includes('select data from backups')) {
        // Select backups
        return Promise.resolve([
          {
            rows: {
              length: mockDatabase.backups.length,
              item: (i) => mockDatabase.backups[i],
              raw: () => mockDatabase.backups
            }
          }
        ]);
      } else if (queryLower.includes('insert into pending_requests')) {
        // Insert pending request with 3 params: type, data (JSON string), timestamp
        const request = {
          id: Date.now(),
          type: params[0],
          data: params[1], // JSON string
          timestamp: params[2],
          processed: 0
        };
        mockDatabase.pending_requests.push(request);
        return Promise.resolve([{ rows: { length: 0, item: () => ({}), raw: () => [] } }]);
      } else if (queryLower.includes('select * from pending_requests where processed = 0')) {
        // Select pending requests
        const pending = mockDatabase.pending_requests.filter(r => r.processed === 0);
        return Promise.resolve([
          {
            rows: {
              length: pending.length,
              item: (i) => pending[i],
              raw: () => pending
            }
          }
        ]);
      } else if (queryLower.includes('update pending_requests set processed')) {
        // Mark as processed
        const id = params[0];
        const request = mockDatabase.pending_requests.find(r => r.id === id);
        if (request) {
          request.processed = 1;
        }
        return Promise.resolve([{ rows: { length: 0, item: () => ({}), raw: () => [] } }]);
      } else if (queryLower.includes('delete from pending_requests where processed')) {
        // Clear processed
        mockDatabase.pending_requests = mockDatabase.pending_requests.filter(r => r.processed === 0);
        return Promise.resolve([{ rows: { length: 0, item: () => ({}), raw: () => [] } }]);
      } else if (queryLower.includes('select * from sync_status')) {
        // Get sync status
        return Promise.resolve([
          {
            rows: {
              length: mockDatabase.sync_status ? 1 : 0,
              item: () => mockDatabase.sync_status || {},
              raw: () => mockDatabase.sync_status ? [mockDatabase.sync_status] : []
            }
          }
        ]);
      } else if (queryLower.includes('insert or replace into sync_status')) {
        // Update sync status with 3 params: id (always 1), last_sync, pending_count
        mockDatabase.sync_status = { id: 1, last_sync: params[0], pending_count: params[1] };
        return Promise.resolve([{ rows: { length: 0, item: () => ({}), raw: () => [] } }]);
      }

      return Promise.resolve([
        {
          rows: {
            length: 0,
            item: () => ({}),
            raw: () => []
          }
        }
      ]);
    }),
  })),
}));

// Mock Geolocation
jest.mock('@react-native-community/geolocation', () => {
  const mockMethods = {
    requestAuthorization: jest.fn(() => Promise.resolve('granted')),
    getCurrentPosition: jest.fn((success) => {
      success({
        coords: {
          latitude: 37.7749,
          longitude: -122.4194,
          accuracy: 5,
        },
      });
    }),
    watchPosition: jest.fn(() => 1),
    clearWatch: jest.fn(),
    stopObserving: jest.fn(),
  };

  return {
    __esModule: true,
    default: mockMethods,
    ...mockMethods,
  };
});
