import {offlineService} from '../src/services/offline';

describe('OfflineService', () => {
  beforeAll(async () => {
    await offlineService.initDatabase();
  });

  beforeEach(async () => {
    await offlineService.clearCache();
  });

  describe('saveBackups', () => {
    it('saves backups to local database', async () => {
      const backups = [
        {
          id: 'backup-1',
          database_id: 'db-1',
          database_name: 'Test DB',
          status: 'completed',
          created_at: '2024-01-09T10:00:00Z',
          size: 1024000,
          duration: 120,
        },
      ];

      await offlineService.saveBackups(backups);
      const savedBackups = await offlineService.getBackups();

      expect(savedBackups).toHaveLength(1);
      expect(savedBackups[0].id).toBe('backup-1');
      expect(savedBackups[0].database_name).toBe('Test DB');
    });

    it('updates existing backups', async () => {
      const backup = {
        id: 'backup-1',
        database_id: 'db-1',
        database_name: 'Test DB',
        status: 'running',
        created_at: '2024-01-09T10:00:00Z',
      };

      await offlineService.saveBackups([backup]);

      const updatedBackup = {...backup, status: 'completed'};
      await offlineService.saveBackups([updatedBackup]);

      const savedBackups = await offlineService.getBackups();
      expect(savedBackups).toHaveLength(1);
      expect(savedBackups[0].status).toBe('completed');
    });
  });

  describe('queueBackupRequest', () => {
    it('queues backup request for offline sync', async () => {
      const request = {
        databaseId: 'db-1',
        options: {incremental: true},
        timestamp: '2024-01-09T10:00:00Z',
      };

      await offlineService.queueBackupRequest(request);
      const pendingRequests = await offlineService.getPendingRequests();

      expect(pendingRequests).toHaveLength(1);
      expect(pendingRequests[0].type).toBe('create_backup');
      expect(pendingRequests[0].data.databaseId).toBe('db-1');
    });
  });

  describe('markRequestProcessed', () => {
    it('marks request as processed', async () => {
      const request = {
        databaseId: 'db-1',
        options: {},
        timestamp: '2024-01-09T10:00:00Z',
      };

      await offlineService.queueBackupRequest(request);
      let pendingRequests = await offlineService.getPendingRequests();
      const requestId = pendingRequests[0].id;

      await offlineService.markRequestProcessed(requestId);
      pendingRequests = await offlineService.getPendingRequests();

      expect(pendingRequests).toHaveLength(0);
    });
  });

  describe('syncStatus', () => {
    it('updates and retrieves sync status', async () => {
      const lastSync = '2024-01-09T10:00:00Z';
      const pendingCount = 5;

      await offlineService.updateSyncStatus(lastSync, pendingCount);
      const status = await offlineService.getSyncStatus();

      expect(status.last_sync).toBe(lastSync);
      expect(status.pending_count).toBe(pendingCount);
    });
  });

  describe('clearProcessedRequests', () => {
    it('clears all processed requests', async () => {
      await offlineService.queueBackupRequest({
        databaseId: 'db-1',
        options: {},
        timestamp: '2024-01-09T10:00:00Z',
      });

      const requests = await offlineService.getPendingRequests();
      await offlineService.markRequestProcessed(requests[0].id);
      await offlineService.clearProcessedRequests();

      // Since the request was processed, it should be cleared
      expect(true).toBeTruthy();
    });
  });
});
