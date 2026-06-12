from locust import HttpUser, task, between, events
import random
import json

class BackupUser(HttpUser):
    wait_time = between(1, 3)

    def on_start(self):
        """Setup: Login and get auth token"""
        self.client.headers.update({
            'Authorization': 'Bearer test-api-key',
            'Content-Type': 'application/json'
        })
        self.backup_ids = []

    @task(3)
    def create_backup(self):
        """Simulate backup creation"""
        database_types = ['postgres', 'mysql', 'mongodb', 'redis']
        backup_types = ['full', 'incremental', 'differential']

        payload = {
            'database': f'test-db-{random.choice(database_types)}',
            'type': random.choice(backup_types),
            'compress': True,
            'encrypt': random.choice([True, False]),
            'tags': [f'locust-test', f'user-{self.client.base_url}']
        }

        with self.client.post('/api/v1/backups', json=payload, catch_response=True) as response:
            if response.status_code in [200, 201]:
                backup_id = response.json().get('backup_id')
                if backup_id:
                    self.backup_ids.append(backup_id)
                    response.success()
            else:
                response.failure(f'Failed with status {response.status_code}')

    @task(2)
    def list_backups(self):
        """List all backups"""
        self.client.get('/api/v1/backups')

    @task(1)
    def get_backup_status(self):
        """Check backup status"""
        if self.backup_ids:
            backup_id = random.choice(self.backup_ids)
            self.client.get(f'/api/v1/backups/{backup_id}')

    @task(1)
    def search_backups(self):
        """Search backups"""
        search_terms = ['postgres', 'full', 'completed', 'failed']
        term = random.choice(search_terms)
        self.client.get(f'/api/v1/backups/search?q={term}')

    @task(1)
    def restore_backup(self):
        """Initiate restore"""
        if self.backup_ids:
            payload = {
                'backup_id': random.choice(self.backup_ids),
                'target_database': f'restore-target-{random.randint(1, 100)}'
            }
            self.client.post('/api/v1/restore', json=payload)

class WebsocketUser(HttpUser):
    """Simulate users monitoring backups via WebSocket"""
    wait_time = between(5, 10)

    @task
    def monitor_backups(self):
        """Long-polling for backup status updates"""
        self.client.get('/api/v1/backups/stream')
