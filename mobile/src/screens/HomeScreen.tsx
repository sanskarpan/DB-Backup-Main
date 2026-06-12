import React, {useEffect, useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  RefreshControl,
  ActivityIndicator,
} from 'react-native';
import Icon from 'react-native-vector-icons/Ionicons';
import {useDispatch, useSelector} from 'react-redux';
import {fetchBackups} from '../store/backupsSlice';

export default function HomeScreen({navigation}: any) {
  const [refreshing, setRefreshing] = useState(false);
  const dispatch = useDispatch();
  const {backups, loading} = useSelector((state: any) => state.backups);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = () => {
    dispatch(fetchBackups() as any);
  };

  const onRefresh = () => {
    setRefreshing(true);
    loadData();
    setTimeout(() => setRefreshing(false), 1000);
  };

  const stats = {
    totalBackups: backups?.length || 0,
    completedBackups:
      backups?.filter((b: any) => b.status === 'completed').length || 0,
    runningBackups:
      backups?.filter((b: any) => b.status === 'running').length || 0,
    failedBackups:
      backups?.filter((b: any) => b.status === 'failed').length || 0,
  };

  return (
    <ScrollView
      style={styles.container}
      refreshControl={
        <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
      }>
      <View style={styles.header}>
        <Text style={styles.headerTitle}>DB Backup</Text>
        <Text style={styles.headerSubtitle}>Dashboard</Text>
      </View>

      {/* Stats Grid */}
      <View style={styles.statsGrid}>
        <StatCard
          icon="folder"
          title="Total Backups"
          value={stats.totalBackups}
          color="#3b82f6"
        />
        <StatCard
          icon="checkmark-circle"
          title="Completed"
          value={stats.completedBackups}
          color="#10b981"
        />
        <StatCard
          icon="time"
          title="Running"
          value={stats.runningBackups}
          color="#f59e0b"
        />
        <StatCard
          icon="close-circle"
          title="Failed"
          value={stats.failedBackups}
          color="#ef4444"
        />
      </View>

      {/* Quick Actions */}
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Quick Actions</Text>
        <View style={styles.actionsGrid}>
          <ActionButton
            icon="add-circle"
            title="New Backup"
            onPress={() => navigation.navigate('Backups')}
          />
          <ActionButton
            icon="list"
            title="View All"
            onPress={() => navigation.navigate('Backups')}
          />
          <ActionButton
            icon="server"
            title="Databases"
            onPress={() => navigation.navigate('Databases')}
          />
          <ActionButton
            icon="settings"
            title="Settings"
            onPress={() => navigation.navigate('Settings')}
          />
        </View>
      </View>

      {/* Recent Backups */}
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Recent Backups</Text>
        {loading && backups.length === 0 ? (
          <ActivityIndicator size="large" color="#3b82f6" />
        ) : backups.length === 0 ? (
          <View style={styles.emptyState}>
            <Icon name="folder-open-outline" size={64} color="#9ca3af" />
            <Text style={styles.emptyText}>No backups yet</Text>
          </View>
        ) : (
          backups.slice(0, 5).map((backup: any) => (
            <TouchableOpacity
              key={backup.id}
              style={styles.backupCard}
              onPress={() =>
                navigation.navigate('BackupDetail', {backup})
              }>
              <View style={styles.backupIcon}>
                <Icon name="folder" size={24} color="#3b82f6" />
              </View>
              <View style={styles.backupInfo}>
                <Text style={styles.backupName}>{backup.database_name}</Text>
                <Text style={styles.backupTime}>
                  {new Date(backup.created_at).toLocaleString()}
                </Text>
              </View>
              <StatusBadge status={backup.status} />
            </TouchableOpacity>
          ))
        )}
      </View>
    </ScrollView>
  );
}

function StatCard({icon, title, value, color}: any) {
  return (
    <View style={[styles.statCard, {borderLeftColor: color}]}>
      <Icon name={icon} size={24} color={color} style={styles.statIcon} />
      <Text style={styles.statValue}>{value}</Text>
      <Text style={styles.statTitle}>{title}</Text>
    </View>
  );
}

function ActionButton({icon, title, onPress}: any) {
  return (
    <TouchableOpacity style={styles.actionButton} onPress={onPress}>
      <Icon name={icon} size={32} color="#3b82f6" />
      <Text style={styles.actionTitle}>{title}</Text>
    </TouchableOpacity>
  );
}

function StatusBadge({status}: {status: string}) {
  const colors: Record<string, {bg: string; text: string}> = {
    completed: {bg: '#d1fae5', text: '#065f46'},
    running: {bg: '#dbeafe', text: '#1e40af'},
    failed: {bg: '#fee2e2', text: '#991b1b'},
    pending: {bg: '#fef3c7', text: '#92400e'},
  };

  const color = colors[status] || colors.pending;

  return (
    <View style={[styles.badge, {backgroundColor: color.bg}]}>
      <Text style={[styles.badgeText, {color: color.text}]}>
        {status.charAt(0).toUpperCase() + status.slice(1)}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f9fafb',
  },
  header: {
    backgroundColor: '#3b82f6',
    padding: 24,
    paddingTop: 48,
  },
  headerTitle: {
    fontSize: 28,
    fontWeight: 'bold',
    color: 'white',
  },
  headerSubtitle: {
    fontSize: 16,
    color: '#dbeafe',
    marginTop: 4,
  },
  statsGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    padding: 12,
    gap: 12,
  },
  statCard: {
    flex: 1,
    minWidth: '45%',
    backgroundColor: 'white',
    borderRadius: 12,
    padding: 16,
    borderLeftWidth: 4,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 2},
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
  },
  statIcon: {
    marginBottom: 8,
  },
  statValue: {
    fontSize: 32,
    fontWeight: 'bold',
    color: '#111827',
    marginBottom: 4,
  },
  statTitle: {
    fontSize: 14,
    color: '#6b7280',
  },
  section: {
    padding: 16,
  },
  sectionTitle: {
    fontSize: 20,
    fontWeight: '600',
    color: '#111827',
    marginBottom: 16,
  },
  actionsGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 12,
  },
  actionButton: {
    flex: 1,
    minWidth: '45%',
    backgroundColor: 'white',
    borderRadius: 12,
    padding: 20,
    alignItems: 'center',
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 2},
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
  },
  actionTitle: {
    marginTop: 8,
    fontSize: 14,
    fontWeight: '500',
    color: '#374151',
  },
  backupCard: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: 'white',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 2},
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
  },
  backupIcon: {
    width: 48,
    height: 48,
    borderRadius: 24,
    backgroundColor: '#dbeafe',
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  backupInfo: {
    flex: 1,
  },
  backupName: {
    fontSize: 16,
    fontWeight: '600',
    color: '#111827',
    marginBottom: 4,
  },
  backupTime: {
    fontSize: 13,
    color: '#6b7280',
  },
  badge: {
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 12,
  },
  badgeText: {
    fontSize: 12,
    fontWeight: '600',
  },
  emptyState: {
    alignItems: 'center',
    padding: 32,
  },
  emptyText: {
    marginTop: 16,
    fontSize: 16,
    color: '#6b7280',
  },
});
