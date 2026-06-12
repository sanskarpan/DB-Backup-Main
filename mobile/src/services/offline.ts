import SQLite from 'react-native-sqlite-storage';
import NetInfo from '@react-native-community/netinfo';

SQLite.DEBUG(false);
SQLite.enablePromise(true);

class OfflineService {
  private db: SQLite.SQLiteDatabase | null = null;

  async initDatabase() {
    if (this.db) return;

    this.db = await SQLite.openDatabase({
      name: 'dbbackup.db',
      location: 'default',
    });

    await this.createTables();
  }

  private async createTables() {
    if (!this.db) return;

    await this.db.executeSql(`
      CREATE TABLE IF NOT EXISTS backups (
        id TEXT PRIMARY KEY,
        database_id TEXT,
        database_name TEXT,
        status TEXT,
        created_at TEXT,
        size INTEGER,
        duration INTEGER,
        data TEXT
      )
    `);

    await this.db.executeSql(`
      CREATE TABLE IF NOT EXISTS pending_requests (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        type TEXT,
        data TEXT,
        timestamp TEXT,
        processed INTEGER DEFAULT 0
      )
    `);

    await this.db.executeSql(`
      CREATE TABLE IF NOT EXISTS sync_status (
        id INTEGER PRIMARY KEY,
        last_sync TEXT,
        pending_count INTEGER
      )
    `);
  }

  async saveBackups(backups: any[]) {
    if (!this.db) await this.initDatabase();

    for (const backup of backups) {
      await this.db!.executeSql(
        `INSERT OR REPLACE INTO backups
         (id, database_id, database_name, status, created_at, size, duration, data)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
        [
          backup.id,
          backup.database_id,
          backup.database_name,
          backup.status,
          backup.created_at,
          backup.size || 0,
          backup.duration || 0,
          JSON.stringify(backup),
        ],
      );
    }
  }

  async getBackups() {
    if (!this.db) await this.initDatabase();

    const [results] = await this.db!.executeSql(
      'SELECT data FROM backups ORDER BY created_at DESC',
    );

    const backups: any[] = [];
    for (let i = 0; i < results.rows.length; i++) {
      backups.push(JSON.parse(results.rows.item(i).data));
    }

    return backups;
  }

  async queueBackupRequest(request: any) {
    if (!this.db) await this.initDatabase();

    await this.db!.executeSql(
      'INSERT INTO pending_requests (type, data, timestamp) VALUES (?, ?, ?)',
      ['create_backup', JSON.stringify(request), request.timestamp],
    );
  }

  async getPendingRequests() {
    if (!this.db) await this.initDatabase();

    const [results] = await this.db!.executeSql(
      'SELECT * FROM pending_requests WHERE processed = 0',
    );

    const requests: any[] = [];
    for (let i = 0; i < results.rows.length; i++) {
      const row = results.rows.item(i);
      requests.push({
        id: row.id,
        type: row.type,
        data: JSON.parse(row.data),
        timestamp: row.timestamp,
      });
    }

    return requests;
  }

  async markRequestProcessed(id: number) {
    if (!this.db) await this.initDatabase();

    await this.db!.executeSql(
      'UPDATE pending_requests SET processed = 1 WHERE id = ?',
      [id],
    );
  }

  async clearProcessedRequests() {
    if (!this.db) await this.initDatabase();

    await this.db!.executeSql('DELETE FROM pending_requests WHERE processed = 1');
  }

  async updateSyncStatus(lastSync: string, pendingCount: number) {
    if (!this.db) await this.initDatabase();

    await this.db!.executeSql(
      'INSERT OR REPLACE INTO sync_status (id, last_sync, pending_count) VALUES (1, ?, ?)',
      [lastSync, pendingCount],
    );
  }

  async getSyncStatus() {
    if (!this.db) await this.initDatabase();

    const [results] = await this.db!.executeSql(
      'SELECT * FROM sync_status WHERE id = 1',
    );

    if (results.rows.length > 0) {
      return results.rows.item(0);
    }

    return null;
  }

  async isOnline(): Promise<boolean> {
    const state = await NetInfo.fetch();
    return state.isConnected === true;
  }

  async clearCache() {
    if (!this.db) await this.initDatabase();

    await this.db!.executeSql('DELETE FROM backups');
    await this.db!.executeSql('DELETE FROM pending_requests');
    await this.db!.executeSql('DELETE FROM sync_status');
  }
}

export const offlineService = new OfflineService();
